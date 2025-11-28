package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"
)

type QuestionHandler struct {
	questionService *services.QuestionService
}

func NewQuestionHandler(questionService *services.QuestionService) *QuestionHandler {
	return &QuestionHandler{
		questionService: questionService,
	}
}

// ===================== 用户接口 =====================

// GetProjects 获取项目列表
// @Summary 获取项目列表
// @Description 获取所有启用的项目列表（含用户使用记录）
// @Tags 刷题
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} utils.Response{data=[]response.QuestionProjectResponse}
// @Failure 401 {object} utils.Response
// @Router /api/v0/questions/projects [get]
func (h *QuestionHandler) GetProjects(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		helper.ErrorResponse(c, http.StatusUnauthorized, "未获取到用户信息")
		return
	}

	projects, err := h.questionService.GetProjects(userID.(uint))
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, "获取项目列表失败")
		return
	}

	helper.SuccessResponse(c, projects)
}

// GetQuestions 获取题目列表
// @Summary 获取题目列表
// @Description 获取项目下所有题目的ID数组（顺序或乱序），只返回ID，不返回完整信息
// @Tags 刷题
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param project_id query int true "项目ID"
// @Param random query bool false "是否乱序" default(false)
// @Success 200 {object} utils.Response{data=response.QuestionListResponse}
// @Failure 400 {object} utils.Response
// @Router /api/v0/questions/list [get]
func (h *QuestionHandler) GetQuestions(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		helper.ErrorResponse(c, http.StatusUnauthorized, "未获取到用户信息")
		return
	}

	var req request.GetQuestionRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	result, err := h.questionService.GetQuestions(userID.(uint), &req)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.SuccessResponse(c, result)
}

// GetQuestionByID 获取题目详情
// @Summary 获取题目详情
// @Description 根据题目ID获取题目的完整信息（包含答案、选项、子题等）
// @Tags 刷题
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "题目ID"
// @Success 200 {object} utils.Response{data=response.QuestionResponse}
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /api/v0/questions/:id [get]
func (h *QuestionHandler) GetQuestionByID(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		helper.ErrorResponse(c, http.StatusUnauthorized, "未获取到用户信息")
		return
	}

	var req struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	question, err := h.questionService.GetQuestionByID(userID.(uint), req.ID)
	if err != nil {
		helper.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	helper.SuccessResponse(c, question)
}

// RecordStudy 记录学习
// @Summary 记录学习
// @Description 记录用户学习某题（仅记录次数）
// @Tags 刷题
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param body body request.RecordStudyRequest true "学习记录"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Router /api/v0/questions/study [post]
func (h *QuestionHandler) RecordStudy(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		helper.ErrorResponse(c, http.StatusUnauthorized, "未获取到用户信息")
		return
	}

	var req request.RecordStudyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	if err := h.questionService.RecordStudy(userID.(uint), &req); err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.SuccessResponse(c, nil)
}

// SubmitPractice 提交做题
// @Summary 提交做题
// @Description 记录用户做题（仅记录次数）
// @Tags 刷题
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param body body request.SubmitPracticeRequest true "做题记录"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Router /api/v0/questions/practice [post]
func (h *QuestionHandler) SubmitPractice(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		helper.ErrorResponse(c, http.StatusUnauthorized, "未获取到用户信息")
		return
	}

	var req request.SubmitPracticeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	if err := h.questionService.SubmitPractice(userID.(uint), &req); err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.SuccessResponse(c, nil)
}
