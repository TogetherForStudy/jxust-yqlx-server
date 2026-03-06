package handlers

import (
	"net/http"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"

	"github.com/gin-gonic/gin"
)

type UserActivityHandler struct {
	userActivityService *services.UserActivityService
}

func NewUserActivityHandler(userActivityService *services.UserActivityService) *UserActivityHandler {
	return &UserActivityHandler{
		userActivityService: userActivityService,
	}
}

// GetLoginDays 获取用户在过去100天内的登录天数
// @Summary 获取登录天数
// @Description 返回当前用户在过去100天内有多少天登录过系统
// @Tags 用户
// @Produce json
// @Success 200 {object} helper.Response{data=map[string]any}
// @Failure 401 {object} helper.Response
// @Router /api/v0/user/login-days [get]
func (h *UserActivityHandler) GetLoginDays(c *gin.Context) {
	userID := helper.GetUserID(c)
	if userID == 0 {
		helper.ErrorResponse(c, http.StatusUnauthorized, "未获取到用户信息")
		return
	}

	const pastDays = 100
	days, err := h.userActivityService.GetUserLoginDays(c, userID, pastDays)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, "查询登录天数失败")
		return
	}

	helper.SuccessResponse(c, gin.H{
		"past_days":  pastDays,
		"login_days": days,
	})
}
