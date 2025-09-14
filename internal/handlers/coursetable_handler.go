package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"

	"github.com/gin-gonic/gin"
)

type CourseTableHandler struct {
	courseTableService *services.CourseTableService
}

func NewCourseTableHandler(courseTableService *services.CourseTableService) *CourseTableHandler {
	return &CourseTableHandler{
		courseTableService: courseTableService,
	}
}

// GetCourseTable 获取课程表
// @Summary 获取用户课程表
// @Description 根据当前用户ID和学期获取课程表数据
// @Tags 课程表
// @Accept json
// @Produce json
// @Param semester query string true "学期"
// @Success 200 {object} helper.Response{data=response.CourseTableResponse}
// @Failure 400 {object} helper.Response
// @Failure 401 {object} helper.Response
// @Router /api/v0/coursetable [get]
func (h *CourseTableHandler) GetCourseTable(c *gin.Context) {
	// 从上下文中获取用户ID（通过认证中间件设置）
	userID, exists := c.Get("user_id")
	if !exists {
		helper.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	var req request.GetCourseTableRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	result, err := h.courseTableService.GetUserCourseTable(userID.(uint), req.Semester)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.SuccessResponse(c, result)
}

// SearchClasses 搜索班级
// @Summary 模糊搜索班级
// @Description 根据关键字模糊搜索班级列表
// @Tags 课程表
// @Accept json
// @Produce json
// @Param keyword query string true "搜索关键字"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} helper.Response{data=response.SearchClassResponse}
// @Failure 400 {object} helper.Response
// @Router /api/v0/coursetable/search [get]
func (h *CourseTableHandler) SearchClasses(c *gin.Context) {
	var req request.SearchClassRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	// 设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 10
	}

	result, err := h.courseTableService.SearchClasses(req.Keyword, req.Page, req.Size)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.SuccessResponse(c, result)
}

// UpdateUserClass 更新用户班级
// @Summary 更新用户班级
// @Description 更新当前用户的班级信息
// @Tags 课程表
// @Accept json
// @Produce json
// @Param body body request.UpdateUserClassRequest true "更新班级请求"
// @Success 200 {object} helper.Response
// @Failure 400 {object} helper.Response
// @Failure 401 {object} helper.Response
// @Router /api/v0/coursetable/class [put]
func (h *CourseTableHandler) UpdateUserClass(c *gin.Context) {
	// 从上下文中获取用户ID（通过认证中间件设置）
	userID, exists := c.Get("user_id")
	if !exists {
		helper.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	var req request.UpdateUserClassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	err := h.courseTableService.UpdateUserClass(userID.(uint), req.ClassID)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.SuccessResponse(c, "班级信息更新成功")
}

// EditCourseCell 编辑用户个人课表中的单个格子
// @Summary 编辑用户个人课表中的单个格子
// @Description 前端发送 index 与 value，将该格子替换进完整课表JSON
// @Tags 课程表
// @Accept json
// @Produce json
// @Param body body request.EditCourseCellRequest true "编辑格子请求"
// @Success 200 {object} helper.Response
// @Failure 400 {object} helper.Response
// @Failure 401 {object} helper.Response
// @Router /api/v0/coursetable [put]
func (h *CourseTableHandler) EditCourseCell(c *gin.Context) {
	// 从上下文中获取用户ID（通过认证中间件设置）
	userID, exists := c.Get("user_id")
	if !exists {
		helper.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	var req request.EditCourseCellRequest
	// 检查 index 是否为 1-35 之间的字符串
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	indexInt, err := strconv.Atoi(req.Index)
	if err != nil || indexInt < 1 || indexInt > 35 {
		helper.ValidateResponse(c, "参数校验失败")
		return
	}

	// 将任意值编码为 JSON 原样传入服务层
	bytesValue, err := json.Marshal(req.Value)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, "无效的值数据")
		return
	}

	if err := h.courseTableService.EditUserCourseCell(userID.(uint), req.Semester, req.Index, bytesValue); err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	helper.SuccessResponse(c, "编辑成功")
}
