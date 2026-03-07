package handlers

import (
	"strconv"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"

	"github.com/gin-gonic/gin"
)

type CountdownHandler struct {
	countdownService *services.CountdownService
}

func NewCountdownHandler(countdownService *services.CountdownService) *CountdownHandler {
	return &CountdownHandler{
		countdownService: countdownService,
	}
}

// CreateCountdown 创建倒数日
func (h *CountdownHandler) CreateCountdown(c *gin.Context) {
	userID := helper.GetUserID(c)
	if userID == 0 {
		helper.HandleErrCode(c, constant.AuthMissingUserContext)
		return
	}

	var req request.CreateCountdownRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	result, err := h.countdownService.CreateCountdown(c, userID, &req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}

	helper.SuccessResponse(c, result)
}

// GetCountdowns 获取用户倒数日列表
func (h *CountdownHandler) GetCountdowns(c *gin.Context) {
	userID := helper.GetUserID(c)
	if userID == 0 {
		helper.HandleErrCode(c, constant.AuthMissingUserContext)
		return
	}

	result, err := h.countdownService.GetCountdowns(c, userID)
	if err != nil {
		helper.HandleError(c, err)
		return
	}

	helper.SuccessResponse(c, result)
}

// GetCountdownByID 获取倒数日详情
func (h *CountdownHandler) GetCountdownByID(c *gin.Context) {
	userID := helper.GetUserID(c)
	if userID == 0 {
		helper.HandleErrCode(c, constant.AuthMissingUserContext)
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	result, err := h.countdownService.GetCountdownByID(c, uint(id), userID)
	if err != nil {
		helper.HandleError(c, err)
		return
	}

	helper.SuccessResponse(c, result)
}

// UpdateCountdown 更新倒数日
func (h *CountdownHandler) UpdateCountdown(c *gin.Context) {
	userID := helper.GetUserID(c)
	if userID == 0 {
		helper.HandleErrCode(c, constant.AuthMissingUserContext)
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	var req request.UpdateCountdownRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	result, err := h.countdownService.UpdateCountdown(c, uint(id), userID, &req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}

	helper.SuccessResponse(c, result)
}

// DeleteCountdown 删除倒数日
func (h *CountdownHandler) DeleteCountdown(c *gin.Context) {
	userID := helper.GetUserID(c)
	if userID == 0 {
		helper.HandleErrCode(c, constant.AuthMissingUserContext)
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	err = h.countdownService.DeleteCountdown(c, uint(id), userID)
	if err != nil {
		helper.HandleError(c, err)
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "倒数日删除成功"})
}
