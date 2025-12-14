package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/config"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"

	einomcp "github.com/cloudwego/eino-ext/components/tool/mcp"
	einomodel "github.com/cloudwego/eino/components/model"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type mcpClient map[string]*client.Client //MCP 客户端集合 key: mcp name, value: client
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
func (s *ChatService) initRAGFlowMCP(ctx context.Context, sessionId string) (*client.Client, error) {
	// TODO: 根据 eino 的 MCP 集成文档实现
	// 参考: https://www.cloudwego.io/docs/eino/ecosystem_integration/tool/tool_mcp/
	// todo: ragflow sse endpoint 还需要 session_id 参数
	mcpClient, err := client.NewSSEMCPClient(fmt.Sprintf("%s/messages/?session_id=%s", s.cfg.LLM.RAGFlowMCPURL, sessionId),
		transport.WithHeaders(
			map[string]string{
				"api_key": s.cfg.LLM.RAGFlowAPIKey, // global api key
			}),
		// transport.WithHTTPTimeout(30*time.Second),
		transport.WithSSELogger(logger.L()),
		transport.WithHTTPClient(s.httpClient),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create ragflow mcp client: %w", err)
	}
	return mcpClient, mcpClient.Start(ctx)
}

// CreateConversation 创建新对话
func (s *ChatService) CreateConversation(ctx context.Context, userID uint, title string) (*models.Conversation, error) {
	conv := &models.Conversation{
		UserID: userID,
		Title:  title,
	}

	if err := s.db.WithContext(ctx).Create(conv).Error; err != nil {
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
		return nil, 0, err
	}

	if err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&conversations).Error; err != nil {
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
		return result.Error
	}

	if result.RowsAffected == 0 {
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
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("conversation not found")
	}

	return nil
}

// GetMessages 获取对话的所有消息
func (s *ChatService) GetMessages(ctx context.Context, userID, conversationID uint) ([]models.Message, error) {
	// 验证对话属于用户
	var conv models.Conversation
	if err := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", conversationID, userID).
		First(&conv).Error; err != nil {
		return nil, err
	}

	var messages []models.Message
	if err := s.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("created_at ASC").
		Find(&messages).Error; err != nil {
		return nil, err
	}

	return messages, nil
}

// SaveMessage 保存消息
func (s *ChatService) SaveMessage(ctx context.Context, conversationID uint, role, content string, toolCalls interface{}, tokenCount int) (*models.Message, error) {
	var toolCallsJSON datatypes.JSON
	if toolCalls != nil {
		data, err := json.Marshal(toolCalls)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal tool calls: %w", err)
		}
		toolCallsJSON = data
	}

	msg := &models.Message{
		ConversationID: conversationID,
		Role:           role,
		Content:        content,
		ToolCalls:      toolCallsJSON,
		TokenCount:     tokenCount,
	}

	if err := s.db.WithContext(ctx).Create(msg).Error; err != nil {
		return nil, err
	}

	// 更新对话的最后消息时间
	now := time.Now()
	s.db.WithContext(ctx).Model(&models.Conversation{}).
		Where("id = ?", conversationID).
		Updates(map[string]interface{}{
			"last_message_at": now,
			"updated_at":      now,
		})

	return msg, nil
}
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
		logger.Errorf("%s userID:%d", msg, userID)
		return nil, errors.New(msg)
	}
	if err := yqlxMcpClient.Start(ctx); err != nil {
		logger.Errorf("failed to start yqlx mcp client: %v userID:%d", err, userID)
		return nil, err
	}
	// 初始化 RAGFlow MCP 工具
	// todo: sessionId 应该是每个用户唯一的，可以用 userID 或者其他方式生成
	ragMcpClient, err := s.initRAGFlowMCP(ctx, "todoSessionId")
	if err != nil {
		return nil, fmt.Errorf("failed to init ragflow mcp: %w", err)
	}
	m := map[string]*client.Client{
		"yqlx":    yqlxMcpClient,
		"ragflow": ragMcpClient,
	}

	if err := yqlxMcpClient.Start(ctx); err != nil {
		logger.Errorf("failed to start yqlx mcp client: %v userID:%d", err, userID)
		return nil, err
	}
	s.userClientsMu.Lock()
	s.userClients[userID] = m
	s.userClientsMu.Unlock()
	return m, nil
}

// StreamChat 流式聊天 (返回一个通道用于 SSE)
func (s *ChatService) StreamChat(ctx context.Context, userID, conversationID uint, messages []dto.EinoMessage, userToken string) (<-chan string, <-chan error, error) {
	// 验证对话属于用户
	conv, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return nil, nil, err
	}
	var (
		mcpClients mcpClient
		// ok         bool
		einoTools []einotool.BaseTool
	)

	mcpClients, err = s.prepareUserMcpClient(ctx, userID, userToken)
	if err != nil {
		return nil, nil, err
	}
	for _, client := range mcpClients {
		tools, err := einomcp.GetTools(ctx, &einomcp.Config{Cli: client})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get tools from mcp client: %w", err)
		}
		einoTools = append(einoTools, tools...)
	}
	// todo: 缓存 MCP 客户端. 怎么处理 ctx? 用 context.Background ? LRU->need a struct to record it?
	//s.userClientsMu.RLock()
	//if mcpClients, ok = s.userClients[userID]; !ok {
	//	s.userClientsMu.RUnlock()
	//	mcpClients, err = s.prepareUserMcpClient(ctx, userID, userToken)
	//	if err != nil {
	//		return nil, nil, err
	//	}
	//}

	// 转换消息格式为 eino 格式
	einoMessages := make([]*schema.Message, 0, len(messages))
	for _, msg := range messages {
		einoMsg := &schema.Message{
			Role:    schema.RoleType(msg.Role),
			Content: msg.Content,
		}
		// TODO: 处理 tool calls
		einoMessages = append(einoMessages, einoMsg)
	}

	// 创建输出通道
	outputChan := make(chan string, 10)
	errChan := make(chan error, 1)

	// 启动流式处理
	go func() {
		defer close(outputChan)
		defer close(errChan)

		// TODO: 实现真实的 LLM 流式调用
		// 这里是临时的模拟实现
		outputChan <- "data: " + `{"type":"start"}` + "\n\n"
		outputChan <- "data: " + `{"type":"content","content":"这是一个模拟的 LLM 响应。"}` + "\n\n"
		outputChan <- "data: " + `{"type":"content","content":"请配置正确的 LLM 模型。"}` + "\n\n"
		outputChan <- "data: " + `{"type":"end"}` + "\n\n"

		// 保存用户消息
		for _, msg := range messages {
			if msg.Role == "user" {
				s.SaveMessage(ctx, conv.ID, msg.Role, msg.Content, msg.ToolCalls, 0)
			}
		}

		// 保存助手响应
		s.SaveMessage(ctx, conv.ID, "assistant", "这是一个模拟的 LLM 响应。请配置正确的 LLM 模型。", nil, 0)
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
func (s *ChatService) ExportConversation(ctx context.Context, userID, conversationID uint) (*models.Conversation, []models.Message, error) {
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
