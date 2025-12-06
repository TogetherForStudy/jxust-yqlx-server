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

type ContributionHandler struct {
	contributionService *services.ContributionService
}

func NewContributionHandler(contributionService *services.ContributionService) *ContributionHandler {
	return &ContributionHandler{
		contributionService: contributionService,
	}
}

// CreateContribution 创建投稿
func (h *ContributionHandler) CreateContribution(c *gin.Context) {
	userID := helper.GetUserID(c)
	userRole := helper.GetUserRole(c)

	if models.UserRole(userRole) != models.UserRoleNormal {
		helper.ErrorResponse(c, http.StatusForbidden, "无权限操作")
		return
	}

	var req request.CreateContributionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	err := h.contributionService.CreateContribution(c, userID, &req)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "投稿创建成功"})
}

// GetContributions 获取投稿列表
func (h *ContributionHandler) GetContributions(c *gin.Context) {
	userID := helper.GetUserID(c)
	userRole := helper.GetUserRole(c)

	var req request.GetContributionsRequest
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
	if req.UserID != nil && *req.UserID == 0 {
		req.UserID = nil
	}

	result, err := h.contributionService.GetContributions(c, userID, models.UserRole(userRole), &req)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessResponse(c, result)
}

// GetContributionByID 获取投稿详情
func (h *ContributionHandler) GetContributionByID(c *gin.Context) {
	userID := helper.GetUserID(c)

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, "无效的投稿ID")
		return
	}

	result, err := h.contributionService.GetContributionByID(c, uint(id), userID, models.UserRole(helper.GetUserRole(c)))
	if err != nil {
		if err.Error() == "投稿不存在" {
			helper.ErrorResponse(c, http.StatusNotFound, err.Error())
			return
		}
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessResponse(c, result)
}

// ReviewContribution 审核投稿（运营/管理员专用）
func (h *ContributionHandler) ReviewContribution(c *gin.Context) {
	reviewerID := helper.GetUserID(c)
	reviewerRole := helper.GetUserRole(c)

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, "无效的投稿ID")
		return
	}

	var req request.ReviewContributionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	err = h.contributionService.ReviewContribution(c, uint(id), reviewerID, models.UserRole(reviewerRole), &req)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "投稿审核完成"})
}

// GetUserContributionStats 获取用户投稿统计
func (h *ContributionHandler) GetUserContributionStats(c *gin.Context) {
	userID := helper.GetUserID(c)

	result, err := h.contributionService.GetUserContributionStats(c, userID)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessResponse(c, result)
}

// GetAdminContributionStats 获取管理员投稿统计（管理员和运营专用）
func (h *ContributionHandler) GetAdminContributionStats(c *gin.Context) {
	result, err := h.contributionService.GetAdminContributionStats(c)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessResponse(c, result)
}
