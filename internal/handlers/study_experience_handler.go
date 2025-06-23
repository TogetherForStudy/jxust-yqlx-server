package handlers

import (
	"net/http"
	"strconv"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"

	"github.com/gin-gonic/gin"
)

type StudyExperienceHandler struct {
	service *services.StudyExperienceService
}

func NewStudyExperienceHandler(service *services.StudyExperienceService) *StudyExperienceHandler {
	return &StudyExperienceHandler{
		service: service,
	}
}

// CreateExperience 创建备考经验
func (h *StudyExperienceHandler) CreateExperience(c *gin.Context) {
	// 获取当前用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		helper.ErrorResponse(c, http.StatusUnauthorized, "未授权访问")
		return
	}

	// 绑定请求参数
	var req services.CreateExperienceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "请求参数错误")
		return
	}

	// 创建备考经验
	if err := h.service.CreateExperience(userID.(uint), &req); err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, "创建备考经验失败")
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "创建成功，等待审核"})
}

// GetApprovedExperiences 获取已审核通过的备考经验列表
func (h *StudyExperienceHandler) GetApprovedExperiences(c *gin.Context) {
	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

	// 获取课程筛选参数
	course := c.Query("course")

	// 获取备考经验列表
	experiences, total, err := h.service.GetApprovedExperiences(page, size, course)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, "获取备考经验列表失败")
		return
	}

	helper.PageSuccessResponse(c, experiences, total, page, size)
}

// GetUserExperiences 获取用户的备考经验列表
func (h *StudyExperienceHandler) GetUserExperiences(c *gin.Context) {
	// 获取当前用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		helper.ErrorResponse(c, http.StatusUnauthorized, "未授权访问")
		return
	}

	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

	// 获取用户的备考经验列表
	experiences, total, err := h.service.GetUserExperiences(userID.(uint), page, size)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, "获取备考经验列表失败")
		return
	}

	helper.PageSuccessResponse(c, experiences, total, page, size)
}

// GetExperienceByID 根据ID获取备考经验详情
func (h *StudyExperienceHandler) GetExperienceByID(c *gin.Context) {
	// 获取经验ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		helper.ValidateResponse(c, "无效的ID参数")
		return
	}

	// 获取备考经验详情
	experience, err := h.service.GetExperienceByID(uint(id))
	if err != nil {
		helper.ErrorResponse(c, http.StatusNotFound, "备考经验不存在")
		return
	}

	helper.SuccessResponse(c, experience)
}

// LikeExperience 点赞备考经验
func (h *StudyExperienceHandler) LikeExperience(c *gin.Context) {
	// 获取经验ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		helper.ValidateResponse(c, "无效的ID参数")
		return
	}

	// 点赞备考经验
	if err := h.service.LikeExperience(uint(id)); err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, "点赞失败")
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "点赞成功"})
}

// GetExperiences 获取备考经验列表-管理员
func (h *StudyExperienceHandler) GetExperiences(c *gin.Context) {
	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

	// 获取状态筛选参数
	statusStr := c.Query("status")
	var status models.StudyExperienceStatus
	if statusStr != "" {
		statusVal, err := strconv.Atoi(statusStr)
		if err == nil {
			status = models.StudyExperienceStatus(statusVal)
		}
	}

	// 获取备考经验列表
	experiences, total, err := h.service.GetExperiences(page, size, status)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, "获取备考经验列表失败")
		return
	}

	helper.PageSuccessResponse(c, experiences, total, page, size)
}

// ApproveExperience 审核通过备考经验-管理员
func (h *StudyExperienceHandler) ApproveExperience(c *gin.Context) {
	// 获取经验ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		helper.ValidateResponse(c, "无效的ID参数")
		return
	}

	// 获取管理员备注
	var req struct {
		AdminNote string `json:"admin_note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "请求参数错误")
		return
	}

	// 审核通过
	if err := h.service.ApproveExperience(uint(id), req.AdminNote); err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, "审核操作失败")
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "审核通过成功"})
}

// RejectExperience 审核拒绝备考经验-管理员
func (h *StudyExperienceHandler) RejectExperience(c *gin.Context) {
	// 获取经验ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		helper.ValidateResponse(c, "无效的ID参数")
		return
	}

	// 获取管理员备注
	var req struct {
		AdminNote string `json:"admin_note" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "请提供拒绝原因")
		return
	}

	// 审核拒绝
	if err := h.service.RejectExperience(uint(id), req.AdminNote); err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, "审核操作失败")
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "审核拒绝成功"})
}
