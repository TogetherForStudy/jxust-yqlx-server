package handlers

import (
	"strconv"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/apperr"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"

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
		helper.HandleErrCode(c, constant.AuthMissingUserContext)
		return
	}

	features, err := h.featureService.GetUserFeatures(c.Request.Context(), userID)
	if err != nil {
		helper.HandleError(c, err)
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
		helper.HandleError(c, err)
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
		helper.HandleError(c, err)
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
		helper.HandleError(c, apperr.Wrap(constant.CommonBadRequest, err))
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
		helper.HandleError(c, err)
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
		helper.HandleError(c, apperr.Wrap(constant.CommonBadRequest, err))
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
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	err := h.featureService.UpdateFeature(c.Request.Context(), featureKey, updates)
	if err != nil {
		helper.HandleError(c, err)
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
		helper.HandleError(c, err)
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "删除成功"})
}

// ListWhitelist 获取功能的白名单用户列表（管理员）
func (h *FeatureHandler) ListWhitelist(c *gin.Context) {
	featureKey := c.Param("key")

	userIDs, err := h.featureService.GetUserIDsByFeature(c.Request.Context(), featureKey)
	if err != nil {
		helper.HandleError(c, err)
		return
	}

	var users []models.User
	if len(userIDs) > 0 {
		h.featureService.DB().Find(&users, userIDs)
	}

	var result []response.WhitelistUserInfo
	for _, u := range users {
		result = append(result, response.WhitelistUserInfo{
			UserID:    u.ID,
			StudentID: u.StudentID,
			RealName:  u.RealName,
		})
	}

	if result == nil {
		result = []response.WhitelistUserInfo{}
	}
	total := int64(len(result))
	helper.PageSuccessResponse(c, result, total, 1, max(1, int(total)))
}

// GrantFeature 授予用户功能权限（管理员，支持单个 user_id 或批量 user_ids）
func (h *FeatureHandler) GrantFeature(c *gin.Context) {
	featureKey := c.Param("key")

	var req request.GrantFeatureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.HandleError(c, apperr.Wrap(constant.CommonBadRequest, err))
		return
	}

	if req.UserID != 0 {
		// 单个授权
		err := h.featureService.GrantFeatureToUser(c.Request.Context(), req.UserID, featureKey)
		if err != nil {
			helper.HandleError(c, err)
			return
		}
	} else if len(req.UserIDs) > 0 {
		// 批量授权
		err := h.featureService.BatchGrantFeatureToUsers(c.Request.Context(), req.UserIDs, featureKey)
		if err != nil {
			helper.HandleError(c, err)
			return
		}
	} else {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "授权成功"})
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
		helper.HandleError(c, apperr.Wrap(constant.CommonBadRequest, err))
		return
	}

	err = h.featureService.RevokeFeatureFromUser(c.Request.Context(), uint(userID), featureKey)
	if err != nil {
		helper.HandleError(c, err)
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "撤销成功"})
}

// GetUserFeatureDetails 获取用户的功能权限详情（管理员）
func (h *FeatureHandler) GetUserFeatureDetails(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		helper.HandleError(c, apperr.Wrap(constant.CommonBadRequest, err))
		return
	}

	// 查询用户拥有的 feature_key
	features, err := h.featureService.GetUserFeatures(c.Request.Context(), uint(userID))
	if err != nil {
		helper.HandleError(c, err)
		return
	}

	var result []response.UserFeatureInfo
	for _, fk := range features {
		feat, err := h.featureService.GetFeature(c.Request.Context(), fk)
		if err != nil {
			continue
		}
		result = append(result, response.UserFeatureInfo{
			FeatureKey:  feat.FeatureKey,
			FeatureName: feat.FeatureName,
		})
	}

	if result == nil {
		result = []response.UserFeatureInfo{}
	}
	helper.SuccessResponse(c, result)
}

// GrantRoleToFeature 给功能添加授权角色（管理员，支持单个 role_id 或批量 role_ids）
func (h *FeatureHandler) GrantRoleToFeature(c *gin.Context) {
	featureKey := c.Param("key")

	var req struct {
		RoleID  uint   `json:"role_id"`
		RoleIDs []uint `json:"role_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.HandleError(c, apperr.Wrap(constant.CommonBadRequest, err))
		return
	}

	if req.RoleID != 0 {
		err := h.featureService.GrantRoleToFeature(c.Request.Context(), featureKey, req.RoleID)
		if err != nil {
			helper.HandleError(c, err)
			return
		}
	} else if len(req.RoleIDs) > 0 {
		for _, rid := range req.RoleIDs {
			if err := h.featureService.GrantRoleToFeature(c.Request.Context(), featureKey, rid); err != nil {
				helper.HandleError(c, err)
				return
			}
		}
	} else {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "角色授权成功"})
}

// RevokeRoleFromFeature 从功能移除单个授权角色（管理员）
func (h *FeatureHandler) RevokeRoleFromFeature(c *gin.Context) {
	featureKey := c.Param("key")
	roleID, err := strconv.ParseUint(c.Param("rid"), 10, 32)
	if err != nil {
		helper.HandleError(c, apperr.Wrap(constant.CommonBadRequest, err))
		return
	}

	err = h.featureService.RevokeRoleFromFeature(c.Request.Context(), featureKey, uint(roleID))
	if err != nil {
		helper.HandleError(c, err)
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "角色授权撤销成功"})
}
