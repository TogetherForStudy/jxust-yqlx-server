package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/utils"
	"github.com/cloudwego/eino/schema"
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
		helper.ValidateResponse(c, err.Error())
		return
	}

	userID := helper.GetUserID(c)
	conv, err := h.service.CreateConversation(c.Request.Context(), userID, req.Title)
	if err != nil {
		logger.Errorf("RequestID[%s]: Failed to create conversation: %v", utils.GetRequestID(c), err)
		helper.ErrorResponse(c, http.StatusInternalServerError, "Failed to create conversation")
		return
	}

	helper.SuccessResponse(c, dto.ConversationResponse{
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
		helper.ValidateResponse(c, err.Error())
		return
	}

	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	userID := helper.GetUserID(c)
	conversations, total, err := h.service.ListConversations(c.Request.Context(), userID, req.Page, req.PageSize)
	if err != nil {
		logger.Errorf("RequestID[%s]: Failed to list conversations: %v", utils.GetRequestID(c), err)
		helper.ErrorResponse(c, http.StatusInternalServerError, "Failed to list conversations")
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

	helper.SuccessResponse(c, dto.ConversationListResponse{
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
		helper.ValidateResponse(c, "Invalid conversation ID")
		return
	}

	userID := helper.GetUserID(c)
	if err := h.service.DeleteConversation(c.Request.Context(), userID, uint(conversationID)); err != nil {
		logger.Errorf("RequestID[%s]: Failed to delete conversation: %v", utils.GetRequestID(c), err)
		helper.ErrorResponse(c, http.StatusInternalServerError, "Failed to delete conversation")
		return
	}

	helper.SuccessResponse(c, "ok")
}

// UpdateConversation 更新对话
func (h *ChatHandler) UpdateConversation(c *gin.Context) {
	conversationID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		helper.ValidateResponse(c, "Invalid conversation ID")
		return
	}

	var req dto.UpdateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, err.Error())
		return
	}

	userID := helper.GetUserID(c)
	if err := h.service.UpdateConversation(c.Request.Context(), userID, uint(conversationID), req.Title); err != nil {
		logger.Errorf("RequestID[%s]: Failed to update conversation: %v", utils.GetRequestID(c), err)
		helper.ErrorResponse(c, http.StatusInternalServerError, "Failed to update conversation")
		return
	}

	helper.SuccessResponse(c, "ok")
}

// ChooseConversation 选择对话并返回历史消息（只返回User和Assistant角色的消息）
func (h *ChatHandler) ChooseConversation(c *gin.Context) {
	conversationID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		helper.ValidateResponse(c, "Invalid conversation ID")
		return
	}

	userID := helper.GetUserID(c)
	messages, err := h.service.GetMessages(c.Request.Context(), userID, uint(conversationID))
	if err != nil {
		logger.Errorf("RequestID[%s]: Failed to get conversation messages: %v", utils.GetRequestID(c), err)
		helper.ErrorResponse(c, http.StatusInternalServerError, "Failed to get conversation messages")
		return
	}

	// 过滤消息，只返回 User 和 Assistant 角色
	filteredMessages := make([]*schema.Message, 0, len(messages))
	for _, msg := range messages {
		if msg.Role == schema.User || msg.Role == schema.Assistant {
			filteredMessages = append(filteredMessages, msg)
		}
	}

	helper.SuccessResponse(c, filteredMessages)
}

// ExportConversation 导出对话
func (h *ChatHandler) ExportConversation(c *gin.Context) {
	conversationID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		helper.ValidateResponse(c, "Invalid conversation ID")
		return
	}

	userID := helper.GetUserID(c)
	conv, messages, err := h.service.ExportConversation(c.Request.Context(), userID, uint(conversationID))
	if err != nil {
		logger.Errorf("RequestID[%s]: Failed to export conversation: %v", utils.GetRequestID(c), err)
		helper.ErrorResponse(c, http.StatusInternalServerError, "Failed to export conversation")
		return
	}

	helper.SuccessResponse(c, dto.ExportConversationResponse{
		Conversation: dto.ConversationResponse{
			ID:            conv.ID,
			Title:         conv.Title,
			CreatedAt:     conv.CreatedAt,
			UpdatedAt:     conv.UpdatedAt,
			LastMessageAt: conv.LastMessageAt,
		},
		Messages: messages,
	})
}

// StreamConversation SSE 流式对话（支持新对话和恢复中断的对话）
func (h *ChatHandler) StreamConversation(c *gin.Context) {
	var req dto.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, err.Error())
		return
	}

	// 验证请求：新对话需要 message，恢复需要 checkpoint_id
	if !req.IsResume() && req.Message == nil {
		helper.ValidateResponse(c, "message is required for new conversation")
		return
	}

	userID := helper.GetUserID(c)

	outputChan, errChan, err := h.service.StreamChat(
		c.Request.Context(),
		userID,
		req.ConversationID,
		req.Message,
		helper.GetAuthorizationToken(c),
		req.CheckpointID,
		req.ResumeInput,
	)
	if err != nil {
		logger.Errorf("RequestID[%s]: Failed to stream conversation: %v", utils.GetRequestID(c), err)
		helper.ErrorResponse(c, http.StatusInternalServerError, "Failed to stream conversation")
		return
	}

	// 设置 SSE 响应头
	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
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
			write, err := c.Writer.Write([]byte(msg))
			if err != nil {
				logger.Errorf("RequestID[%s]: Failed to write SSE message, written bytes: %d, error: %v", utils.GetRequestID(c), write, err)
				return
			}
			c.Writer.Flush()
		case err := <-errChan:
			if err != nil {
				errorEvent := map[string]interface{}{
					"type":  "error",
					"error": err.Error(),
				}
				if data, jsonErr := json.Marshal(errorEvent); jsonErr == nil {
					write, err := c.Writer.Write([]byte("data: " + string(data) + "\n\n"))
					if err != nil {
						logger.Errorf("RequestID[%s]: Failed to write SSE error message, written bytes: %d, error: %v", utils.GetRequestID(c), write, err)
					}
					c.Writer.Flush()
				}
			}
			return
		case <-c.Request.Context().Done():
			return
		}
	}
}
