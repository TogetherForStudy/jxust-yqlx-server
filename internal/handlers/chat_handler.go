package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"
	"github.com/gin-gonic/gin"
)

type ChatHandler struct {
	service *services.ChatService
}

func NewChatHandler(service *services.ChatService) *ChatHandler {
	return &ChatHandler{
		service: service,
	}
}

// CreateConversation 创建新对话
func (h *ChatHandler) CreateConversation(c *gin.Context) {
	var req dto.CreateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")
	conv, err := h.service.CreateConversation(c.Request.Context(), userID, req.Title)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create conversation"})
		return
	}

	c.JSON(http.StatusOK, dto.ConversationResponse{
		ID:            conv.ID,
		Title:         conv.Title,
		CreatedAt:     conv.CreatedAt,
		UpdatedAt:     conv.UpdatedAt,
		LastMessageAt: conv.LastMessageAt,
	})
}

// ListConversations 列出对话
func (h *ChatHandler) ListConversations(c *gin.Context) {
	var req dto.ListConversationsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	userID := c.GetUint("user_id")
	conversations, total, err := h.service.ListConversations(c.Request.Context(), userID, req.Page, req.PageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list conversations"})
		return
	}

	convResponses := make([]dto.ConversationResponse, len(conversations))
	for i, conv := range conversations {
		convResponses[i] = dto.ConversationResponse{
			ID:            conv.ID,
			Title:         conv.Title,
			CreatedAt:     conv.CreatedAt,
			UpdatedAt:     conv.UpdatedAt,
			LastMessageAt: conv.LastMessageAt,
		}
	}

	c.JSON(http.StatusOK, dto.ConversationListResponse{
		Total:         total,
		Page:          req.Page,
		PageSize:      req.PageSize,
		Conversations: convResponses,
	})
}

// DeleteConversation 删除对话
func (h *ChatHandler) DeleteConversation(c *gin.Context) {
	conversationID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	userID := c.GetUint("user_id")
	if err := h.service.DeleteConversation(c.Request.Context(), userID, uint(conversationID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Conversation deleted successfully"})
}

// UpdateConversation 更新对话
func (h *ChatHandler) UpdateConversation(c *gin.Context) {
	conversationID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	var req dto.UpdateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")
	if err := h.service.UpdateConversation(c.Request.Context(), userID, uint(conversationID), req.Title); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Conversation updated successfully"})
}

// ChooseConversation 选择对话并返回历史消息
func (h *ChatHandler) ChooseConversation(c *gin.Context) {
	conversationID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	userID := c.GetUint("user_id")
	messages, err := h.service.GetMessages(c.Request.Context(), userID, uint(conversationID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	messageResponses := make([]dto.MessageResponse, len(messages))
	for i, msg := range messages {
		var toolCalls []map[string]interface{}
		if len(msg.ToolCalls) > 0 {
			json.Unmarshal(msg.ToolCalls, &toolCalls)
		}

		messageResponses[i] = dto.MessageResponse{
			ID:             msg.ID,
			ConversationID: msg.ConversationID,
			Role:           msg.Role,
			Content:        msg.Content,
			ToolCalls:      toolCalls,
			TokenCount:     msg.TokenCount,
			CreatedAt:      msg.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{"messages": messageResponses})
}

// ExportConversation 导出对话
func (h *ChatHandler) ExportConversation(c *gin.Context) {
	conversationID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	userID := c.GetUint("user_id")
	conv, messages, err := h.service.ExportConversation(c.Request.Context(), userID, uint(conversationID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	messageResponses := make([]dto.MessageResponse, len(messages))
	for i, msg := range messages {
		var toolCalls []map[string]interface{}
		if len(msg.ToolCalls) > 0 {
			json.Unmarshal(msg.ToolCalls, &toolCalls)
		}

		messageResponses[i] = dto.MessageResponse{
			ID:             msg.ID,
			ConversationID: msg.ConversationID,
			Role:           msg.Role,
			Content:        msg.Content,
			ToolCalls:      toolCalls,
			TokenCount:     msg.TokenCount,
			CreatedAt:      msg.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, dto.ExportConversationResponse{
		Conversation: dto.ConversationResponse{
			ID:            conv.ID,
			Title:         conv.Title,
			CreatedAt:     conv.CreatedAt,
			UpdatedAt:     conv.UpdatedAt,
			LastMessageAt: conv.LastMessageAt,
		},
		Messages: messageResponses,
	})
}

// StreamConversation SSE 流式对话
func (h *ChatHandler) StreamConversation(c *gin.Context) {
	var req dto.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")

	outputChan, errChan, err := h.service.StreamChat(c.Request.Context(), userID, req.ConversationID, req.Messages)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 设置 SSE 响应头
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	c.Writer.Flush()

	for {
		select {
		case msg, ok := <-outputChan:
			if !ok {
				return
			}
			c.Writer.Write([]byte(msg))
			c.Writer.Flush()
		case err := <-errChan:
			if err != nil {
				c.Writer.Write([]byte("data: {\"type\":\"error\",\"error\":\"" + err.Error() + "\"}\n\n"))
				c.Writer.Flush()
			}
			return
		case <-c.Request.Context().Done():
			return
		}
	}
}
