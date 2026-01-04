package handlers

import (
	"net/http"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"

	"github.com/gin-gonic/gin"
)

type PointsHandler struct {
	pointsService *services.PointsService
}

func NewPointsHandler(pointsService *services.PointsService) *PointsHandler {
	return &PointsHandler{
		pointsService: pointsService,
	}
}

// GetUserPoints 获取用户积分信息
func (h *PointsHandler) GetUserPoints(c *gin.Context) {
	userID := helper.GetUserID(c)

	result, err := h.pointsService.GetUserPoints(c, userID)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessResponse(c, result)
}

// GetPointsTransactions 获取积分交易记录
func (h *PointsHandler) GetPointsTransactions(c *gin.Context) {
	userID := helper.GetUserID(c)

	var req request.GetPointsTransactionsRequest
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
	if req.Type != nil && *req.Type == 0 {
		req.Type = nil
	}
	if req.Source != nil && *req.Source == "" {
		req.Source = nil
	}
	if req.UserID != nil && *req.UserID == 0 {
		req.UserID = nil
	}

	result, err := h.pointsService.GetPointsTransactions(c, userID, &req)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessResponse(c, result)
}

// RedeemPoints 兑换积分
func (h *PointsHandler) SpendPoints(c *gin.Context) {
	userID := helper.GetUserID(c)

	var req request.SpendPointsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	err := h.pointsService.SpendPoints(c, userID, &req)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "积分消费成功"})
}

// GetUserPointsStats 获取用户积分统计
func (h *PointsHandler) GetUserPointsStats(c *gin.Context) {
	userID := helper.GetUserID(c)

	result, err := h.pointsService.GetUserPointsStats(c, userID)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessResponse(c, result)
}

// GrantPoints 管理员手动赋予积分
func (h *PointsHandler) GrantPoints(c *gin.Context) {
	var req request.GrantPointsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	err := h.pointsService.GrantPoints(c, req.UserID, req.Points, req.Description)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "积分操作成功"})
}
