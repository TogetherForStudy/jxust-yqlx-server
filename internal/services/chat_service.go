package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/config"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/cache"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"
	"github.com/mark3labs/mcp-go/mcp"

	einoopenai "github.com/cloudwego/eino-ext/components/model/openai"
	einomcp "github.com/cloudwego/eino-ext/components/tool/mcp"
	einomodel "github.com/cloudwego/eino/components/model"
	einotool "github.com/cloudwego/eino/components/tool"

	json "github.com/bytedance/sonic"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"gorm.io/gorm"
)

type mcpClient map[string]*client.Client

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

	//mcpClients    mcpClient          // global
	userClients   map[uint]mcpClient // per-user MCP 客户端集合 key: userID, value: mcpClient
	userClientsMu sync.RWMutex
}

func NewChatService(db *gorm.DB, cfg *config.Config) *ChatService {
	return &ChatService{
		db:              db,
		cfg:             cfg,
		httpClient:      &http.Client{Timeout: 30 * time.Second}, // todo: 全局 http client 可以考虑放到更上层统一管理
		checkPointStore: newRedisCheckPointStore(),
		userClients:     make(map[uint]mcpClient),
	}
}

// initRAGFlowMCP 初始化 RAGFlow MCP 客户端
func (s *ChatService) initRAGFlowMCP(ctx context.Context) (*client.Client, error) {
	// TODO: 根据 eino 的 MCP 集成文档实现
	// 参考: https://www.cloudwego.io/docs/eino/ecosystem_integration/tool/tool_mcp/
	// todo: ragflow sse endpoint 还需要 session_id 参数
	mcpClient, err := client.NewSSEMCPClient(s.cfg.LLM.RAGFlowMCPURL, //fmt.Sprintf("%s/messages/?session_id=%s", s.cfg.LLM.RAGFlowMCPURL, sessionId),
		transport.WithHeaders(
			map[string]string{
				"api_key": s.cfg.LLM.RAGFlowAPIKey, // global api key
			}),
		// transport.WithHTTPTimeout(30*time.Second),
		transport.WithSSELogger(logger.L()),
		transport.WithHTTPClient(s.httpClient),
	)
	if err != nil {
		logger.ErrorCtx(ctx, map[string]any{
			"action": "init_ragflow_mcp_client",
			"stage":  "new_client",
			"error":  err.Error(),
			"url":    s.cfg.LLM.RAGFlowMCPURL,
		})
		return nil, fmt.Errorf("failed to create ragflow mcp client: %w", err)
	}
	err = mcpClient.Start(ctx)
	if err != nil {
		logger.ErrorCtx(ctx, map[string]any{
			"action": "init_ragflow_mcp_client",
			"stage":  "start_client",
			"error":  err.Error(),
		})
		return nil, err
	}
	_, err = mcpClient.Initialize(ctx, mcp.InitializeRequest{})
	if err != nil {
		logger.ErrorCtx(ctx, map[string]any{
			"action": "init_ragflow_mcp_client",
			"stage":  "initialize",
			"error":  err.Error(),
		})
		return nil, err
	}
	return mcpClient, nil
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
		return nil, err
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
		Where("user_id = ? AND deleted_at = null", userID).
		Count(&total).Error; err != nil {
		logger.ErrorCtx(ctx, map[string]any{
			"action":  "list_conversations_total",
			"user_id": userID,
			"error":   err.Error(),
		})
		return nil, 0, err
	}

	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND deleted_at = null", userID).
		Order("updated_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&conversations).Error; err != nil {
		logger.ErrorCtx(ctx, map[string]any{
			"action":  "list_conversations",
			"user_id": userID,
			"error":   err.Error(),
		})
		return nil, 0, err
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
		Where("id = ? AND user_id = ? AND deleted_at = null", conversationID, userID).
		First(&conv).Error; err != nil {
		logger.ErrorCtx(ctx, map[string]any{
			"action":          "get_conversation",
			"user_id":         userID,
			"conversation_id": conversationID,
			"error":           err.Error(),
		})
		return nil, err
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
	_ = cache.GlobalCache.Delete(ctx, fmt.Sprintf(constant.CacheKeyConversationMessages, userID, conversationID))
}

// GetConversation 获取对话详情
func (s *ChatService) GetConversation(ctx context.Context, userID, conversationID uint) (*models.Conversation, error) {
	return s.getOwnedConversation(ctx, userID, conversationID)
}

// DeleteConversation 删除对话
func (s *ChatService) DeleteConversation(ctx context.Context, userID, conversationID uint) error {
	result := s.db.WithContext(ctx).Model(models.Conversation{}).
		Where("id = ? AND user_id = ?", conversationID, userID).
		Update("deleted_at", time.Now())

	if result.Error != nil {
		logger.ErrorCtx(ctx, map[string]any{
			"action":          "delete_conversation",
			"user_id":         userID,
			"conversation_id": conversationID,
			"error":           result.Error.Error(),
		})
		return result.Error
	}

	if result.RowsAffected == 0 {

		logger.WarnCtx(ctx, map[string]any{
			"action":          "delete_conversation_not_found",
			"user_id":         userID,
			"conversation_id": conversationID,
		})
		return errors.New("conversation not found")
	}

	s.deleteConversationAllCaches(ctx, userID, conversationID)
	return nil
}

// UpdateConversation 更新对话标题
func (s *ChatService) UpdateConversation(ctx context.Context, userID, conversationID uint, title string) error {
	result := s.db.WithContext(ctx).
		Model(&models.Conversation{}).
		Where("id = ? AND user_id = ? AND deleted_at = null", conversationID, userID).
		Update("title", title)

	if result.Error != nil {
		logger.ErrorCtx(ctx, map[string]any{
			"action":          "update_conversation",
			"user_id":         userID,
			"conversation_id": conversationID,
			"error":           result.Error.Error(),
		})
		return result.Error
	}

	if result.RowsAffected == 0 {
		//logger.Warnf("RequestID[%s]: Conversation not found for update: conversationID=%d, userID=%d", utils.GetRequestID(ctx), conversationID, userID)
		logger.WarnCtx(ctx, map[string]any{
			"action":          "update_conversation_not_found",
			"user_id":         userID,
			"conversation_id": conversationID,
		})
		return errors.New("conversation not found")
	}

	s.deleteConversationInfoCache(ctx, userID, conversationID)
	return nil
}

// GetMessages 获取对话的所有消息（使用缓存，不存在则从数据库构建缓存）
func (s *ChatService) GetMessages(ctx context.Context, userID, conversationID uint) ([]*schema.Message, error) {
	// 验证对话属于用户（带缓存）
	conv, err := s.getOwnedConversation(ctx, userID, conversationID)
	if err != nil {
		return nil, err
	}

	// 尝试从缓存获取
	cacheKey := fmt.Sprintf(constant.CacheKeyConversationMessages, userID, conversationID)
	if cachedData, err := cache.GlobalCache.Get(ctx, cacheKey); err == nil && cachedData != "" {
		var messages []*schema.Message
		if err := json.Unmarshal([]byte(cachedData), &messages); err == nil {
			return messages, nil
		}
		logger.WarnCtx(ctx, map[string]any{
			"action":          "get_messages_cache_unmarshal",
			"user_id":         userID,
			"conversation_id": conversationID,
			"error":           err.Error(),
		})
	}

	// 从数据库加载
	messages := []*schema.Message{}
	if len(conv.Messages) > 0 {
		if err := json.Unmarshal(conv.Messages, &messages); err != nil {
			logger.ErrorCtx(ctx, map[string]any{
				"action":          "get_messages_unmarshal",
				"user_id":         userID,
				"conversation_id": conversationID,
				"error":           err.Error(),
			})
			return nil, fmt.Errorf("failed to unmarshal messages: %w", err)
		}
	}

	// 更新缓存
	if data, err := json.Marshal(messages); err == nil {
		expiration := 30 * time.Minute
		err := cache.GlobalCache.Set(ctx, cacheKey, string(data), &expiration)
		if err != nil {
			logger.ErrorCtx(ctx, map[string]any{
				"action":          "get_messages_set_cache",
				"user_id":         userID,
				"conversation_id": conversationID,
				"error":           err.Error(),
			})
			return nil, err
		}
	}

	return messages, nil
}

// SaveMessages 保存完整的消息列表到数据库和缓存
func (s *ChatService) SaveMessages(ctx context.Context, userID, conversationID uint, messages []*schema.Message) error {
	messagesJSON, err := json.Marshal(messages)
	if err != nil {
		logger.ErrorCtx(ctx, map[string]any{
			"action":          "save_messages_marshal",
			"user_id":         userID,
			"conversation_id": conversationID,
			"messages":        messages,
			"error":           err.Error(),
		})
		return fmt.Errorf("failed to marshal messages: %w", err)
	}

	// 更新数据库
	now := time.Now()
	if err := s.db.WithContext(ctx).Model(&models.Conversation{}).
		Where("id = ?", conversationID).
		Updates(map[string]interface{}{
			"messages":        messagesJSON,
			"last_message_at": now,
			"updated_at":      now,
		}).Error; err != nil {
		logger.ErrorCtx(ctx, map[string]any{
			"action":          "save_messages_update_db",
			"user_id":         userID,
			"messages":        messages,
			"conversation_id": conversationID,
			"error":           err.Error(),
		})
		return err
	}

	// 更新缓存
	cacheKey := fmt.Sprintf(constant.CacheKeyConversationMessages, userID, conversationID)
	expiration := 30 * time.Minute
	err = cache.GlobalCache.Set(ctx, cacheKey, string(messagesJSON), &expiration)
	if err != nil {
		logger.ErrorCtx(ctx, map[string]any{
			"action":          "save_messages_set_cache",
			"user_id":         userID,
			"conversation_id": conversationID,
			"messages":        messages,
			"error":           err.Error(),
		})
		return err
	}

	s.deleteConversationInfoCache(ctx, userID, conversationID)
	return nil
}

// todo: mcp server 除了系统内置的 mcp client 之外，还可以支持用户自定义的 mcp client
func (s *ChatService) prepareUserMcpClient(ctx context.Context, userID uint, userToken string) (mcpClient, error) {
	yqlxMcpClient, err := client.NewStreamableHttpClient(fmt.Sprintf("http://127.0.0.1:%s/api/mcp", s.cfg.ServerPort), // yqlx自身的 MCP 转发接口 //todo:使用github.com/mark3labs/mcp-go/mcp重构后直接走api调用，不过网络栈
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
		return nil, errors.New(msg)
	}
	if err := yqlxMcpClient.Start(ctx); err != nil {
		logger.ErrorCtx(ctx, map[string]any{
			"action":  "prepare_user_mcp_client",
			"stage":   "start_yqlx_mcp_client",
			"user_id": userID,
			"error":   err.Error(),
		})
		return nil, err
	}
	_, err = yqlxMcpClient.Initialize(ctx, mcp.InitializeRequest{})
	if err != nil {
		logger.ErrorCtx(ctx, map[string]any{
			"action":  "prepare_user_mcp_client",
			"stage":   "init_yqlx_mcp_client",
			"user_id": userID,
			"error":   err.Error(),
		})
		return nil, err
	}
	// 初始化 RAGFlow MCP 工具
	// todo: sessionId 应该是每个用户唯一的，可以用 userID 或者其他方式生成
	ragMcpClient, err := s.initRAGFlowMCP(ctx)
	if err != nil {
		logger.ErrorCtx(ctx, map[string]any{
			"action":  "prepare_user_mcp_client",
			"stage":   "init_ragflow_mcp_client",
			"user_id": userID,
			"error":   err.Error(),
		})
		return nil, fmt.Errorf("failed to init ragflow mcp: %w", err)
	}
	m := map[string]*client.Client{
		"yqlx":    yqlxMcpClient,
		"ragflow": ragMcpClient,
	}

	s.userClientsMu.Lock()
	s.userClients[userID] = m
	s.userClientsMu.Unlock()
	return m, nil
}

// streamEventProcessor 处理 Agent 事件流并输出到通道
// 这是 StreamChat 和 ResumeChat 共用的核心逻辑
type streamEventProcessor struct {
	ctx            context.Context
	userID         uint
	conversationID uint
	checkpointID   string
	messages       []*schema.Message
	conv           *models.Conversation
	service        *ChatService
	outputChan     chan string
	errChan        chan error
	startEventType string // "start" 或 "resume_start"
}

func (p *streamEventProcessor) process(iter *adk.AsyncIterator[*adk.AgentEvent]) {
	defer close(p.outputChan)
	defer close(p.errChan)

	var fullContent string
	var fullToolCalls []schema.ToolCall
	var usage *schema.TokenUsage
	messageCount := 0

	// 发送开始事件
	startEvent := map[string]interface{}{
		"type":          p.startEventType,
		"checkpoint_id": p.checkpointID,
	}
	if data, err := json.Marshal(startEvent); err == nil {
		p.outputChan <- "data: " + string(data) + "\n\n"
	}

	// 遍历 Agent 事件
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
			p.errChan <- event.Err
			return
		}

		// 处理中断事件
		if event.Action != nil && event.Action.Interrupted != nil {
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

		// 处理消息输出
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		msgOutput := event.Output.MessageOutput
		messageCount++

		// 处理流式消息
		if msgOutput.IsStreaming {
			st := msgOutput.MessageStream
			for {
				recv, err := st.Recv()
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
					p.errChan <- err
					return
				}

				// 累积并发送内容
				if recv.Content != "" {
					fullContent += recv.Content
					contentEvent := map[string]interface{}{
						"type":    "content",
						"content": recv.Content,
					}
					if data, err := json.Marshal(contentEvent); err == nil {
						p.outputChan <- "data: " + string(data) + "\n\n"
					}
				}

				// 处理 reasoning content
				if recv.ReasoningContent != "" {
					reasoningEvent := map[string]interface{}{
						"type":    "reasoning",
						"content": recv.ReasoningContent,
					}
					if data, err := json.Marshal(reasoningEvent); err == nil {
						p.outputChan <- "data: " + string(data) + "\n\n"
					}
				}

				// 累积工具调用
				if len(recv.ToolCalls) > 0 {
					fullToolCalls = append(fullToolCalls, recv.ToolCalls...)
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

				// 保存 usage 信息
				if recv.ResponseMeta != nil && recv.ResponseMeta.Usage != nil {
					usage = recv.ResponseMeta.Usage
				}
			}
		} else {
			// 非流式消息（如工具调用结果）
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

			// 发送工具结果事件
			if msg.Role == schema.Tool {
				toolResultEvent := map[string]interface{}{
					"type":         "tool_result",
					"tool_call_id": msg.ToolCallID,
					"content":      msg.Content,
				}
				if data, err := json.Marshal(toolResultEvent); err == nil {
					p.outputChan <- "data: " + string(data) + "\n\n"
				}
			}

			// 累积 Assistant 消息内容
			if msg.Role == schema.Assistant && msg.Content != "" {
				fullContent += msg.Content
			}
			if len(msg.ToolCalls) > 0 {
				fullToolCalls = append(fullToolCalls, msg.ToolCalls...)
			}
		}
	}

	logger.InfoCtx(p.ctx, map[string]any{
		"action":          "agent_stream_finished",
		"user_id":         p.userID,
		"conversation_id": p.conversationID,
		"checkpoint_id":   p.checkpointID,
		"message_count":   messageCount,
	})

	// 构建助手响应消息
	assistantMsg := &schema.Message{
		Role:    schema.Assistant,
		Content: fullContent,
	}
	if len(fullToolCalls) > 0 {
		assistantMsg.ToolCalls = fullToolCalls
	}

	// 添加助手响应到消息列表
	p.messages = append(p.messages, assistantMsg)

	// 保存更新后的消息列表到数据库和缓存
	if err := p.service.SaveMessages(p.ctx, p.userID, p.conv.ID, p.messages); err != nil {
		logger.ErrorCtx(p.ctx, map[string]any{
			"action":          "agent_save_messages",
			"user_id":         p.userID,
			"conversation_id": p.conversationID,
			"checkpoint_id":   p.checkpointID,
			"error":           err.Error(),
		})
	}

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
		return nil, fmt.Errorf("failed to create chat model: %w", err)
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
				Tools: tools,
			},
		},
		MaxIterations: 15, //todo: 配置化
	})
	if err != nil {
		logger.ErrorCtx(ctx, map[string]any{
			"action": "create_agent",
			"error":  err.Error(),
		})
		return nil, fmt.Errorf("failed to create agent: %w", err)
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
		return nil, nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	// 获取完整的会话消息（使用缓存，不存在则从数据库构建）
	messages, err := s.GetMessages(ctx, userID, conversationID)
	if err != nil {
		logger.ErrorCtx(ctx, map[string]any{
			"action":          "stream_chat_get_messages",
			"user_id":         userID,
			"conversation_id": conversationID,
			"error":           err.Error(),
		})
		return nil, nil, fmt.Errorf("failed to get messages: %w", err)
	}

	var allTools []einotool.BaseTool
	isResume := checkpointID != ""

	// 新对话：合并新消息，加载工具
	if !isResume {
		if newMessage == nil {
			return nil, nil, errors.New("message is required for new conversation")
		}
		messages = append(messages, newMessage)

		// MCP 工具为"增强能力"，不应阻塞基础聊天能力。
		mcpClients, err := s.prepareUserMcpClient(ctx, userID, userToken)
		if err != nil {
			logger.WarnCtx(ctx, map[string]any{
				"action":  "stream_chat_prepare_mcp_client",
				"user_id": userID,
				"msg":     "MCP client unavailable, fallback to no-tools chat",
				"error":   err.Error(),
			})
		} else {
			for _, cli := range mcpClients {
				tools, err := einomcp.GetTools(ctx, &einomcp.Config{Cli: cli})
				if err != nil {
					logger.WarnCtx(ctx, map[string]any{
						"action":  "stream_chat_get_mcp_tools",
						"user_id": userID,
						"msg":     "Failed to load MCP tools, skipping",
						"error":   err.Error(),
					})
					continue
				}
				allTools = append(allTools, tools...)
			}
		}
		logger.InfoCtx(ctx, map[string]any{
			"action":          "stream_chat_loaded_mcp_tools",
			"user_id":         userID,
			"conversation_id": conversationID,
			"tool_count":      len(allTools),
		})
	}

	// 创建 Agent
	agent, err := s.createAgent(ctx, allTools)
	if err != nil {
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
			logger.ErrorCtx(ctx, map[string]any{
				"action":          "stream_chat_resume_agent",
				"user_id":         userID,
				"conversation_id": conversationID,
				"checkpoint_id":   checkpointID,
				"error":           err.Error(),
			})
			return nil, nil, fmt.Errorf("failed to resume agent: %w", err)
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

		// 生成新的 checkpointID
		checkpointID = fmt.Sprintf("%d:%d:%d", userID, conversationID, time.Now().UnixNano())

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
		messages:       messages,
		conv:           conv,
		service:        s,
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

// Cleanup 清理资源
func (s *ChatService) Cleanup() {
	s.userClientsMu.Lock()
	defer s.userClientsMu.Unlock()
	for _, client := range s.userClients {
		if client != nil {
			// TODO: close client when needed
		}
	}
}
