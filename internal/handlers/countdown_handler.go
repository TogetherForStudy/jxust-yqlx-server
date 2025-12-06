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

	var req request.CreateCountdownRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	result, err := h.countdownService.CreateCountdown(c, userID, &req)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.SuccessResponse(c, result)
}

// GetCountdowns 获取用户倒数日列表
func (h *CountdownHandler) GetCountdowns(c *gin.Context) {
	userID := helper.GetUserID(c)
	userRole := helper.GetUserRole(c)

	result, err := h.countdownService.GetCountdowns(c, userID, models.UserRole(userRole))
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessResponse(c, result)
}

// GetCountdownByID 获取倒数日详情
func (h *CountdownHandler) GetCountdownByID(c *gin.Context) {
	userID := helper.GetUserID(c)

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, "无效的倒数日ID")
		return
	}

	result, err := h.countdownService.GetCountdownByID(c, uint(id), userID)
	if err != nil {
		if err.Error() == "倒数日不存在或无权限访问" {
			helper.ErrorResponse(c, http.StatusNotFound, err.Error())
			return
		}
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessResponse(c, result)
}

// UpdateCountdown 更新倒数日
func (h *CountdownHandler) UpdateCountdown(c *gin.Context) {
	userID := helper.GetUserID(c)

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, "无效的倒数日ID")
		return
	}

	var req request.UpdateCountdownRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	result, err := h.countdownService.UpdateCountdown(c, uint(id), userID, &req)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.SuccessResponse(c, result)
}

// DeleteCountdown 删除倒数日
func (h *CountdownHandler) DeleteCountdown(c *gin.Context) {
	userID := helper.GetUserID(c)

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, "无效的倒数日ID")
		return
	}

	err = h.countdownService.DeleteCountdown(c, uint(id), userID)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "倒数日删除成功"})
}
