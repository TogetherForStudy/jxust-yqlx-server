package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"
)

type StatHandler struct {
	statService *services.StatService
}

func NewStatHandler(statService *services.StatService) *StatHandler {
	return &StatHandler{
		statService: statService,
	}
}

// GetSystemOnlineCount 获取系统在线人数
// @Summary 获取系统在线人数
// @Description 获取系统当前在线人数（TTL 1分钟）
// @Tags 统计
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} utils.Response{data=response.SystemOnlineStatResponse}
// @Failure 500 {object} utils.Response
// @Router /api/v0/stat/system/online [get]
func (h *StatHandler) GetSystemOnlineCount(c *gin.Context) {
	ctx := c.Request.Context()

	count, err := h.statService.GetSystemOnlineCount(ctx)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessResponse(c, response.SystemOnlineStatResponse{
		OnlineCount: count,
	})
}

// GetProjectOnlineCount 获取项目在线人数
// @Summary 获取项目在线人数
// @Description 获取指定项目的当前在线人数（TTL 1分钟）
// @Tags 统计
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param project_id path int true "项目ID"
// @Success 200 {object} utils.Response{data=response.ProjectOnlineStatResponse}
// @Failure 400 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v0/stat/project/:project_id/online [get]
func (h *StatHandler) GetProjectOnlineCount(c *gin.Context) {
	var req struct {
		ProjectID uint `uri:"project_id" binding:"required"`
	}
	if err := c.ShouldBindUri(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	ctx := c.Request.Context()

	// 获取当前用户ID（如果已登录）
	userID := helper.GetUserID(c)
	if userID == 0 {
		helper.ErrorResponse(c, http.StatusUnauthorized, "未获取到用户信息")
		return
	}

	count, err := h.statService.GetProjectOnlineCount(ctx, req.ProjectID, userID)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessResponse(c, response.ProjectOnlineStatResponse{
		ProjectID:   req.ProjectID,
		OnlineCount: count,
	})
}
