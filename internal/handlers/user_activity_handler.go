package handlers

import (
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"

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
		helper.HandleErrCode(c, constant.AuthMissingUserContext)
		return
	}

	const pastDays = 100
	days, err := h.userActivityService.GetUserLoginDays(c, userID, pastDays)
	if err != nil {
		helper.HandleError(c, err)
		return
	}

	helper.SuccessResponse(c, gin.H{
		"past_days":  pastDays,
		"login_days": days,
	})
}
