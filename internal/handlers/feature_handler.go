package handlers

import (
	"net/http"
	"strconv"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"

	"github.com/gin-gonic/gin"
)

type FeatureHandler struct {
	featureService *services.FeatureService
}

func NewFeatureHandler(featureService *services.FeatureService) *FeatureHandler {
	return &FeatureHandler{
		featureService: featureService,
	}
}

// GetUserFeatures 获取当前用户的可用功能列表
// @Summary 获取用户功能列表
// @Tags Feature
// @Accept json
// @Produce json
// @Success 200 {object} dto.Response{Result=response.UserFeaturesResponse}
// @Router /api/v0/user/features [get]
func (h *FeatureHandler) GetUserFeatures(c *gin.Context) {
	userID := helper.GetUserID(c)
	if userID == 0 {
		helper.ErrorResponse(c, http.StatusUnauthorized, "未获取到用户信息")
		return
	}

	features, err := h.featureService.GetUserFeatures(c.Request.Context(), userID)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, "获取功能列表失败")
		return
	}

	// 如果没有任何功能，返回空数组而不是nil
	if features == nil {
		features = []string{}
	}

	helper.SuccessResponse(c, response.UserFeaturesResponse{
		Features: features,
	})
}

// ListFeatures 获取所有功能列表（管理员）
// @Summary 获取所有功能列表
// @Tags Feature
// @Accept json
// @Produce json
// @Success 200 {object} dto.Response{Result=[]response.FeatureResponse}
// @Router /api/v0/admin/features [get]
func (h *FeatureHandler) ListFeatures(c *gin.Context) {
	features, err := h.featureService.ListFeatures(c.Request.Context())
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, "获取功能列表失败")
		return
	}

	// 转换为响应格式
	var result []response.FeatureResponse
	for _, f := range features {
		result = append(result, response.FeatureResponse{
			ID:          f.ID,
			FeatureKey:  f.FeatureKey,
			FeatureName: f.FeatureName,
			Description: f.Description,
			IsEnabled:   f.IsEnabled,
			CreatedAt:   f.CreatedAt,
			UpdatedAt:   f.UpdatedAt,
		})
	}

	helper.SuccessResponse(c, result)
}

// GetFeature 获取功能详情（管理员）
// @Summary 获取功能详情
// @Tags Feature
// @Accept json
// @Produce json
// @Param key path string true "功能标识"
// @Success 200 {object} dto.Response{Result=response.FeatureResponse}
// @Router /api/v0/admin/features/:key [get]
func (h *FeatureHandler) GetFeature(c *gin.Context) {
	featureKey := c.Param("key")

	feature, err := h.featureService.GetFeature(c, featureKey)
	if err != nil {
		helper.ErrorResponse(c, http.StatusNotFound, "功能不存在")
		return
	}

	helper.SuccessResponse(c, feature)
}

// CreateFeature 创建功能（管理员）
// @Summary 创建功能
// @Tags Feature
// @Accept json
// @Produce json
// @Param request body request.CreateFeatureRequest true "创建功能请求"
// @Success 200 {object} dto.Response{Result=response.FeatureResponse}
// @Router /api/v0/admin/features [post]
func (h *FeatureHandler) CreateFeature(c *gin.Context) {
	var req request.CreateFeatureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "请求参数错误: "+err.Error())
		return
	}

	// 设置默认值
	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	feature := &models.Feature{
		FeatureKey:  req.FeatureKey,
		FeatureName: req.FeatureName,
		Description: req.Description,
		IsEnabled:   isEnabled,
	}

	err := h.featureService.CreateFeature(c.Request.Context(), feature)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.SuccessResponse(c, feature)
}

// UpdateFeature 更新功能（管理员）
// @Summary 更新功能
// @Tags Feature
// @Accept json
// @Produce json
// @Param key path string true "功能标识"
// @Param request body request.UpdateFeatureRequest true "更新功能请求"
// @Success 200 {object} dto.Response
// @Router /api/v0/admin/features/:key [put]
func (h *FeatureHandler) UpdateFeature(c *gin.Context) {
	featureKey := c.Param("key")

	var req request.UpdateFeatureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "请求参数错误: "+err.Error())
		return
	}

	// 构建更新字段
	updates := make(map[string]interface{})
	if req.FeatureName != nil {
		updates["feature_name"] = *req.FeatureName
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.IsEnabled != nil {
		updates["is_enabled"] = *req.IsEnabled
	}

	if len(updates) == 0 {
		helper.ValidateResponse(c, "没有需要更新的字段")
		return
	}

	err := h.featureService.UpdateFeature(c.Request.Context(), featureKey, updates)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, "更新功能失败")
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "更新成功"})
}

// DeleteFeature 删除功能（管理员）
// @Summary 删除功能
// @Tags Feature
// @Accept json
// @Produce json
// @Param key path string true "功能标识"
// @Success 200 {object} dto.Response
// @Router /api/v0/admin/features/:key [delete]
func (h *FeatureHandler) DeleteFeature(c *gin.Context) {
	featureKey := c.Param("key")

	err := h.featureService.DeleteFeature(c.Request.Context(), featureKey)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, "删除功能失败")
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "删除成功"})
}

// ListWhitelist 获取功能的白名单用户列表（管理员）
// @Summary 获取功能白名单列表
// @Tags Feature
// @Accept json
// @Produce json
// @Param key path string true "功能标识"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} dto.Response{Result=response.PageResponse{Data=[]response.WhitelistUserInfo}}
// @Router /api/v0/admin/features/:key/whitelist [get]
func (h *FeatureHandler) ListWhitelist(c *gin.Context) {
	featureKey := c.Param("key")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	whitelists, total, err := h.featureService.ListWhitelist(c.Request.Context(), featureKey, page, pageSize)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, "获取白名单失败")
		return
	}

	// 查询用户信息
	var userIDs []uint
	for _, w := range whitelists {
		userIDs = append(userIDs, w.UserID)
	}

	var users []models.User
	if len(userIDs) > 0 {
		h.featureService.DB().Find(&users, userIDs)
	}

	// 构建用户信息map
	userMap := make(map[uint]models.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	// 转换为响应格式
	var result []response.WhitelistUserInfo
	for _, w := range whitelists {
		user := userMap[w.UserID]
		result = append(result, response.WhitelistUserInfo{
			ID:        w.ID,
			UserID:    w.UserID,
			StudentID: user.StudentID,
			RealName:  user.RealName,
			GrantedBy: w.GrantedBy,
			GrantedAt: w.GrantedAt,
			ExpiresAt: w.ExpiresAt,
			IsExpired: w.IsExpired(),
			CreatedAt: w.CreatedAt,
		})
	}

	helper.PageSuccessResponse(c, result, total, page, pageSize)
}

// GrantFeature 授予用户功能权限（管理员）
// @Summary 授予功能权限
// @Tags Feature
// @Accept json
// @Produce json
// @Param key path string true "功能标识"
// @Param request body request.GrantFeatureRequest true "授权请求"
// @Success 200 {object} dto.Response
// @Router /api/v0/admin/features/:key/whitelist [post]
func (h *FeatureHandler) GrantFeature(c *gin.Context) {
	featureKey := c.Param("key")
	adminID := helper.GetUserID(c)

	var req request.GrantFeatureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "请求参数错误: "+err.Error())
		return
	}

	err := h.featureService.GrantFeatureToUser(
		c.Request.Context(),
		req.UserID,
		adminID,
		featureKey,
		req.ExpiresAt,
	)

	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "授权成功"})
}

// BatchGrantFeature 批量授予用户功能权限（管理员）
// @Summary 批量授予功能权限
// @Tags Feature
// @Accept json
// @Produce json
// @Param key path string true "功能标识"
// @Param request body request.BatchGrantFeatureRequest true "批量授权请求"
// @Success 200 {object} dto.Response
// @Router /api/v0/admin/features/:key/whitelist/batch [post]
func (h *FeatureHandler) BatchGrantFeature(c *gin.Context) {
	featureKey := c.Param("key")
	adminID := helper.GetUserID(c)

	var req request.BatchGrantFeatureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "请求参数错误: "+err.Error())
		return
	}

	err := h.featureService.BatchGrantFeatureToUsers(
		c.Request.Context(),
		req.UserIDs,
		adminID,
		featureKey,
		req.ExpiresAt,
	)

	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "批量授权成功"})
}

// RevokeFeature 撤销用户功能权限（管理员）
// @Summary 撤销功能权限
// @Tags Feature
// @Accept json
// @Produce json
// @Param key path string true "功能标识"
// @Param uid path int true "用户ID"
// @Success 200 {object} dto.Response
// @Router /api/v0/admin/features/:key/whitelist/:uid [delete]
func (h *FeatureHandler) RevokeFeature(c *gin.Context) {
	featureKey := c.Param("key")
	userID, err := strconv.ParseUint(c.Param("uid"), 10, 32)
	if err != nil {
		helper.ValidateResponse(c, "无效的用户ID")
		return
	}

	err = h.featureService.RevokeFeatureFromUser(c.Request.Context(), uint(userID), featureKey)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, "撤销权限失败")
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "撤销成功"})
}

// GetUserFeatureDetails 获取用户的功能权限详情（管理员）
// @Summary 获取用户功能权限详情
// @Tags Feature
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} dto.Response{Result=[]response.UserFeatureInfo}
// @Router /api/v0/admin/users/:id/features [get]
func (h *FeatureHandler) GetUserFeatureDetails(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		helper.ValidateResponse(c, "无效的用户ID")
		return
	}

	whitelists, err := h.featureService.GetUserFeatureDetails(c.Request.Context(), uint(userID))
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, "获取用户功能详情失败")
		return
	}

	// 查询功能信息
	var featureKeys []string
	for _, w := range whitelists {
		featureKeys = append(featureKeys, w.FeatureKey)
	}

	var features []models.Feature
	if len(featureKeys) > 0 {
		h.featureService.DB().Where("feature_key IN ?", featureKeys).Find(&features)
	}

	// 构建功能信息map
	featureMap := make(map[string]models.Feature)
	for _, f := range features {
		featureMap[f.FeatureKey] = f
	}

	// 转换为响应格式
	var result []response.UserFeatureInfo
	for _, w := range whitelists {
		feature := featureMap[w.FeatureKey]
		result = append(result, response.UserFeatureInfo{
			FeatureKey:  w.FeatureKey,
			FeatureName: feature.FeatureName,
			GrantedBy:   w.GrantedBy,
			GrantedAt:   w.GrantedAt,
			ExpiresAt:   w.ExpiresAt,
			IsExpired:   w.IsExpired(),
		})
	}

	helper.SuccessResponse(c, result)
}
