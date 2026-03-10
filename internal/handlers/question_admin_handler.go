package handlers

import (
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"github.com/gin-gonic/gin"
)

func (h *QuestionHandler) AdminListQuestionProjects(c *gin.Context) {
	var req request.AdminListQuestionProjectsRequest
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

	items, total, err := h.questionService.ListAdminQuestionProjects(c.Request.Context(), &req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.PageSuccessResponse(c, items, total, req.Page, req.PageSize)
}

func (h *QuestionHandler) AdminGetQuestionProjectByID(c *gin.Context) {
	var req struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&req); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	result, err := h.questionService.GetAdminQuestionProjectByID(c.Request.Context(), req.ID)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessResponse(c, result)
}

func (h *QuestionHandler) AdminCreateQuestionProject(c *gin.Context) {
	var req request.AdminCreateQuestionProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	result, err := h.questionService.CreateAdminQuestionProject(c.Request.Context(), &req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessResponse(c, result)
}

func (h *QuestionHandler) AdminUpdateQuestionProject(c *gin.Context) {
	var uriReq struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uriReq); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	var req request.AdminUpdateQuestionProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	result, err := h.questionService.UpdateAdminQuestionProject(c.Request.Context(), uriReq.ID, &req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessResponse(c, result)
}

func (h *QuestionHandler) AdminDeleteQuestionProject(c *gin.Context) {
	var req struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&req); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	if err := h.questionService.DeleteAdminQuestionProject(c.Request.Context(), req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessResponse(c, gin.H{"message": "删除成功"})
}

func (h *QuestionHandler) AdminListQuestions(c *gin.Context) {
	var req request.AdminListQuestionsRequest
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

	items, total, err := h.questionService.ListAdminQuestions(c.Request.Context(), &req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.PageSuccessResponse(c, items, total, req.Page, req.PageSize)
}

func (h *QuestionHandler) AdminGetQuestionByID(c *gin.Context) {
	var req struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&req); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	result, err := h.questionService.GetAdminQuestionByID(c.Request.Context(), req.ID)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessResponse(c, result)
}

func (h *QuestionHandler) AdminCreateQuestion(c *gin.Context) {
	var req request.AdminCreateQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	result, err := h.questionService.CreateAdminQuestion(c.Request.Context(), &req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessResponse(c, result)
}

func (h *QuestionHandler) AdminUpdateQuestion(c *gin.Context) {
	var uriReq struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uriReq); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	var req request.AdminUpdateQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	result, err := h.questionService.UpdateAdminQuestion(c.Request.Context(), uriReq.ID, &req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessResponse(c, result)
}

func (h *QuestionHandler) AdminDeleteQuestion(c *gin.Context) {
	var req struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&req); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	if err := h.questionService.DeleteAdminQuestion(c.Request.Context(), req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessResponse(c, gin.H{"message": "删除成功"})
}
