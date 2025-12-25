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
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/utils"
	"github.com/mark3labs/mcp-go/mcp"

	einoopenai "github.com/cloudwego/eino-ext/components/model/openai"
	einomcp "github.com/cloudwego/eino-ext/components/tool/mcp"
	einomodel "github.com/cloudwego/eino/components/model"
	einotool "github.com/cloudwego/eino/components/tool"

	json "github.com/bytedance/sonic"
	"github.com/cloudwego/eino/schema"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"gorm.io/gorm"
)

type mcpClient map[string]*client.Client

type ChatService struct {
	httpClient *http.Client // 复用的 HTTP 客户端,用于 MCP 等外部请求,减少GC压力和连接创建开销
	db         *gorm.DB
	cfg        *config.Config
	llm        einomodel.ChatModel
	tools      []einotool.BaseTool

	//mcpClients    mcpClient          // global
	userClients   map[uint]mcpClient // per-user MCP 客户端集合 key: userID, value: mcpClient
	userClientsMu sync.RWMutex
}

func NewChatService(db *gorm.DB, cfg *config.Config) *ChatService {
	services := &ChatService{
		db:         db,
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		//mcpClients:  make(mcpClient),
		userClients: make(map[uint]mcpClient),
	}
	if err := services.InitializeLLM(); err != nil {
		logger.Fatalf("Failed to initialize LLM: %v", err)
	}
	return services
}

// InitializeLLM 初始化 LLM 和工具
func (s *ChatService) InitializeLLM() error {
	// TODO: 初始化 LLM 模型
	// 这里需要根据配置初始化具体的 LLM 提供商 (OpenAI, etc)
	// s.llm = ...

	return nil
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
		logger.Errorf("RequestID[%s]:Failed to initialize RAG flow MCP client: %v ", utils.GetRequestID(ctx), err)
		return nil, fmt.Errorf("failed to create ragflow mcp client: %w", err)
	}
	err = mcpClient.Start(ctx)
	if err != nil {
		logger.Errorf("RequestID[%s]:Failed to start RAG flow MCP client: %v ", utils.GetRequestID(ctx), err)
		return nil, err
	}
	_, err = mcpClient.Initialize(ctx, mcp.InitializeRequest{})
	if err != nil {
		logger.Errorf("RequestID[%s]:Failed to initialize RAG flow MCP client connection: %v ", utils.GetRequestID(ctx), err)
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
		logger.Errorf("RequestID[%s]:Failed to create conversation: %v", utils.GetRequestID(ctx), err)
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
		Where("user_id = ?", userID).
		Count(&total).Error; err != nil {
		logger.Errorf("RequestID[%s]:Failed to list conversations: %v", utils.GetRequestID(ctx), err)
		return nil, 0, err
	}

	if err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&conversations).Error; err != nil {
		logger.Errorf("RequestID[%s]: Failed to list conversations: %v", utils.GetRequestID(ctx), err)
		return nil, 0, err
	}

	return conversations, total, nil
}

// GetConversation 获取对话详情
func (s *ChatService) GetConversation(ctx context.Context, userID, conversationID uint) (*models.Conversation, error) {
	var conv models.Conversation
	if err := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", conversationID, userID).
		First(&conv).Error; err != nil {
		logger.Errorf("RequestID[%s]: Failed to get conversation: %v", utils.GetRequestID(ctx), err)
		return nil, err
	}
	return &conv, nil
}

// DeleteConversation 删除对话
func (s *ChatService) DeleteConversation(ctx context.Context, userID, conversationID uint) error {
	result := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", conversationID, userID).
		Delete(&models.Conversation{})

	if result.Error != nil {
		logger.Errorf("RequestID[%s]: Failed to delete conversation: %v", utils.GetRequestID(ctx), result.Error)
		return result.Error
	}

	if result.RowsAffected == 0 {
		logger.Warnf("RequestID[%s]: Conversation not found for deletion: conversationID=%d, userID=%d", utils.GetRequestID(ctx), conversationID, userID)
		return errors.New("conversation not found")
	}

	return nil
}

// UpdateConversation 更新对话标题
func (s *ChatService) UpdateConversation(ctx context.Context, userID, conversationID uint, title string) error {
	result := s.db.WithContext(ctx).
		Model(&models.Conversation{}).
		Where("id = ? AND user_id = ?", conversationID, userID).
		Update("title", title)

	if result.Error != nil {
		logger.Errorf("RequestID[%s]: Failed to update conversation: %v", utils.GetRequestID(ctx), result.Error)
		return result.Error
	}

	if result.RowsAffected == 0 {
		logger.Warnf("RequestID[%s]: Conversation not found for update: conversationID=%d, userID=%d", utils.GetRequestID(ctx), conversationID, userID)
		return errors.New("conversation not found")
	}

	return nil
}

// GetMessages 获取对话的所有消息（使用缓存，不存在则从数据库构建缓存）
func (s *ChatService) GetMessages(ctx context.Context, userID, conversationID uint) ([]*schema.Message, error) {
	// 验证对话属于用户
	var conv models.Conversation
	if err := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", conversationID, userID).
		First(&conv).Error; err != nil {
		logger.Errorf("RequestID[%s]: Failed to get conversation from database: %v", utils.GetRequestID(ctx), err)
		return nil, err
	}

	// 尝试从缓存获取
	cacheKey := fmt.Sprintf(constant.CacheKeyConversationMessages, userID, conversationID)
	if cachedData, err := cache.GlobalCache.Get(ctx, cacheKey); err == nil && cachedData != "" {
		var messages []*schema.Message
		if err := json.Unmarshal([]byte(cachedData), &messages); err == nil {
			logger.Errorf("RequestID[%s]: Failed to Unmarshal cached data: %v", utils.GetRequestID(ctx), err)
			return messages, nil
		}
	}

	// 从数据库加载
	messages := []*schema.Message{}
	if len(conv.Messages) > 0 {
		if err := json.Unmarshal(conv.Messages, &messages); err != nil {
			logger.Errorf("RequestID[%s]: Failed to unmarshal messages: %v", utils.GetRequestID(ctx), err)
			return nil, fmt.Errorf("failed to unmarshal messages: %w", err)
		}
	}

	// 更新缓存
	if data, err := json.Marshal(messages); err == nil {
		expiration := 30 * time.Minute
		err := cache.GlobalCache.Set(ctx, cacheKey, string(data), &expiration)
		if err != nil {
			logger.Errorf("RequestID[%s]: Failed to set messages cache: %v", utils.GetRequestID(ctx), err)
			return nil, err
		}
	}

	return messages, nil
}

// SaveMessages 保存完整的消息列表到数据库和缓存
func (s *ChatService) SaveMessages(ctx context.Context, userID, conversationID uint, messages []*schema.Message) error {
	messagesJSON, err := json.Marshal(messages)
	if err != nil {
		logger.Errorf("RequestID[%s]: Failed to marshal messages: %v", utils.GetRequestID(ctx), err)
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
		logger.Errorf("RequestID[%s]: Failed to update messages in database: %v", utils.GetRequestID(ctx), err)
		return err
	}

	// 更新缓存
	cacheKey := fmt.Sprintf(constant.CacheKeyConversationMessages, userID, conversationID)
	expiration := 30 * time.Minute
	err = cache.GlobalCache.Set(ctx, cacheKey, string(messagesJSON), &expiration)
	if err != nil {
		logger.Errorf("RequestID[%s]: Failed to set messages cache: %v", utils.GetRequestID(ctx), err)
		return err
	}

	return nil
}

// todo: mcp server 除了系统内置的 mcp client 之外，还可以支持用户自定义的 mcp client
func (s *ChatService) prepareUserMcpClient(ctx context.Context, userID uint, userToken string) (mcpClient, error) {
	yqlxMcpClient, err := client.NewStreamableHttpClient(fmt.Sprintf("http://127.0.0.1:%s/api/mcp", s.cfg.ServerPort), // yqlx自身的 MCP 转发接口
		transport.WithHTTPHeaders(
			map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", userToken),
			}),
		transport.WithHTTPTimeout(30*time.Second),
		transport.WithHTTPLogger(logger.L()),
		transport.WithHTTPBasicClient(s.httpClient))
	if err != nil {
		msg := fmt.Sprintf("failed to create user mcp client: %v", err)
		logger.Errorf("RequestID[%s]: %s userID:%d", utils.GetRequestID(ctx), msg, userID)
		return nil, errors.New(msg)
	}
	if err := yqlxMcpClient.Start(ctx); err != nil {
		logger.Errorf("RequestID[%s]: failed to start yqlx mcp client: %v userID:%d", utils.GetRequestID(ctx), err, userID)
		return nil, err
	}
	_, err = yqlxMcpClient.Initialize(ctx, mcp.InitializeRequest{})
	if err != nil {
		logger.Errorf("RequestID[%s]: failed to initialize yqlx mcp client connection: %v userID:%d", utils.GetRequestID(ctx), err, userID)
		return nil, err
	}
	// 初始化 RAGFlow MCP 工具
	// todo: sessionId 应该是每个用户唯一的，可以用 userID 或者其他方式生成
	ragMcpClient, err := s.initRAGFlowMCP(ctx)
	if err != nil {
		logger.Errorf("RequestID[%s]: failed to init ragflow mcp client: %v userID:%d", utils.GetRequestID(ctx), err, userID)
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

// StreamChat 流式聊天 (返回一个通道用于 SSE)
func (s *ChatService) StreamChat(ctx context.Context, userID, conversationID uint, newMessage *schema.Message, userToken string) (<-chan string, <-chan error, error) {
	// 验证对话属于用户
	conv, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		logger.Errorf("RequestID[%s]: Failed to get conversation: %v", utils.GetRequestID(ctx), err)
		return nil, nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	// 获取完整的会话消息（使用缓存，不存在则从数据库构建）
	messages, err := s.GetMessages(ctx, userID, conversationID)
	if err != nil {
		logger.Errorf("RequestID[%s]: Failed to get messages: %v", utils.GetRequestID(ctx), err)
		return nil, nil, fmt.Errorf("failed to get messages: %w", err)
	}

	// 合并新消息到会话中
	messages = append(messages, newMessage)

	var (
		mcpClients mcpClient
		einoTools  []*schema.ToolInfo
	)

	// MCP 工具为“增强能力”，不应阻塞基础聊天能力。
	mcpClients, err = s.prepareUserMcpClient(ctx, userID, userToken)
	if err != nil {
		logger.Warnf("RequestID[%s]: MCP client unavailable, fallback to no-tools chat: %v", utils.GetRequestID(ctx), err)
	} else {
		for _, client := range mcpClients {
			tools, err := einomcp.GetTools(ctx, &einomcp.Config{Cli: client})
			if err != nil {
				logger.Warnf("RequestID[%s]: Failed to get tools from mcp client, ignore: %v", utils.GetRequestID(ctx), err)
				continue
			}
			for toolsIndex := range tools {
				info, err := tools[toolsIndex].Info(ctx)
				if err != nil {
					logger.Warnf("RequestID[%s]: Failed to get tool info, ignore tool: %v", utils.GetRequestID(ctx), err)
					return nil, nil, err
				}
				einoTools = append(einoTools, info)
			}

		}
	}
	logger.Infof("RequestID[%s]: Streaming chat loaded mcp tools count:%d", utils.GetRequestID(ctx), len(einoTools))
	// 添加系统提示词（仅在首次聊天时）
	var prompts []*schema.Message
	if len(messages) > 0 && messages[0].Role == schema.System {
		// 已有系统提示词，直接使用
		prompts = messages
	} else {
		// 首次聊天，添加系统提示词
		prompts = append([]*schema.Message{schema.SystemMessage(constant.ChatSystemPrompt)}, messages...)
	}

	// 使用配置的 LLM 或创建新实例
	var chatModel einomodel.ChatModel
	if s.llm != nil {
		chatModel = s.llm
	} else {
		// 如果没有全局 LLM，为这个请求创建临时实例
		model, err := einoopenai.NewChatModel(ctx, &einoopenai.ChatModelConfig{
			HTTPClient: s.httpClient,
			Model:      s.cfg.LLM.Model,
			APIKey:     s.cfg.LLM.APIKey,
			BaseURL:    s.cfg.LLM.BaseURL,
		})
		if err != nil {
			logger.Errorf("RequestID[%s]: Failed to create chat model: %v", utils.GetRequestID(ctx), err)
			return nil, nil, fmt.Errorf("failed to create chat model: %w", err)
		}
		chatModel = model
	}

	// 创建流式请求
	sr, err := chatModel.Stream(ctx, prompts, einomodel.WithTools(einoTools))
	if err != nil {
		logger.Errorf("RequestID[%s]: Failed to create chat stream: %v", utils.GetRequestID(ctx), err)
		return nil, nil, fmt.Errorf("failed to create chat stream: %w", err)
	}

	// 创建输出通道
	outputChan := make(chan string, 50)
	errChan := make(chan error, 1)

	// 启动流式处理 goroutine
	go func() {
		defer close(outputChan)
		defer close(errChan)
		defer sr.Close()

		var fullContent string
		var fullToolCalls []schema.ToolCall
		var usage *schema.TokenUsage
		messageCount := 0

		// 发送开始事件
		startEvent := map[string]interface{}{
			"type": "start",
		}
		if data, err := json.Marshal(startEvent); err == nil {
			outputChan <- "data: " + string(data) + "\n\n"
		}

		// 读取流式消息
		for {
			message, err := sr.Recv()
			if err != nil {
				if err == io.EOF {
					break
				}
				logger.Errorf("RequestID[%s]: Error receiving chat stream message: %v", utils.GetRequestID(ctx), err)
				errChan <- err
				return
			}

			messageCount++

			// 累积内容
			if message.Content != "" {
				fullContent += message.Content

				// 发送内容增量
				contentEvent := map[string]interface{}{
					"type":    "content",
					"content": message.Content,
				}
				if data, err := json.Marshal(contentEvent); err == nil {
					outputChan <- "data: " + string(data) + "\n\n"
				}
			}

			// 处理 reasoning content (如果有)
			if message.ReasoningContent != "" {
				reasoningEvent := map[string]interface{}{
					"type":    "reasoning",
					"content": message.ReasoningContent,
				}
				if data, err := json.Marshal(reasoningEvent); err == nil {
					outputChan <- "data: " + string(data) + "\n\n"
				}
			}

			// 累积工具调用
			if len(message.ToolCalls) > 0 {
				fullToolCalls = append(fullToolCalls, message.ToolCalls...)
				// 发送工具调用事件
				for _, tc := range message.ToolCalls {
					toolCallEvent := map[string]interface{}{
						"type":      "tool_call",
						"tool_call": tc,
						"function":  tc.Function.Name,
						"arguments": tc.Function.Arguments,
					}
					if data, err := json.Marshal(toolCallEvent); err == nil {
						outputChan <- "data: " + string(data) + "\n\n"
					}
				}
			}

			// 保存 usage 信息
			if message.ResponseMeta != nil && message.ResponseMeta.Usage != nil {
				usage = message.ResponseMeta.Usage
			}
		}

		logger.Infof("Stream finished with %d messages for conversation %d", messageCount, conversationID)

		// 构建助手响应消息
		assistantMsg := &schema.Message{
			Role:    schema.Assistant,
			Content: fullContent,
		}
		if len(fullToolCalls) > 0 {
			assistantMsg.ToolCalls = fullToolCalls
		}

		// 添加助手响应到消息列表
		messages = append(messages, assistantMsg)

		// 保存更新后的消息列表到数据库和缓存
		if err := s.SaveMessages(ctx, userID, conv.ID, messages); err != nil {
			logger.Errorf("RequestID[%s]: Failed to save messages: %v", utils.GetRequestID(ctx), err)
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
			outputChan <- "data: " + string(data) + "\n\n"
		}
	}()

	return outputChan, errChan, nil
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
