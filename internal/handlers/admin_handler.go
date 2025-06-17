package handlers

import (
	"net/http"
	"strconv"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/utils"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	reviewService *services.ReviewService
}

func NewAdminHandler(reviewService *services.ReviewService) *AdminHandler {
	return &AdminHandler{
		reviewService: reviewService}
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
func (h *AdminHandler) GetReviews(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	teacherName := c.Query("teacher_name")
	status, _ := strconv.Atoi(c.Query("status"))

	reviews, total, err := h.reviewService.GetReviews(page, size, teacherName, models.TeacherReviewStatus(status))
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取评价列表失败")
		return
	}

	utils.PageSuccessResponse(c, reviews, total, page, size)
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
func (h *AdminHandler) ApproveReview(c *gin.Context) {
	reviewID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.ValidateResponse(c, "无效的评价ID")
		return
	}

	var req ApproveReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidateResponse(c, "请提供管理员备注")
		return
	}

	if err := h.reviewService.ApproveReview(uint(reviewID), req.AdminNote); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "审核失败")
		return
	}

	utils.SuccessResponse(c, gin.H{"message": "审核通过"})
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
func (h *AdminHandler) RejectReview(c *gin.Context) {
	reviewID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.ValidateResponse(c, "无效的评价ID")
		return
	}

	var req ApproveReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidateResponse(c, "请提供拒绝理由")
		return
	}

	if req.AdminNote == "" {
		utils.ValidateResponse(c, "请提供拒绝理由")
		return
	}

	if err := h.reviewService.RejectReview(uint(reviewID), req.AdminNote); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "审核失败")
		return
	}

	utils.SuccessResponse(c, gin.H{"message": "审核拒绝"})
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
func (h *AdminHandler) DeleteReview(c *gin.Context) {
	reviewID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.ValidateResponse(c, "无效的评价ID")
		return
	}

	if err := h.reviewService.DeleteReview(uint(reviewID)); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除失败")
		return
	}

	utils.SuccessResponse(c, gin.H{"message": "删除成功"})
}
