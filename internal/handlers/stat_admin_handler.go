package handlers

import (
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"github.com/gin-gonic/gin"
)

func (h *StatHandler) GetCountdownCountsByUser(c *gin.Context) {
	var req struct {
		Page     int `form:"page"`
		PageSize int `form:"page_size" binding:"min=1,max=100"`
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}

	items, total, err := h.statService.GetCountdownCountsByUser(c.Request.Context(), req.Page, req.PageSize)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.PageSuccessResponse(c, items, total, req.Page, req.PageSize)
}

func (h *StatHandler) GetStudyTaskCountsByUser(c *gin.Context) {
	var req struct {
		Page     int `form:"page"`
		PageSize int `form:"page_size" binding:"min=1,max=100"`
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}

	items, total, err := h.statService.GetStudyTaskCountsByUser(c.Request.Context(), req.Page, req.PageSize)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.PageSuccessResponse(c, items, total, req.Page, req.PageSize)
}

func (h *StatHandler) GetGPABackupCountsByUser(c *gin.Context) {
	var req struct {
		Page     int `form:"page"`
		PageSize int `form:"page_size" binding:"min=1,max=100"`
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}

	items, total, err := h.statService.GetGPABackupCountsByUser(c.Request.Context(), req.Page, req.PageSize)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.PageSuccessResponse(c, items, total, req.Page, req.PageSize)
}
