package handlers

import (
	"net/http"
	"strconv"

	"goJxust/internal/services"
	"goJxust/internal/utils"

	"github.com/gin-gonic/gin"
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
// @Param body body services.CreateReviewRequest true "评价信息"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Router /api/reviews [post]
func (h *ReviewHandler) CreateReview(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未获取到用户信息")
		return
	}

	var req services.CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidateResponse(c, "参数验证失败")
		return
	}

	if err := h.reviewService.CreateReview(userID.(uint), &req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{"message": "评价提交成功，等待审核"})
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
		utils.ValidateResponse(c, "请提供教师姓名")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

	reviews, total, err := h.reviewService.GetReviewsByTeacher(teacherName, page, size)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取评价失败")
		return
	}

	utils.PageSuccessResponse(c, reviews, total, page, size)
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
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未获取到用户信息")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

	reviews, total, err := h.reviewService.GetUserReviews(userID.(uint), page, size)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取评价记录失败")
		return
	}

	utils.PageSuccessResponse(c, reviews, total, page, size)
}
