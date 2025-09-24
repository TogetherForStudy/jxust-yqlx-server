package handlers

import (
	"net/http"
	"strconv"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"

	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	notificationService *services.NotificationService
}

func NewNotificationHandler(notificationService *services.NotificationService) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
	}
}

// CreateNotification 创建通知（运营专用）
func (h *NotificationHandler) CreateNotification(c *gin.Context) {
	userID := helper.GetUserID(c)
	userRole := helper.GetUserRole(c)

	var req request.CreateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	result, err := h.notificationService.CreateNotification(userID, models.UserRole(userRole), &req)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.SuccessResponse(c, result)
}

// GetNotifications 获取通知列表
func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	var req request.GetNotificationsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 20
	}
	if req.Status != nil && *req.Status == 0 {
		req.Status = nil
	}
	if len(req.Categories) > 0 && req.Categories[0] == 0 {
		req.Categories = nil
	}

	result, err := h.notificationService.GetNotifications(&req)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessResponse(c, result)
}

// GetNotificationByID 获取通知详情
func (h *NotificationHandler) GetNotificationByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, "无效的通知ID")
		return
	}

	result, err := h.notificationService.GetNotificationByID(uint(id))
	if err != nil {
		if err.Error() == "通知不存在" {
			helper.ErrorResponse(c, http.StatusNotFound, err.Error())
			return
		}
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessResponse(c, result)
}

// GetNotificationAdminByID 获取通知详情(管理员)
func (h *NotificationHandler) GetNotificationAdminByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, "无效的通知ID")
		return
	}

	result, err := h.notificationService.GetNotificationAdminByID(uint(id))
	if err != nil {
		if err.Error() == "通知不存在" {
			helper.ErrorResponse(c, http.StatusNotFound, err.Error())
			return
		}
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessResponse(c, result)
}

// UpdateNotification 更新通知
func (h *NotificationHandler) UpdateNotification(c *gin.Context) {
	userRole := helper.GetUserRole(c)

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, "无效的通知ID")
		return
	}

	var req request.UpdateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	result, err := h.notificationService.UpdateNotification(uint(id), models.UserRole(userRole), &req)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.SuccessResponse(c, result)
}

// PublishNotification 发布通知
func (h *NotificationHandler) PublishNotification(c *gin.Context) {
	userID := helper.GetUserID(c)
	userRole := helper.GetUserRole(c)

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, "无效的通知ID")
		return
	}

	err = h.notificationService.PublishNotification(uint(id), userID, models.UserRole(userRole))
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "通知发布成功"})
}

// ConvertToSchedule 转换通知为日程
func (h *NotificationHandler) ConvertToSchedule(c *gin.Context) {

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, "无效的通知ID")
		return
	}

	var req request.ConvertToScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	err = h.notificationService.ConvertToSchedule(uint(id), &req)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "转换为日程成功"})
}

// DeleteNotification 删除通知
func (h *NotificationHandler) DeleteNotification(c *gin.Context) {
	userID := helper.GetUserID(c)
	userRole := helper.GetUserRole(c)

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, "无效的通知ID")
		return
	}

	err = h.notificationService.DeleteNotification(uint(id), userID, models.UserRole(userRole))
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "通知删除成功"})
}

// CreateCategory 创建分类
func (h *NotificationHandler) CreateCategory(c *gin.Context) {
	var req request.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	result, err := h.notificationService.CreateCategory(&req)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.SuccessResponse(c, result)
}

// GetCategories 获取所有分类
func (h *NotificationHandler) GetCategories(c *gin.Context) {
	result, err := h.notificationService.GetCategories()
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessResponse(c, result)
}

// UpdateCategory 更新分类
func (h *NotificationHandler) UpdateCategory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 8)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, "无效的分类ID")
		return
	}

	var req request.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	result, err := h.notificationService.UpdateCategory(uint8(id), &req)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.SuccessResponse(c, result)
}

// ApproveNotification 审核通知（管理员专用）
func (h *NotificationHandler) ApproveNotification(c *gin.Context) {
	userID := helper.GetUserID(c)
	userRole := helper.GetUserRole(c)

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, "无效的通知ID")
		return
	}

	var req request.ApproveNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	err = h.notificationService.ApproveNotification(uint(id), userID, models.UserRole(userRole), &req)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "审核完成"})
}

// GetAdminNotifications 获取管理员通知列表（包括待审核的）
func (h *NotificationHandler) GetAdminNotifications(c *gin.Context) {
	userRole := helper.GetUserRole(c)

	var req request.GetNotificationsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 20
	}
	if req.Status != nil && *req.Status == 0 {
		req.Status = nil
	}
	if len(req.Categories) > 0 && req.Categories[0] == 0 {
		req.Categories = nil
	}

	result, err := h.notificationService.GetAdminNotifications(models.UserRole(userRole), &req)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessResponse(c, result)
}
