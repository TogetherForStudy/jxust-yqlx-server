package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/config"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/apperr"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/cache"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"
	"github.com/mark3labs/mcp-go/mcp"

	einoopenai "github.com/cloudwego/eino-ext/components/model/openai"
	einomcp "github.com/cloudwego/eino-ext/components/tool/mcp"
	einomodel "github.com/cloudwego/eino/components/model"
	einotool "github.com/cloudwego/eino/components/tool"
	einotoolutils "github.com/cloudwego/eino/components/tool/utils"

	json "github.com/bytedance/sonic"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type mcpClient map[string]*client.Client

func (m mcpClient) Close(ctx context.Context) {
	for name, cli := range m {
		if cli == nil {
			continue
		}
		if err := cli.Close(); err != nil {
			logger.WarnCtx(ctx, map[string]any{
				"action": "close_mcp_client",
				"name":   name,
				"error":  err.Error(),
			})
		}
	}
}

// redisCheckPointStore 使用 Redis 实现的 CheckPointStore，用于 Agent 中断恢复
type redisCheckPointStore struct {
	cache      cache.Cache
	expiration time.Duration
}

func newRedisCheckPointStore() compose.CheckPointStore {
	return &redisCheckPointStore{
		cache:      cache.GlobalCache,
		expiration: 2 * time.Hour,
	}
}

func (r *redisCheckPointStore) Get(ctx context.Context, checkPointID string) ([]byte, bool, error) {
	key := fmt.Sprintf(constant.CacheKeyAgentCheckpoint, checkPointID)
	data, err := r.cache.Get(ctx, key)
	if err != nil {
		// Key not found is not an error, just return false
		return nil, false, nil
	}
	if data == "" {
		return nil, false, nil
	}
	return []byte(data), true, nil
}

func (r *redisCheckPointStore) Set(ctx context.Context, checkPointID string, checkPoint []byte) error {
	key := fmt.Sprintf(constant.CacheKeyAgentCheckpoint, checkPointID)
	return r.cache.Set(ctx, key, string(checkPoint), &r.expiration)
}

type ChatService struct {
	httpClient      *http.Client // 复用的 HTTP 客户端,用于 MCP 等外部请求,减少GC压力和连接创建开销
	db              *gorm.DB
	cfg             *config.Config
	llm             einomodel.ToolCallingChatModel
	checkPointStore compose.CheckPointStore
}

func NewChatService(db *gorm.DB, cfg *config.Config) *ChatService {
	return &ChatService{
		db:              db,
		cfg:             cfg,
		httpClient:      &http.Client{Timeout: 30 * time.Second}, // todo: 全局 http client 可以考虑放到更上层统一管理
		checkPointStore: newRedisCheckPointStore(),
	}
}

// initRAGFlowMCP 初始化 RAGFlow MCP 客户端
func (s *ChatService) initRAGFlowMCP(ctx context.Context, userID, conversationID uint) (*client.Client, error) {
	if s.cfg.LLM.RAGFlowMCPURL == "" {
		return nil, nil
	}

	ragflowURL, err := ragflowMCPURLWithSession(s.cfg.LLM.RAGFlowMCPURL, fmt.Sprintf("u%d-c%d", userID, conversationID))
	if err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("invalid ragflow mcp url: %w", err))
	}

	headers := ragflowMCPHeaders(s.cfg.LLM.RAGFlowAPIKey)
	mcpClient, err := client.NewStreamableHttpClient(ragflowURL,
		transport.WithHTTPHeaders(headers),
		transport.WithHTTPTimeout(30*time.Second),
		transport.WithHTTPLogger(logger.L()),
		transport.WithHTTPBasicClient(s.httpClient),
	)
	if err != nil {
		logger.ErrorCtx(ctx, map[string]any{
			"action": "init_ragflow_mcp_client",
			"stage":  "new_client",
			"error":  err.Error(),
			"url":    ragflowURL,
		})
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("failed to create ragflow mcp client: %w", err))
	}
	err = mcpClient.Start(ctx)
	if err != nil {
		_ = mcpClient.Close()
		logger.ErrorCtx(ctx, map[string]any{
			"action": "init_ragflow_mcp_client",
			"stage":  "start_client",
			"error":  err.Error(),
		})
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("failed to start ragflow mcp client: %w", err))
	}
	_, err = mcpClient.Initialize(ctx, mcp.InitializeRequest{})
	if err != nil {
		_ = mcpClient.Close()
		logger.ErrorCtx(ctx, map[string]any{
			"action": "init_ragflow_mcp_client",
			"stage":  "initialize",
			"error":  err.Error(),
		})
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("failed to initialize ragflow mcp client: %w", err))
	}
	return mcpClient, nil
}

func ragflowMCPURLWithSession(rawURL, sessionID string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	query := u.Query()
	query.Set("session_id", sessionID)
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func ragflowMCPHeaders(apiKey string) map[string]string {
	headers := make(map[string]string)
	rawToken, bearerToken := normalizeBearerToken(apiKey)
	if rawToken == "" {
		return headers
	}
	headers["api_key"] = rawToken
	headers["Authorization"] = bearerToken
	return headers
}

func normalizeBearerToken(apiKey string) (string, string) {
	token := strings.TrimSpace(apiKey)
	if token == "" {
		return "", ""
	}
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = strings.TrimSpace(token[len("bearer "):])
	}
	if token == "" {
		return "", ""
	}
	return token, "Bearer " + token
}

// CreateConversation 创建新对话
func (s *ChatService) CreateConversation(ctx context.Context, userID uint, title string) (*models.Conversation, error) {
	conv := &models.Conversation{
		UserID: userID,
		Title:  title,
	}

	if err := s.db.WithContext(ctx).Create(conv).Error; err != nil {
		logger.ErrorCtx(ctx, map[string]any{
			"action":  "create_conversation",
			"user_id": userID,
			"error":   err.Error(),
		})
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("failed to create conversation: %w", err))
	}

	return conv, nil
}

// ListConversations 列出用户的对话
func (s *ChatService) ListConversations(ctx context.Context, userID uint, page, pageSize int) ([]models.Conversation, int64, error) {
	var conversations []models.Conversation
	var total int64

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	if err := s.db.WithContext(ctx).Model(&models.Conversation{}).
		Where("user_id = ?", userID).
		Count(&total).Error; err != nil {
		logger.ErrorCtx(ctx, map[string]any{
			"action":  "list_conversations_total",
			"user_id": userID,
			"error":   err.Error(),
		})
		return nil, 0, apperr.Wrap(constant.CommonInternal, fmt.Errorf("failed to count conversations: %w", err))
	}

	if err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&conversations).Error; err != nil {
		logger.ErrorCtx(ctx, map[string]any{
			"action":  "list_conversations",
			"user_id": userID,
			"error":   err.Error(),
		})
		return nil, 0, apperr.Wrap(constant.CommonInternal, fmt.Errorf("failed to list conversations: %w", err))
	}

	return conversations, total, nil
}

// getOwnedConversation returns the conversation if it belongs to the user, with a small cache to reduce DB hits.
func (s *ChatService) getOwnedConversation(ctx context.Context, userID, conversationID uint) (*models.Conversation, error) {
	cacheKey := fmt.Sprintf(constant.CacheKeyConversationInfo, userID, conversationID)

	if cached, err := cache.GlobalCache.Get(ctx, cacheKey); err == nil && cached != "" {
		var conv models.Conversation
		if err := json.Unmarshal([]byte(cached), &conv); err == nil {
			return &conv, nil
		}
		logger.WarnCtx(ctx, map[string]any{
			"action":          "conversation_cache_unmarshal",
			"user_id":         userID,
			"conversation_id": conversationID,
			"error":           err.Error(),
		})
		_ = cache.GlobalCache.Delete(ctx, cacheKey)
	}

	var conv models.Conversation
	if err := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", conversationID, userID).
		First(&conv).Error; err != nil {
		logger.ErrorCtx(ctx, map[string]any{
			"action":          "get_conversation",
			"user_id":         userID,
			"conversation_id": conversationID,
			"error":           err.Error(),
		})
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.New(constant.ConversationNotFound)
		}
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("failed to get conversation: %w", err))
	}

	if data, err := json.Marshal(&conv); err == nil {
		expiration := 10 * time.Minute
		if err := cache.GlobalCache.Set(ctx, cacheKey, string(data), &expiration); err != nil {
			logger.WarnCtx(ctx, map[string]any{
				"action":          "set_conversation_cache",
				"user_id":         userID,
				"conversation_id": conversationID,
				"error":           err.Error(),
			})
		}
	}

	return &conv, nil
}

func (s *ChatService) deleteConversationInfoCache(ctx context.Context, userID, conversationID uint) {
	_ = cache.GlobalCache.Delete(ctx, fmt.Sprintf(constant.CacheKeyConversationInfo, userID, conversationID))
}

func (s *ChatService) deleteConversationAllCaches(ctx context.Context, userID, conversationID uint) {
	_ = cache.GlobalCache.Delete(ctx, fmt.Sprintf(constant.CacheKeyConversationInfo, userID, conversationID))
}

// GetConversation 获取对话详情
func (s *ChatService) GetConversation(ctx context.Context, userID, conversationID uint) (*models.Conversation, error) {
	return s.getOwnedConversation(ctx, userID, conversationID)
}

// DeleteConversation 删除对话
func (s *ChatService) DeleteConversation(ctx context.Context, userID, conversationID uint) error {
	result := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", conversationID, userID).
		Delete(&models.Conversation{})

	if result.Error != nil {
		logger.ErrorCtx(ctx, map[string]any{
			"action":          "delete_conversation",
			"user_id":         userID,
			"conversation_id": conversationID,
			"error":           result.Error.Error(),
		})
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("failed to delete conversation: %w", result.Error))
	}

	if result.RowsAffected == 0 {

		logger.WarnCtx(ctx, map[string]any{
			"action":          "delete_conversation_not_found",
			"user_id":         userID,
			"conversation_id": conversationID,
		})
		return apperr.New(constant.ConversationNotFound)
	}

	s.deleteConversationAllCaches(ctx, userID, conversationID)
	return nil
}

// UpdateConversation 更新对话标题
func (s *ChatService) UpdateConversation(ctx context.Context, userID, conversationID uint, title string) error {
	result := s.db.WithContext(ctx).
		Model(&models.Conversation{}).
		Where("id = ? AND user_id = ?", conversationID, userID).
		Update("title", title)

	if result.Error != nil {
		logger.ErrorCtx(ctx, map[string]any{
			"action":          "update_conversation",
			"user_id":         userID,
			"conversation_id": conversationID,
			"error":           result.Error.Error(),
		})
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("failed to update conversation: %w", result.Error))
	}

	if result.RowsAffected == 0 {
		//logger.Warnf("RequestID[%s]: Conversation not found for update: conversationID=%d, userID=%d", utils.GetRequestID(ctx), conversationID, userID)
		logger.WarnCtx(ctx, map[string]any{
			"action":          "update_conversation_not_found",
			"user_id":         userID,
			"conversation_id": conversationID,
		})
		return apperr.New(constant.ConversationNotFound)
	}

	s.deleteConversationInfoCache(ctx, userID, conversationID)
	return nil
}

func schemaMessageToConversationMessage(userID, conversationID uint, checkpointID string, msg *schema.Message) (*models.ConversationMessage, error) {
	rawMessage, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	var toolCalls []byte
	if len(msg.ToolCalls) > 0 {
		toolCalls, err = json.Marshal(msg.ToolCalls)
		if err != nil {
			return nil, err
		}
	}

	return &models.ConversationMessage{
		ConversationID:   conversationID,
		UserID:           userID,
		CheckpointID:     checkpointID,
		Role:             string(msg.Role),
		Content:          msg.Content,
		ReasoningContent: msg.ReasoningContent,
		ToolCallID:       msg.ToolCallID,
		ToolName:         msg.ToolName,
		ToolCalls:        datatypes.JSON(toolCalls),
		RawMessage:       datatypes.JSON(rawMessage),
	}, nil
}

func conversationMessageToSchemaMessage(row *models.ConversationMessage) (*schema.Message, error) {
	if len(row.RawMessage) > 0 {
		var msg schema.Message
		if err := json.Unmarshal(row.RawMessage, &msg); err == nil {
			return &msg, nil
		}
	}

	msg := &schema.Message{
		Role:             schema.RoleType(row.Role),
		Content:          row.Content,
		ReasoningContent: row.ReasoningContent,
		ToolCallID:       row.ToolCallID,
		ToolName:         row.ToolName,
	}
	if len(row.ToolCalls) > 0 {
		if err := json.Unmarshal(row.ToolCalls, &msg.ToolCalls); err != nil {
			return nil, err
		}
	}
	return msg, nil
}

// GetMessages 获取对话的所有消息
func (s *ChatService) GetMessages(ctx context.Context, userID, conversationID uint) ([]*schema.Message, error) {
	// 验证对话属于用户（带缓存）
	if _, err := s.getOwnedConversation(ctx, userID, conversationID); err != nil {
		return nil, err
	}

	var rows []models.ConversationMessage
	if err := s.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Order("created_at ASC, id ASC").
		Find(&rows).Error; err != nil {
		logger.ErrorCtx(ctx, map[string]any{
			"action":          "get_messages",
			"user_id":         userID,
			"conversation_id": conversationID,
			"error":           err.Error(),
		})
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("failed to get messages: %w", err))
	}

	messages := make([]*schema.Message, 0, len(rows))
	for i := range rows {
		msg, err := conversationMessageToSchemaMessage(&rows[i])
		if err != nil {
			logger.ErrorCtx(ctx, map[string]any{
				"action":          "get_messages_unmarshal",
				"user_id":         userID,
				"conversation_id": conversationID,
				"message_id":      rows[i].ID,
				"error":           err.Error(),
			})
			return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("failed to unmarshal message: %w", err))
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// AppendMessages 追加消息到对话消息表
func (s *ChatService) AppendMessages(ctx context.Context, userID, conversationID uint, checkpointID string, messages []*schema.Message) error {
	records := make([]models.ConversationMessage, 0, len(messages))
	for _, msg := range messages {
		if msg == nil {
			continue
		}
		record, err := schemaMessageToConversationMessage(userID, conversationID, checkpointID, msg)
		if err != nil {
			logger.ErrorCtx(ctx, map[string]any{
				"action":          "append_messages_marshal",
				"user_id":         userID,
				"conversation_id": conversationID,
				"error":           err.Error(),
			})
			return apperr.Wrap(constant.CommonInternal, fmt.Errorf("failed to marshal message: %w", err))
		}
		records = append(records, *record)
	}
	if len(records) == 0 {
		return nil
	}

	now := time.Now()
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var conv models.Conversation
		if err := tx.Select("id").
			Where("id = ? AND user_id = ?", conversationID, userID).
			First(&conv).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return apperr.New(constant.ConversationNotFound)
			}
			logger.ErrorCtx(ctx, map[string]any{
				"action":          "append_messages_get_conversation",
				"user_id":         userID,
				"conversation_id": conversationID,
				"error":           err.Error(),
			})
			return apperr.Wrap(constant.CommonInternal, fmt.Errorf("failed to get conversation: %w", err))
		}

		if err := tx.Create(&records).Error; err != nil {
			logger.ErrorCtx(ctx, map[string]any{
				"action":          "append_messages_create",
				"user_id":         userID,
				"conversation_id": conversationID,
				"error":           err.Error(),
			})
			return apperr.Wrap(constant.CommonInternal, fmt.Errorf("failed to append messages: %w", err))
		}

		result := tx.Model(&models.Conversation{}).
			Where("id = ? AND user_id = ?", conversationID, userID).
			Updates(map[string]interface{}{
				"last_message_at": now,
				"updated_at":      now,
			})
		if result.Error != nil {
			logger.ErrorCtx(ctx, map[string]any{
				"action":          "append_messages_update_conversation",
				"user_id":         userID,
				"conversation_id": conversationID,
				"error":           result.Error.Error(),
			})
			return apperr.Wrap(constant.CommonInternal, fmt.Errorf("failed to update conversation: %w", result.Error))
		}
		if result.RowsAffected == 0 {
			return apperr.New(constant.ConversationNotFound)
		}
		return nil
	})
	if err != nil {
		return err
	}

	s.deleteConversationInfoCache(ctx, userID, conversationID)
	return nil
}

func (s *ChatService) initYQLXMCP(ctx context.Context, userID uint, userToken string) (*client.Client, error) {
	if userToken == "" {
		return nil, fmt.Errorf("missing user token")
	}

	yqlxMcpClient, err := client.NewStreamableHttpClient(fmt.Sprintf("%s://%s/api/mcp", s.cfg.Scheme, s.cfg.Host), // yqlx自身的 MCP 转发接口 //todo:使用github.com/mark3labs/mcp-go/mcp重构后直接走api调用，不过网络栈
		transport.WithHTTPHeaders(
			map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", userToken),
			}),
		transport.WithHTTPTimeout(30*time.Second),
		transport.WithHTTPLogger(logger.L()),
		transport.WithHTTPBasicClient(s.httpClient))
	if err != nil {
		msg := fmt.Sprintf("failed to create user mcp client: %v", err)
		logger.ErrorCtx(ctx, map[string]any{
			"action":  "prepare_user_mcp_client",
			"stage":   "create_client",
			"user_id": userID,
			"error":   err.Error(),
		})
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("%s", msg))
	}
	if err := yqlxMcpClient.Start(ctx); err != nil {
		_ = yqlxMcpClient.Close()
		logger.ErrorCtx(ctx, map[string]any{
			"action":  "prepare_user_mcp_client",
			"stage":   "start_yqlx_mcp_client",
			"user_id": userID,
			"error":   err.Error(),
		})
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("failed to start user mcp client: %w", err))
	}
	_, err = yqlxMcpClient.Initialize(ctx, mcp.InitializeRequest{})
	if err != nil {
		_ = yqlxMcpClient.Close()
		logger.ErrorCtx(ctx, map[string]any{
			"action":  "prepare_user_mcp_client",
			"stage":   "init_yqlx_mcp_client",
			"user_id": userID,
			"error":   err.Error(),
		})
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("failed to initialize user mcp client: %w", err))
	}

	return yqlxMcpClient, nil
}

// todo: mcp server 除了系统内置的 mcp client 之外，还可以支持用户自定义的 mcp client
func (s *ChatService) prepareUserMcpClient(ctx context.Context, userID, conversationID uint, userToken string) mcpClient {
	clients := make(mcpClient)

	yqlxMcpClient, err := s.initYQLXMCP(ctx, userID, userToken)
	if err != nil {
		logger.WarnCtx(ctx, map[string]any{
			"action":  "prepare_user_mcp_client",
			"stage":   "init_yqlx_mcp_client",
			"user_id": userID,
			"error":   err.Error(),
		})
	} else if yqlxMcpClient != nil {
		clients["yqlx"] = yqlxMcpClient
	}

	ragMcpClient, err := s.initRAGFlowMCP(ctx, userID, conversationID)
	if err != nil {
		logger.WarnCtx(ctx, map[string]any{
			"action":  "prepare_user_mcp_client",
			"stage":   "init_ragflow_mcp_client",
			"user_id": userID,
			"error":   err.Error(),
		})
	} else if ragMcpClient != nil {
		clients["ragflow"] = ragMcpClient
	}

	return clients
}

func (s *ChatService) loadMCPTools(ctx context.Context, userID, conversationID uint, userToken string) ([]einotool.BaseTool, mcpClient) {
	clients := s.prepareUserMcpClient(ctx, userID, conversationID, userToken)
	allTools := make([]einotool.BaseTool, 0)
	for name, cli := range clients {
		if cli == nil {
			continue
		}
		tools, err := einomcp.GetTools(ctx, &einomcp.Config{Cli: cli})
		if err != nil {
			logger.WarnCtx(ctx, map[string]any{
				"action":  "stream_chat_get_mcp_tools",
				"user_id": userID,
				"name":    name,
				"msg":     "Failed to load MCP tools, skipping",
				"error":   err.Error(),
			})
			continue
		}
		tools = wrapAgentToolsWithFailureResults(ctx, tools, name, userID, conversationID)
		allTools = append(allTools, tools...)
	}

	logger.InfoCtx(ctx, map[string]any{
		"action":          "stream_chat_loaded_mcp_tools",
		"user_id":         userID,
		"conversation_id": conversationID,
		"tool_count":      len(allTools),
		"client_count":    len(clients),
	})

	return allTools, clients
}

func wrapAgentToolsWithFailureResults(ctx context.Context, tools []einotool.BaseTool, source string, userID, conversationID uint) []einotool.BaseTool {
	if len(tools) == 0 {
		return tools
	}

	wrappedTools := make([]einotool.BaseTool, 0, len(tools))
	for _, t := range tools {
		if t == nil {
			continue
		}

		toolName := "unknown"
		if info, err := t.Info(ctx); err != nil {
			logger.WarnCtx(ctx, map[string]any{
				"action":          "agent_tool_info_before_wrap",
				"user_id":         userID,
				"conversation_id": conversationID,
				"source":          source,
				"error":           err.Error(),
			})
		} else if info != nil && info.Name != "" {
			toolName = info.Name
		}

		nameForHandler := toolName
		wrappedTools = append(wrappedTools, einotoolutils.WrapToolWithErrorHandler(t, func(ctx context.Context, err error) string {
			logger.WarnCtx(ctx, map[string]any{
				"action":          "agent_tool_failed_as_result",
				"user_id":         userID,
				"conversation_id": conversationID,
				"source":          source,
				"tool_name":       nameForHandler,
				"error":           err.Error(),
			})
			return agentToolFailureResult(nameForHandler, err.Error())
		}))
	}

	return wrappedTools
}

func agentToolFailureResult(toolName, errMsg string) string {
	payload := map[string]any{
		"success":   false,
		"tool_name": toolName,
		"error":     errMsg,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return `{"success":false,"error":"tool call failed"}`
	}
	return string(data)
}

func unknownAgentToolHandler(ctx context.Context, name, input string) (string, error) {
	logger.WarnCtx(ctx, map[string]any{
		"action":    "agent_unknown_tool_as_result",
		"tool_name": name,
		"arguments": input,
	})
	return agentToolFailureResult(name, fmt.Sprintf("tool %s not found", name)), nil
}

// streamEventProcessor 处理 Agent 事件流并输出到通道
// 这是 StreamChat 和 ResumeChat 共用的核心逻辑
type streamEventProcessor struct {
	ctx            context.Context
	userID         uint
	conversationID uint
	checkpointID   string
	conv           *models.Conversation
	service        *ChatService
	mcpClients     mcpClient
	outputChan     chan string
	errChan        chan error
	startEventType string // "start" 或 "resume_start"
}

func (p *streamEventProcessor) process(iter *adk.AsyncIterator[*adk.AgentEvent]) {
	defer close(p.outputChan)
	defer close(p.errChan)
	defer p.mcpClients.Close(p.ctx)

	var usage *schema.TokenUsage
	var turnMessages []*schema.Message
	messageCount := 0

	startEvent := map[string]interface{}{
		"type":          p.startEventType,
		"checkpoint_id": p.checkpointID,
	}
	if data, err := json.Marshal(startEvent); err == nil {
		p.outputChan <- "data: " + string(data) + "\n\n"
	}

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			logger.ErrorCtx(p.ctx, map[string]any{
				"action":          "agent_stream_event",
				"user_id":         p.userID,
				"conversation_id": p.conversationID,
				"checkpoint_id":   p.checkpointID,
				"error":           event.Err.Error(),
			})
			p.saveMessages(turnMessages)
			p.errChan <- event.Err
			return
		}

		if event.Action != nil && event.Action.Interrupted != nil {
			p.saveMessages(turnMessages)

			interruptEvent := map[string]interface{}{
				"type":          "interrupt",
				"checkpoint_id": p.checkpointID,
				"data":          event.Action.Interrupted.Data,
			}
			if data, err := json.Marshal(interruptEvent); err == nil {
				p.outputChan <- "data: " + string(data) + "\n\n"
			}
			logger.InfoCtx(p.ctx, map[string]any{
				"action":          "agent_interrupted",
				"user_id":         p.userID,
				"conversation_id": p.conversationID,
				"checkpoint_id":   p.checkpointID,
			})
			return
		}

		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		msgOutput := event.Output.MessageOutput
		messageCount++

		if msgOutput.IsStreaming {
			isToolStream := msgOutput.Role == schema.Tool
			var chunks []*schema.Message
			for {
				recv, err := msgOutput.MessageStream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					logger.ErrorCtx(p.ctx, map[string]any{
						"action":          "agent_stream_recv",
						"user_id":         p.userID,
						"conversation_id": p.conversationID,
						"checkpoint_id":   p.checkpointID,
						"error":           err.Error(),
					})
					p.saveMessages(turnMessages)
					p.errChan <- err
					return
				}

				chunks = append(chunks, recv)
				if !isToolStream {
					p.emitStreamingChunk(recv)
				}
				if recv.ResponseMeta != nil && recv.ResponseMeta.Usage != nil {
					usage = recv.ResponseMeta.Usage
				}
			}

			if len(chunks) > 0 {
				msg, err := schema.ConcatMessages(chunks)
				if err != nil {
					logger.WarnCtx(p.ctx, map[string]any{
						"action":          "agent_stream_concat_message",
						"user_id":         p.userID,
						"conversation_id": p.conversationID,
						"checkpoint_id":   p.checkpointID,
						"error":           err.Error(),
					})
				} else {
					if msg.Role == "" {
						msg.Role = msgOutput.Role
					}
					if msg.ToolName == "" {
						msg.ToolName = msgOutput.ToolName
					}
					if msg.Role == schema.Tool {
						p.emitToolResult(msg)
					}
					turnMessages = appendAgentMessage(turnMessages, msg)
				}
			}
			continue
		}

		msg, err := msgOutput.GetMessage()
		if err != nil {
			logger.ErrorCtx(p.ctx, map[string]any{
				"action":          "agent_get_message",
				"user_id":         p.userID,
				"conversation_id": p.conversationID,
				"checkpoint_id":   p.checkpointID,
				"error":           err.Error(),
			})
			continue
		}
		if msg.Role == "" {
			msg.Role = msgOutput.Role
		}
		if msg.ToolName == "" {
			msg.ToolName = msgOutput.ToolName
		}
		if msg.Role == schema.Tool {
			p.emitToolResult(msg)
		}
		turnMessages = appendAgentMessage(turnMessages, msg)
	}

	logger.InfoCtx(p.ctx, map[string]any{
		"action":          "agent_stream_finished",
		"user_id":         p.userID,
		"conversation_id": p.conversationID,
		"checkpoint_id":   p.checkpointID,
		"message_count":   messageCount,
	})

	// 保存本轮 Agent 产生的消息
	p.saveMessages(turnMessages)

	// 发送结束事件
	endEvent := map[string]interface{}{
		"type":          "end",
		"message_count": messageCount,
	}
	if usage != nil {
		endEvent["usage"] = map[string]interface{}{
			"prompt_tokens":     usage.PromptTokens,
			"completion_tokens": usage.CompletionTokens,
			"total_tokens":      usage.TotalTokens,
		}
	}
	if data, err := json.Marshal(endEvent); err == nil {
		p.outputChan <- "data: " + string(data) + "\n\n"
	}
}

func (p *streamEventProcessor) emitStreamingChunk(recv *schema.Message) {
	if recv.Content != "" {
		contentEvent := map[string]interface{}{
			"type":    "content",
			"content": recv.Content,
		}
		if data, err := json.Marshal(contentEvent); err == nil {
			p.outputChan <- "data: " + string(data) + "\n\n"
		}
	}

	if recv.ReasoningContent != "" {
		reasoningEvent := map[string]interface{}{
			"type":    "reasoning",
			"content": recv.ReasoningContent,
		}
		if data, err := json.Marshal(reasoningEvent); err == nil {
			p.outputChan <- "data: " + string(data) + "\n\n"
		}
	}

	for _, tc := range recv.ToolCalls {
		toolCallEvent := map[string]interface{}{
			"type":      "tool_call",
			"tool_call": tc,
			"function":  tc.Function.Name,
			"arguments": tc.Function.Arguments,
		}
		if data, err := json.Marshal(toolCallEvent); err == nil {
			p.outputChan <- "data: " + string(data) + "\n\n"
		}
	}
}

func (p *streamEventProcessor) emitToolResult(msg *schema.Message) {
	toolResultEvent := map[string]interface{}{
		"type":         "tool_result",
		"tool_call_id": msg.ToolCallID,
		"tool_name":    msg.ToolName,
		"content":      msg.Content,
	}
	if data, err := json.Marshal(toolResultEvent); err == nil {
		p.outputChan <- "data: " + string(data) + "\n\n"
	}
}

func appendAgentMessage(messages []*schema.Message, msg *schema.Message) []*schema.Message {
	if msg == nil {
		return messages
	}
	if msg.Role == schema.Assistant || msg.Role == schema.Tool {
		return append(messages, msg)
	}
	return messages
}

func (p *streamEventProcessor) saveMessages(messages []*schema.Message) {
	if len(messages) == 0 || p.service == nil || p.conv == nil {
		return
	}
	if err := p.service.AppendMessages(context.WithoutCancel(p.ctx), p.userID, p.conv.ID, p.checkpointID, messages); err != nil {
		logger.ErrorCtx(p.ctx, map[string]any{
			"action":          "agent_save_messages",
			"user_id":         p.userID,
			"conversation_id": p.conversationID,
			"checkpoint_id":   p.checkpointID,
			"error":           err.Error(),
		})
	}
}

// createChatModel 创建或获取 ChatModel
func (s *ChatService) createChatModel(ctx context.Context) (einomodel.ToolCallingChatModel, error) {
	if s.llm != nil {
		chatModel := s.llm
		return chatModel, nil
	}

	// 如果没有全局 LLM，为这个请求创建临时实例
	model, err := einoopenai.NewChatModel(ctx, &einoopenai.ChatModelConfig{
		HTTPClient: s.httpClient,
		Model:      s.cfg.LLM.Model,
		APIKey:     s.cfg.LLM.APIKey,
		BaseURL:    s.cfg.LLM.BaseURL,
	})
	if err != nil {
		logger.ErrorCtx(ctx, map[string]any{
			"action": "create_chat_model",
			"error":  err.Error(),
		})
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("failed to create chat model: %w", err))
	}
	return model, nil
}

// createAgent 创建 ChatModelAgent
func (s *ChatService) createAgent(ctx context.Context, tools []einotool.BaseTool) (adk.Agent, error) {
	chatModel, err := s.createChatModel(ctx)
	if err != nil {
		return nil, err
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "StudyAssistant",
		Description: "A helpful assistant for students at JXUST.",
		Instruction: constant.ChatSystemPrompt,
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools:               tools,
				UnknownToolsHandler: unknownAgentToolHandler,
			},
		},
		MaxIterations: 15, //todo: 配置化
	})
	if err != nil {
		logger.ErrorCtx(ctx, map[string]any{
			"action": "create_agent",
			"error":  err.Error(),
		})
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("failed to create agent: %w", err))
	}
	return agent, nil
}

// StreamChat 流式聊天 (返回一个通道用于 SSE)
// 使用 NewChatModelAgent 实现，支持中断恢复和流式传输
// 当 checkpointID 不为空时，执行恢复操作；否则执行新对话
func (s *ChatService) StreamChat(ctx context.Context, userID, conversationID uint, newMessage *schema.Message, userToken string, checkpointID string, resumeInput string) (<-chan string, <-chan error, error) {
	// 验证对话属于用户
	conv, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		logger.ErrorCtx(ctx, map[string]any{
			"action":          "stream_chat_get_conversation",
			"user_id":         userID,
			"conversation_id": conversationID,
			"error":           err.Error(),
		})
		return nil, nil, err
	}

	// 获取完整的会话消息
	messages, err := s.GetMessages(ctx, userID, conversationID)
	if err != nil {
		logger.ErrorCtx(ctx, map[string]any{
			"action":          "stream_chat_get_messages",
			"user_id":         userID,
			"conversation_id": conversationID,
			"error":           err.Error(),
		})
		return nil, nil, err
	}

	isResume := checkpointID != ""

	// 新对话：合并新消息
	if !isResume {
		if newMessage == nil {
			return nil, nil, apperr.New(constant.ConversationMessageRequired)
		}
		checkpointID = fmt.Sprintf("%d:%d:%d", userID, conversationID, time.Now().UnixNano())
		if err := s.AppendMessages(ctx, userID, conversationID, checkpointID, []*schema.Message{newMessage}); err != nil {
			logger.ErrorCtx(ctx, map[string]any{
				"action":          "stream_chat_append_user_message",
				"user_id":         userID,
				"conversation_id": conversationID,
				"checkpoint_id":   checkpointID,
				"error":           err.Error(),
			})
			return nil, nil, err
		}
		messages = append(messages, newMessage)
	}

	// MCP 工具为增强能力，不应阻塞基础聊天能力。resume 也需要加载工具以恢复中断点。
	allTools, mcpClients := s.loadMCPTools(ctx, userID, conversationID, userToken)

	// 创建 Agent
	agent, err := s.createAgent(ctx, allTools)
	if err != nil {
		mcpClients.Close(ctx)
		logger.ErrorCtx(ctx, map[string]any{
			"action":          "stream_chat_create_agent",
			"user_id":         userID,
			"conversation_id": conversationID,
			"error":           err.Error(),
		})
		return nil, nil, err
	}

	// 创建 Runner（支持 checkpoint）
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
		CheckPointStore: s.checkPointStore,
	})

	var iter *adk.AsyncIterator[*adk.AgentEvent]
	var startEventType string

	if isResume {
		// 恢复运行
		var toolOpts []einotool.Option
		if resumeInput != "" {
			toolOpts = append(toolOpts, WithResumeInput(resumeInput))
		}
		iter, err = runner.Resume(ctx, checkpointID, adk.WithToolOptions(toolOpts))
		if err != nil {
			mcpClients.Close(ctx)
			logger.ErrorCtx(ctx, map[string]any{
				"action":          "stream_chat_resume_agent",
				"user_id":         userID,
				"conversation_id": conversationID,
				"checkpoint_id":   checkpointID,
				"error":           err.Error(),
			})
			return nil, nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("failed to resume agent: %w", err))
		}
		startEventType = "resume_start"
		logger.InfoCtx(ctx, map[string]any{
			"action":          "stream_chat_resuming",
			"user_id":         userID,
			"conversation_id": conversationID,
			"checkpoint_id":   checkpointID,
		})
	} else {
		// 构建 Agent 输入消息（不包含系统提示词，由 Agent 的 Instruction 处理）
		var inputMessages []adk.Message
		for _, msg := range messages {
			if msg.Role != schema.System {
				inputMessages = append(inputMessages, msg)
			}
		}

		// 运行 Agent
		iter = runner.Run(ctx, inputMessages, adk.WithCheckPointID(checkpointID))
		startEventType = "start"
	}

	// 创建输出通道
	outputChan := make(chan string, 50)
	errChan := make(chan error, 1)

	// 创建事件处理器并启动
	processor := &streamEventProcessor{
		ctx:            ctx,
		userID:         userID,
		conversationID: conversationID,
		checkpointID:   checkpointID,
		conv:           conv,
		service:        s,
		mcpClients:     mcpClients,
		outputChan:     outputChan,
		errChan:        errChan,
		startEventType: startEventType,
	}

	go processor.process(iter)

	return outputChan, errChan, nil
}

// resumeInputOption 用于传递恢复时的用户输入
type resumeInputOption struct {
	input string
}

// WithResumeInput 创建一个包含恢复输入的工具选项
func WithResumeInput(input string) einotool.Option {
	return einotool.WrapImplSpecificOptFn(func(o *resumeInputOption) {
		o.input = input
	})
}

// GenerateTitleFromFirstMessage 从第一条消息生成标题
func (s *ChatService) GenerateTitleFromFirstMessage(ctx context.Context, content string) (string, error) {
	// TODO: 使用 LLM 生成标题
	// 临时实现：截取前30个字符
	if len(content) > 30 {
		return content[:30] + "...", nil
	}
	return content, nil
}

// ExportConversation 导出对话
func (s *ChatService) ExportConversation(ctx context.Context, userID, conversationID uint) (*models.Conversation, []*schema.Message, error) {
	conv, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return nil, nil, err
	}

	messages, err := s.GetMessages(ctx, userID, conversationID)
	if err != nil {
		return nil, nil, err
	}

	return conv, messages, nil
}
