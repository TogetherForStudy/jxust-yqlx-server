package handlers

import (
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"github.com/gin-gonic/gin"
)

func (h *FailRateHandler) AdminListFailRates(c *gin.Context) {
	var req request.AdminListFailRatesRequest
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

	items, total, err := h.service.ListAdminFailRates(c.Request.Context(), &req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.PageSuccessResponse(c, items, total, req.Page, req.PageSize)
}

func (h *FailRateHandler) AdminGetFailRateByID(c *gin.Context) {
	var req struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&req); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	result, err := h.service.GetAdminFailRateByID(c.Request.Context(), req.ID)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessResponse(c, result)
}

func (h *FailRateHandler) AdminCreateFailRate(c *gin.Context) {
	var req request.AdminCreateFailRateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	result, err := h.service.CreateAdminFailRate(c.Request.Context(), &req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessResponse(c, result)
}

func (h *FailRateHandler) AdminUpdateFailRate(c *gin.Context) {
	var uriReq struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uriReq); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	var req request.AdminUpdateFailRateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	result, err := h.service.UpdateAdminFailRate(c.Request.Context(), uriReq.ID, &req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessResponse(c, result)
}

func (h *FailRateHandler) AdminDeleteFailRate(c *gin.Context) {
	var req struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&req); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	if err := h.service.DeleteAdminFailRate(c.Request.Context(), req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessResponse(c, gin.H{"message": "删除成功"})
}
