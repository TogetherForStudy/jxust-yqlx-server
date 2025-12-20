package dto

import (
	"time"

	"github.com/cloudwego/eino/schema"
)

// CreateConversationRequest 创建对话请求
type CreateConversationRequest struct {
	Title string `json:"title" binding:"required,max=200"`
}

// UpdateConversationRequest 更新对话请求
type UpdateConversationRequest struct {
	Title string `json:"title" binding:"required,max=200"`
}

// ListConversationsRequest 列出对话请求
type ListConversationsRequest struct {
	Page     int `form:"page" binding:"omitempty,min=1"`
	PageSize int `form:"page_size" binding:"omitempty,min=1,max=100"`
}

// ConversationResponse 对话响应
type ConversationResponse struct {
	ID            uint       `json:"id"`
	Title         string     `json:"title"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	LastMessageAt *time.Time `json:"last_message_at"`
}

// ConversationListResponse 对话列表响应
type ConversationListResponse struct {
	Total         int64                  `json:"total"`
	Page          int                    `json:"page"`
	PageSize      int                    `json:"page_size"`
	Conversations []ConversationResponse `json:"conversations"`
}

// EinoMessage eino消息格式
type EinoMessage struct {
	Role       schema.RoleType   `json:"role"`
	Content    string            `json:"content"`
	ToolCalls  []schema.ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string            `json:"tool_call_id,omitempty"`
}

// ChatRequest 聊天请求
type ChatRequest struct {
	ConversationID uint           `json:"conversation_id" binding:"required"`
	Message        schema.Message `json:"message" binding:"required"`
}

// ExportConversationResponse 导出对话响应
type ExportConversationResponse struct {
	Conversation ConversationResponse `json:"conversation"`
	Messages     []*schema.Message    `json:"messages"`
}
