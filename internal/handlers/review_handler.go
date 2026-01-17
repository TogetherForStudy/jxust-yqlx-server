package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"
)

type ReviewHandler struct {
	reviewService *services.ReviewService
}

func NewReviewHandler(reviewService *services.ReviewService) *ReviewHandler {
	return &ReviewHandler{
		reviewService: reviewService,
	}
}

// CreateReview 创建教师评价
// @Summary 创建教师评价
// @Description 用户对教师进行评价
// @Tags 评价
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param body body request.CreateReviewRequest true "评价信息"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Router /api/reviews [post]
func (h *ReviewHandler) CreateReview(c *gin.Context) {
	userID := helper.GetUserID(c)
	if userID == 0 {
		helper.ErrorResponse(c, http.StatusUnauthorized, "未获取到用户信息")
		return
	}

	var req request.CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	if err := h.reviewService.CreateReview(c.Request.Context(), userID, &req); err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "评价提交成功，等待审核"})
}

// GetReviewsByTeacher 获取指定教师的评价
// @Summary 获取教师评价列表
// @Description 根据教师姓名获取该教师的所有已审核评价
// @Tags 评价
// @Accept json
// @Produce json
// @Param teacher_name query string true "教师姓名"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} utils.PageResponse{data=[]models.TeacherReview}
// @Failure 400 {object} utils.Response
// @Router /api/reviews/teacher [get]
func (h *ReviewHandler) GetReviewsByTeacher(c *gin.Context) {
	teacherName := c.Query("teacher_name")
	if teacherName == "" {
		helper.ValidateResponse(c, "请提供教师姓名")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

	reviews, total, err := h.reviewService.GetReviewsByTeacher(c, teacherName, page, size)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, "获取评价失败")
		return
	}

	helper.PageSuccessResponse(c, reviews, total, page, size)
}

// GetUserReviews 获取用户的评价记录
// @Summary 获取用户评价记录
// @Description 获取当前用户的所有评价记录
// @Tags 评价
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} utils.PageResponse{data=[]models.TeacherReview}
// @Failure 401 {object} utils.Response
// @Router /api/reviews/user [get]
func (h *ReviewHandler) GetUserReviews(c *gin.Context) {
	userID := helper.GetUserID(c)
	if userID == 0 {
		helper.ErrorResponse(c, http.StatusUnauthorized, "未获取到用户信息")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

	reviews, total, err := h.reviewService.GetUserReviews(c.Request.Context(), userID, page, size)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, "获取评价记录失败")
		return
	}

	helper.PageSuccessResponse(c, reviews, total, page, size)
}

// GetReviews 获取所有评价列表（管理员）
// @Summary 获取评价列表
// @Description 管理员获取所有评价列表，支持筛选
// @Tags 管理员
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param teacher_name query string false "教师姓名"
// @Param status query int false "状态：1=待审核,2=已通过,3=已拒绝"
// @Success 200 {object} utils.PageResponse{data=[]models.TeacherReview}
// @Failure 403 {object} utils.Response
// @Router /api/admin/reviews [get]
func (h *ReviewHandler) GetReviews(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	teacherName := c.Query("teacher_name")
	status, _ := strconv.Atoi(c.Query("status"))

	reviews, total, err := h.reviewService.GetReviews(c, page, size, teacherName, models.TeacherReviewStatus(status))
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, "获取评价列表失败")
		return
	}

	helper.PageSuccessResponse(c, reviews, total, page, size)
}

// ApproveReviewRequest 审核评价请求
type ApproveReviewRequest struct {
	AdminNote string `json:"admin_note"`
}

// ApproveReview 审核通过评价
// @Summary 审核通过评价
// @Description 管理员审核通过评价
// @Tags 管理员
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "评价ID"
// @Param body body ApproveReviewRequest false "管理员备注"
// @Success 200 {object} utils.Response
// @Failure 403 {object} utils.Response
// @Router /api/admin/reviews/{id}/approve [post]
func (h *ReviewHandler) ApproveReview(c *gin.Context) {
	reviewID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		helper.ValidateResponse(c, "无效的评价ID")
		return
	}

	var req ApproveReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "请提供管理员备注")
		return
	}

	if err := h.reviewService.ApproveReview(c, uint(reviewID), req.AdminNote); err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, "审核失败")
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "审核通过"})
}

// RejectReview 审核拒绝评价
// @Summary 审核拒绝评价
// @Description 管理员审核拒绝评价
// @Tags 管理员
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "评价ID"
// @Param body body ApproveReviewRequest true "管理员备注"
// @Success 200 {object} utils.Response
// @Failure 403 {object} utils.Response
// @Router /api/admin/reviews/{id}/reject [post]
func (h *ReviewHandler) RejectReview(c *gin.Context) {
	reviewID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		helper.ValidateResponse(c, "无效的评价ID")
		return
	}

	var req ApproveReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "请提供拒绝理由")
		return
	}

	if req.AdminNote == "" {
		helper.ValidateResponse(c, "请提供拒绝理由")
		return
	}

	if err := h.reviewService.RejectReview(c, uint(reviewID), req.AdminNote); err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, "审核失败")
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "审核拒绝"})
}

// DeleteReview 删除评价
// @Summary 删除评价
// @Description 管理员删除评价
// @Tags 管理员
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "评价ID"
// @Success 200 {object} utils.Response
// @Failure 403 {object} utils.Response
// @Router /api/admin/reviews/{id} [delete]
func (h *ReviewHandler) DeleteReview(c *gin.Context) {
	reviewID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		helper.ValidateResponse(c, "无效的评价ID")
		return
	}

	if err := h.reviewService.DeleteReview(c, uint(reviewID)); err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, "删除失败")
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "删除成功"})
}
