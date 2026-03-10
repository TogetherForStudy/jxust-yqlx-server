package handlers

import (
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"github.com/gin-gonic/gin"
)

func (h *CourseTableHandler) AdminListCourseTables(c *gin.Context) {
	var req request.AdminListCourseTablesRequest
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

	items, total, err := h.courseTableService.ListAdminCourseTables(c.Request.Context(), &req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.PageSuccessResponse(c, items, total, req.Page, req.PageSize)
}

func (h *CourseTableHandler) AdminGetCourseTableByID(c *gin.Context) {
	var req struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&req); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	result, err := h.courseTableService.GetAdminCourseTableByID(c.Request.Context(), req.ID)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessResponse(c, result)
}

func (h *CourseTableHandler) AdminCreateCourseTable(c *gin.Context) {
	var req request.AdminCreateCourseTableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	result, err := h.courseTableService.CreateAdminCourseTable(c.Request.Context(), &req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessResponse(c, result)
}

func (h *CourseTableHandler) AdminUpdateCourseTable(c *gin.Context) {
	var uriReq struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uriReq); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	var req request.AdminUpdateCourseTableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	result, err := h.courseTableService.UpdateAdminCourseTable(c.Request.Context(), uriReq.ID, &req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessResponse(c, result)
}

func (h *CourseTableHandler) AdminDeleteCourseTable(c *gin.Context) {
	var req struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&req); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	if err := h.courseTableService.DeleteAdminCourseTable(c.Request.Context(), req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessResponse(c, gin.H{"message": "删除成功"})
}

func (h *CourseTableHandler) ResetUserBindCount(c *gin.Context) {
	var req struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&req); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	if err := h.courseTableService.ResetUserBindCount(c.Request.Context(), req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessResponse(c, gin.H{"message": "重置成功"})
}
