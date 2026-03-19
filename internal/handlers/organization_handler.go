package handlers

import (
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"github.com/gin-gonic/gin"
)

type OrganizationHandler struct {
	service *services.OrganizationService
}

func NewOrganizationHandler(service *services.OrganizationService) *OrganizationHandler {
	return &OrganizationHandler{service: service}
}

func (h *OrganizationHandler) ListOrganizations(c *gin.Context) {
	var req request.ListOrganizationsRequest
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

	items, total, err := h.service.ListOrganizations(c.Request.Context(), &req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.PageSuccessResponse(c, items, total, req.Page, req.PageSize)
}

func (h *OrganizationHandler) GetOrganizationByID(c *gin.Context) {
	var req struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&req); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	item, err := h.service.GetOrganizationByID(c.Request.Context(), req.ID)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessResponse(c, item)
}

func (h *OrganizationHandler) AdminListOrganizations(c *gin.Context) {
	h.ListOrganizations(c)
}

func (h *OrganizationHandler) AdminGetOrganizationByID(c *gin.Context) {
	h.GetOrganizationByID(c)
}

func (h *OrganizationHandler) AdminCreateOrganization(c *gin.Context) {
	var req request.CreateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	item, err := h.service.CreateOrganization(c.Request.Context(), &req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessResponse(c, item)
}

func (h *OrganizationHandler) AdminUpdateOrganization(c *gin.Context) {
	var uriReq struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uriReq); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	var req request.UpdateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	item, err := h.service.UpdateOrganization(c.Request.Context(), uriReq.ID, &req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessResponse(c, item)
}

func (h *OrganizationHandler) AdminDeleteOrganization(c *gin.Context) {
	var req struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&req); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	if err := h.service.DeleteOrganization(c.Request.Context(), req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessResponse(c, gin.H{"message": "删除成功"})
}
