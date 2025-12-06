package handlers

import (
	"net/http"
	"strconv"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"

	"github.com/gin-gonic/gin"
)

type StudyTaskHandler struct {
	studyTaskService *services.StudyTaskService
}

func NewStudyTaskHandler(studyTaskService *services.StudyTaskService) *StudyTaskHandler {
	return &StudyTaskHandler{
		studyTaskService: studyTaskService,
	}
}

// CreateStudyTask 创建学习任务
func (h *StudyTaskHandler) CreateStudyTask(c *gin.Context) {
	userID := helper.GetUserID(c)

	var req request.CreateStudyTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	result, err := h.studyTaskService.CreateStudyTask(c, userID, &req)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.SuccessResponse(c, result)
}

// GetStudyTasks 获取用户学习任务列表
func (h *StudyTaskHandler) GetStudyTasks(c *gin.Context) {
	userID := helper.GetUserID(c)

	var req request.GetStudyTasksRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}
	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 20
	}

	// 处理查询参数：如果传递了0值，则设置为nil（表示不过滤）
	if req.Status != nil && *req.Status == 0 {
		req.Status = nil
	}
	if req.Priority != nil && *req.Priority == 0 {
		req.Priority = nil
	}

	result, err := h.studyTaskService.GetStudyTasks(c, userID, &req)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessResponse(c, result)
}

// GetStudyTaskByID 获取学习任务详情
func (h *StudyTaskHandler) GetStudyTaskByID(c *gin.Context) {
	userID := helper.GetUserID(c)

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, "无效的任务ID")
		return
	}

	result, err := h.studyTaskService.GetStudyTaskByID(c, uint(id), userID)
	if err != nil {
		if err.Error() == "学习任务不存在或无权限访问" {
			helper.ErrorResponse(c, http.StatusNotFound, err.Error())
			return
		}
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessResponse(c, result)
}

// UpdateStudyTask 更新学习任务
func (h *StudyTaskHandler) UpdateStudyTask(c *gin.Context) {
	userID := helper.GetUserID(c)

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, "无效的任务ID")
		return
	}

	var req request.UpdateStudyTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	result, err := h.studyTaskService.UpdateStudyTask(c, uint(id), userID, &req)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.SuccessResponse(c, result)
}

// DeleteStudyTask 删除学习任务
func (h *StudyTaskHandler) DeleteStudyTask(c *gin.Context) {
	userID := helper.GetUserID(c)

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, "无效的任务ID")
		return
	}

	err = h.studyTaskService.DeleteStudyTask(c, uint(id), userID)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "学习任务删除成功"})
}

// GetStudyTaskStats 获取用户学习任务统计
func (h *StudyTaskHandler) GetStudyTaskStats(c *gin.Context) {
	userID := helper.GetUserID(c)

	result, err := h.studyTaskService.GetStudyTaskStats(c, userID)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessResponse(c, result)
}

// GetCompletedTasks 获取已完成的任务（历史记录）
func (h *StudyTaskHandler) GetCompletedTasks(c *gin.Context) {
	userID := helper.GetUserID(c)

	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1
	}

	sizeStr := c.DefaultQuery("size", "20")
	size, err := strconv.Atoi(sizeStr)
	if err != nil || size <= 0 || size > 100 {
		size = 20
	}

	result, err := h.studyTaskService.GetCompletedTasks(c, userID, page, size)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessResponse(c, result)
}
