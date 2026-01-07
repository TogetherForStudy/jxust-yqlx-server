package handlers

import (
	"net/http"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"

	"github.com/gin-gonic/gin"
)

type MaterialHandler struct {
	materialService *services.MaterialService
}

func NewMaterialHandler(materialService *services.MaterialService) *MaterialHandler {
	return &MaterialHandler{
		materialService: materialService,
	}
}

// ==================== 资料管理接口 ====================

// GetMaterialList 获取资料列表
// @Summary 获取资料列表
// @Description 获取资料列表，支持按分类筛选和排序
// @Tags 资料管理
// @Accept json
// @Produce json
// @Param category_id query uint false "分类ID"
// @Param page query int true "页码" default(1)
// @Param page_size query int true "每页数量" default(20)
// @Param sort_by query string false "排序方式" Enums(hotness, time) default(hotness)
// @Success 200 {object} helper.PageResponse{data=[]response.MaterialListResponse}
// @Failure 400 {object} helper.Response
// @Router /api/materials [get]
func (h *MaterialHandler) GetMaterialList(c *gin.Context) {
	var req request.MaterialListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	// 设置默认值
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}
	if req.SortBy == "" {
		req.SortBy = "hotness"
	}

	materials, total, err := h.materialService.GetMaterialList(&req)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.PageSuccessResponse(c, materials, total, req.Page, req.PageSize)
}

// GetMaterialDetail 获取资料详情
// @Summary 获取资料详情
// @Description 根据MD5获取资料详情
// @Tags 资料管理
// @Accept json
// @Produce json
// @Param md5 path string true "资料MD5"
// @Success 200 {object} helper.Response{data=response.MaterialDetailResponse}
// @Failure 404 {object} helper.Response
// @Router /api/materials/{md5} [get]
func (h *MaterialHandler) GetMaterialDetail(c *gin.Context) {
	md5 := c.Param("md5")
	if md5 == "" {
		helper.ErrorResponse(c, http.StatusBadRequest, "MD5参数不能为空")
		return
	}

	// 获取用户ID（可选，未登录用户也可以查看）
	var userIDPtr *uint
	if userID, exists := c.Get("user_id"); exists {
		uid := userID.(uint)
		userIDPtr = &uid
	}

	detail, err := h.materialService.GetMaterialByMD5(md5, userIDPtr)
	if err != nil {
		helper.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	// 记录查看日志
	if userIDPtr != nil {
		logReq := &request.MaterialLogCreateRequest{
			Type:        2, // 查看
			MaterialMD5: md5,
		}
		if err := h.materialService.CreateMaterialLog(*userIDPtr, logReq); err != nil {
			helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
			return
		}
	}

	helper.SuccessResponse(c, detail)
}

// DeleteMaterial 删除资料（管理员）
// @Summary 删除资料
// @Description 管理员删除资料
// @Tags 资料管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param md5 path string true "资料MD5"
// @Success 200 {object} helper.Response
// @Failure 403 {object} helper.Response
// @Router /api/admin/materials/{md5} [delete]
func (h *MaterialHandler) DeleteMaterial(c *gin.Context) {
	md5 := c.Param("md5")
	if md5 == "" {
		helper.ErrorResponse(c, http.StatusBadRequest, "MD5参数不能为空")
		return
	}

	if err := h.materialService.DeleteMaterial(md5); err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "删除成功"})
}

// SearchMaterials 搜索资料
// @Summary 搜索资料
// @Description 根据关键词搜索资料
// @Tags 资料管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param keywords query string true "搜索关键词"
// @Param page query int true "页码" default(1)
// @Param page_size query int true "每页数量" default(20)
// @Success 200 {object} helper.Response{data=response.MaterialSearchResponse}
// @Failure 400 {object} helper.Response
// @Router /api/materials/search [get]
func (h *MaterialHandler) SearchMaterials(c *gin.Context) {
	var req request.MaterialSearchRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	// 设置默认值
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	result, err := h.materialService.SearchMaterials(&req)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 记录搜索日志
	if userID, exists := c.Get("user_id"); exists {
		logReq := &request.MaterialLogCreateRequest{
			Type:     1, // 搜索
			Keywords: req.Keywords,
		}
		if err := h.materialService.CreateMaterialLog(userID.(uint), logReq); err != nil {
			helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
			return
		}
	}

	helper.SuccessResponse(c, result)
}

// GetTopMaterials 获取热门资料
// @Summary 获取热门资料
// @Description 获取TOP热门资料
// @Tags 资料管理
// @Accept json
// @Produce json
// @Param limit query int false "数量限制" default(10)
// @Param type query int false "热度类型：0=全部热度，7=期间热度(7天)" default(0) Enums(0, 7)
// @Success 200 {object} helper.Response{data=[]response.MaterialListResponse}
// @Failure 400 {object} helper.Response
// @Router /api/materials/top [get]
func (h *MaterialHandler) GetTopMaterials(c *gin.Context) {
	var req request.TopMaterialsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	if req.Limit == 0 {
		req.Limit = 10
	}

	materials, err := h.materialService.GetTopMaterials(&req)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessResponse(c, materials)
}

// RateMaterial 资料评分
// @Summary 资料评分
// @Description 用户为资料评分
// @Tags 资料管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param md5 path string true "资料MD5"
// @Param body body request.MaterialRatingRequest true "评分请求"
// @Success 200 {object} helper.Response
// @Failure 400 {object} helper.Response
// @Router /api/materials/{md5}/rating [post]
func (h *MaterialHandler) RateMaterial(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		helper.ErrorResponse(c, http.StatusUnauthorized, "未获取到用户信息")
		return
	}

	md5 := c.Param("md5")
	if md5 == "" {
		helper.ErrorResponse(c, http.StatusBadRequest, "MD5参数不能为空")
		return
	}

	var req request.MaterialRatingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	if err := h.materialService.RateMaterial(userID.(uint), md5, req.Rating); err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "评分成功"})
}

// ==================== 资料描述管理接口 ====================

// UpdateMaterialDesc 更新资料描述（管理员）
// @Summary 更新资料描述
// @Description 管理员更新资料描述信息
// @Tags 资料描述管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param md5 path string true "资料MD5"
// @Param body body request.MaterialDescUpdateRequest true "资料描述更新请求"
// @Success 200 {object} helper.Response
// @Failure 400 {object} helper.Response
// @Router /api/admin/material-desc/{md5} [put]
func (h *MaterialHandler) UpdateMaterialDesc(c *gin.Context) {
	md5 := c.Param("md5")
	if md5 == "" {
		helper.ErrorResponse(c, http.StatusBadRequest, "MD5参数不能为空")
		return
	}

	var req request.MaterialDescUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	if err := h.materialService.UpdateMaterialDesc(md5, &req); err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "更新成功"})
}

// ==================== 分类管理接口 ====================

// GetCategories 获取分类列表
// @Summary 获取分类列表
// @Description 根据上级分类ID获取分类列表
// @Tags 分类管理
// @Accept json
// @Produce json
// @Param parent_id query uint false "上级分类ID"
// @Success 200 {object} helper.Response{data=[]response.MaterialCategoryResponse}
// @Failure 400 {object} helper.Response
// @Router /api/material-categories [get]
func (h *MaterialHandler) GetCategories(c *gin.Context) {
	var req request.MaterialCategoryListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	categories, err := h.materialService.GetCategoriesByParent(req.ParentID)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessResponse(c, categories)
}

// ==================== 其他接口 ====================

// GetHotWords 获取搜索热词
// @Summary 获取搜索热词
// @Description 获取搜索热词统计
// @Tags 资料管理
// @Accept json
// @Produce json
// @Param limit query int false "数量限制" default(20)
// @Success 200 {object} helper.Response{data=[]response.HotWordsResponse}
// @Failure 400 {object} helper.Response
// @Router /api/materials/hot-words [get]
func (h *MaterialHandler) GetHotWords(c *gin.Context) {
	var req request.HotWordsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	if req.Limit == 0 {
		req.Limit = 20
	}

	hotWords, err := h.materialService.GetHotWords(req.Limit)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessResponse(c, hotWords)
}

// DownloadMaterial 下载资料
// @Summary 下载资料
// @Description 记录用户下载资料的行为
// @Tags 资料管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param md5 path string true "资料MD5"
// @Success 200 {object} helper.Response
// @Failure 400 {object} helper.Response
// @Router /api/materials/{md5}/download [post]
func (h *MaterialHandler) DownloadMaterial(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		helper.ErrorResponse(c, http.StatusUnauthorized, "未获取到用户信息")
		return
	}

	md5 := c.Param("md5")
	if md5 == "" {
		helper.ErrorResponse(c, http.StatusBadRequest, "MD5参数不能为空")
		return
	}

	// 记录下载日志
	logReq := &request.MaterialLogCreateRequest{
		Type:        4, // 下载
		MaterialMD5: md5,
	}

	if err := h.materialService.CreateMaterialLog(userID.(uint), logReq); err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "下载记录成功"})
}
