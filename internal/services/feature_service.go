package services

import (
	"context"
	"fmt"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/apperr"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/cache"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"

	json "github.com/bytedance/sonic"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type FeatureService struct {
	db    *gorm.DB
	cache cache.Cache
}

func NewFeatureService(db *gorm.DB) *FeatureService {
	return &FeatureService{
		db:    db,
		cache: cache.GlobalCache,
	}
}

// DB 返回数据库实例（供Handler使用）
func (s *FeatureService) DB() *gorm.DB {
	return s.db
}

// CheckUserFeature 检查用户是否有指定功能权限（带缓存）
func (s *FeatureService) CheckUserFeature(ctx context.Context, userID uint, featureKey string) (bool, error) {
	enabled, err := s.IsFeatureEnabled(ctx, featureKey)
	if err != nil {
		return false, err
	}
	if !enabled {
		return false, nil
	}

	features, err := s.GetUserFeatures(ctx, userID)
	if err != nil {
		return false, err
	}

	for _, f := range features {
		if f == featureKey {
			return true, nil
		}
	}

	return false, nil
}

// GetUserFeatures 获取用户的所有可用功能列表（带缓存）
// 来源：① user_id IN user_ids  ② user_role IN role_ids（一条SQL搞定）
func (s *FeatureService) GetUserFeatures(ctx context.Context, userID uint) ([]string, error) {
	// 1. 尝试从缓存获取
	if s.cache != nil {
		cacheKey := fmt.Sprintf(constant.CacheKeyUserFeatures, userID)
		cachedData, err := s.cache.Get(ctx, cacheKey)
		if err == nil && cachedData != "" {
			var features []string
			if err := json.Unmarshal([]byte(cachedData), &features); err == nil {
				return features, nil
			}
		}
	}

	// 2. 获取用户角色ID列表
	var roleIDs []uint
	s.db.WithContext(ctx).
		Table("user_roles").
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Pluck("role_id", &roleIDs)

	// 3. 一条SQL：JSON_CONTAINS(user_ids, userID) OR JSON_OVERLAPS(role_ids, userRoleIDs)
	roleIDsJSON, _ := json.Marshal(roleIDs)

	var features []string
	err := s.db.WithContext(ctx).
		Table("features").
		Select("DISTINCT feature_key").
		Where("is_enabled = ? AND deleted_at IS NULL", true).
		Where(`(
			JSON_CONTAINS(user_ids, CAST(? AS JSON))
			OR
			JSON_OVERLAPS(role_ids, CAST(? AS JSON))
		)`, fmt.Sprintf("%d", userID), string(roleIDsJSON)).
		Pluck("feature_key", &features).Error

	if err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询用户功能列表失败: %w", err))
	}

	// 4. 缓存结果
	if s.cache != nil {
		cacheKey := fmt.Sprintf(constant.CacheKeyUserFeatures, userID)
		data, _ := json.Marshal(features)
		ttl := constant.UserFeaturesCacheTTL
		_ = s.cache.Set(ctx, cacheKey, string(data), &ttl)
	}

	return features, nil
}

// IsFeatureEnabled 检查功能是否全局启用（带缓存）
func (s *FeatureService) IsFeatureEnabled(ctx context.Context, featureKey string) (bool, error) {
	if s.cache != nil {
		cacheKey := fmt.Sprintf(constant.CacheKeyFeatureEnabled, featureKey)
		cachedData, err := s.cache.Get(ctx, cacheKey)
		if err == nil && cachedData != "" {
			return cachedData == "1", nil
		}
	}

	var feature models.Feature
	err := s.db.WithContext(ctx).
		Where("feature_key = ?", featureKey).
		First(&feature).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, apperr.Wrap(constant.CommonInternal, err)
	}

	if s.cache != nil {
		cacheKey := fmt.Sprintf(constant.CacheKeyFeatureEnabled, featureKey)
		value := "0"
		if feature.IsEnabled {
			value = "1"
		}
		ttl := constant.FeatureEnabledCacheTTL
		_ = s.cache.Set(ctx, cacheKey, value, &ttl)
	}

	return feature.IsEnabled, nil
}

// GrantFeatureToUser 授予用户功能权限（往 UserIDs JSON 数组添加 userID）
func (s *FeatureService) GrantFeatureToUser(ctx context.Context, userID uint, featureKey string) error {
	var feature models.Feature
	err := s.db.WithContext(ctx).Where("feature_key = ?", featureKey).First(&feature).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperr.New(constant.FeatureNotFound)
		}
		return apperr.Wrap(constant.CommonInternal, err)
	}

	var currentIDs []uint
	if feature.UserIDs != nil {
		_ = json.Unmarshal(feature.UserIDs, &currentIDs)
	}

	for _, id := range currentIDs {
		if id == userID {
			return nil // 已授权
		}
	}

	currentIDs = append(currentIDs, userID)
	newJSON, _ := json.Marshal(currentIDs)

	err = s.db.WithContext(ctx).
		Model(&models.Feature{}).
		Where("feature_key = ?", featureKey).
		Update("user_ids", datatypes.JSON(newJSON)).Error
	if err != nil {
		return apperr.Wrap(constant.CommonInternal, err)
	}

	s.clearUserFeaturesCache(ctx, userID)
	return nil
}

// BatchGrantFeatureToUsers 批量授予用户功能权限
func (s *FeatureService) BatchGrantFeatureToUsers(ctx context.Context, userIDs []uint, featureKey string) error {
	if len(userIDs) == 0 {
		return nil
	}

	var feature models.Feature
	err := s.db.WithContext(ctx).Where("feature_key = ?", featureKey).First(&feature).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperr.New(constant.FeatureNotFound)
		}
		return apperr.Wrap(constant.CommonInternal, err)
	}

	var currentIDs []uint
	if feature.UserIDs != nil {
		_ = json.Unmarshal(feature.UserIDs, &currentIDs)
	}

	idSet := make(map[uint]bool)
	for _, id := range currentIDs {
		idSet[id] = true
	}
	for _, id := range userIDs {
		idSet[id] = true
	}

	merged := make([]uint, 0, len(idSet))
	for id := range idSet {
		merged = append(merged, id)
	}

	newJSON, _ := json.Marshal(merged)

	err = s.db.WithContext(ctx).
		Model(&models.Feature{}).
		Where("feature_key = ?", featureKey).
		Update("user_ids", datatypes.JSON(newJSON)).Error
	if err != nil {
		return apperr.Wrap(constant.CommonInternal, err)
	}

	for _, uid := range userIDs {
		s.clearUserFeaturesCache(ctx, uid)
	}
	return nil
}

// RevokeFeatureFromUser 撤销用户功能权限（从 UserIDs JSON 数组移除）
func (s *FeatureService) RevokeFeatureFromUser(ctx context.Context, userID uint, featureKey string) error {
	var feature models.Feature
	err := s.db.WithContext(ctx).Where("feature_key = ?", featureKey).First(&feature).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperr.New(constant.FeatureNotFound)
		}
		return apperr.Wrap(constant.CommonInternal, err)
	}

	var currentIDs []uint
	if feature.UserIDs != nil {
		_ = json.Unmarshal(feature.UserIDs, &currentIDs)
	}

	filtered := make([]uint, 0, len(currentIDs))
	for _, id := range currentIDs {
		if id != userID {
			filtered = append(filtered, id)
		}
	}

	newJSON, _ := json.Marshal(filtered)

	err = s.db.WithContext(ctx).
		Model(&models.Feature{}).
		Where("feature_key = ?", featureKey).
		Update("user_ids", datatypes.JSON(newJSON)).Error
	if err != nil {
		return apperr.Wrap(constant.CommonInternal, err)
	}

	s.clearUserFeaturesCache(ctx, userID)
	return nil
}

// SetFeatureRoles 设置功能的授权角色ID列表
func (s *FeatureService) SetFeatureRoles(ctx context.Context, featureKey string, roleIDs []uint) error {
	roleIDsJSON, _ := json.Marshal(roleIDs)
	result := s.db.WithContext(ctx).
		Model(&models.Feature{}).
		Where("feature_key = ?", featureKey).
		Update("role_ids", datatypes.JSON(roleIDsJSON))

	if result.Error != nil {
		return apperr.Wrap(constant.CommonInternal, result.Error)
	}
	if result.RowsAffected == 0 {
		return apperr.New(constant.FeatureNotFound)
	}

	s.clearFeatureEnabledCache(ctx, featureKey)

	// 清除拥有这些角色的所有用户的 feature 缓存
	if s.cache != nil {
		var userIDs []uint
		s.db.WithContext(ctx).
			Table("user_roles").
			Where("role_id IN ? AND deleted_at IS NULL", roleIDs).
			Pluck("DISTINCT user_id", &userIDs)
		for _, uid := range userIDs {
			s.clearUserFeaturesCache(ctx, uid)
		}
	}

	return nil
}

// GrantRoleToFeature 给功能添加单个授权角色（追加到 role_ids）
func (s *FeatureService) GrantRoleToFeature(ctx context.Context, featureKey string, roleID uint) error {
	var feature models.Feature
	err := s.db.WithContext(ctx).Where("feature_key = ?", featureKey).First(&feature).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperr.New(constant.FeatureNotFound)
		}
		return apperr.Wrap(constant.CommonInternal, err)
	}

	var currentIDs []uint
	if feature.RoleIDs != nil {
		_ = json.Unmarshal(feature.RoleIDs, &currentIDs)
	}

	for _, id := range currentIDs {
		if id == roleID {
			return nil // 已存在
		}
	}

	currentIDs = append(currentIDs, roleID)
	newJSON, _ := json.Marshal(currentIDs)

	err = s.db.WithContext(ctx).
		Model(&models.Feature{}).
		Where("feature_key = ?", featureKey).
		Update("role_ids", datatypes.JSON(newJSON)).Error
	if err != nil {
		return apperr.Wrap(constant.CommonInternal, err)
	}

	// 清除拥有该角色的所有用户的缓存
	s.invalidateRoleUsersCache(ctx, roleID)
	return nil
}

// RevokeRoleFromFeature 从功能移除单个授权角色
func (s *FeatureService) RevokeRoleFromFeature(ctx context.Context, featureKey string, roleID uint) error {
	var feature models.Feature
	err := s.db.WithContext(ctx).Where("feature_key = ?", featureKey).First(&feature).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperr.New(constant.FeatureNotFound)
		}
		return apperr.Wrap(constant.CommonInternal, err)
	}

	var currentIDs []uint
	if feature.RoleIDs != nil {
		_ = json.Unmarshal(feature.RoleIDs, &currentIDs)
	}

	filtered := make([]uint, 0, len(currentIDs))
	for _, id := range currentIDs {
		if id != roleID {
			filtered = append(filtered, id)
		}
	}

	newJSON, _ := json.Marshal(filtered)

	err = s.db.WithContext(ctx).
		Model(&models.Feature{}).
		Where("feature_key = ?", featureKey).
		Update("role_ids", datatypes.JSON(newJSON)).Error
	if err != nil {
		return apperr.Wrap(constant.CommonInternal, err)
	}

	s.invalidateRoleUsersCache(ctx, roleID)
	return nil
}

func (s *FeatureService) invalidateRoleUsersCache(ctx context.Context, roleID uint) {
	if s.cache == nil {
		return
	}
	var userIDs []uint
	s.db.WithContext(ctx).
		Table("user_roles").
		Where("role_id = ? AND deleted_at IS NULL", roleID).
		Pluck("DISTINCT user_id", &userIDs)
	for _, uid := range userIDs {
		s.clearUserFeaturesCache(ctx, uid)
	}
}

// ListFeatures 获取所有功能列表
func (s *FeatureService) ListFeatures(ctx context.Context) ([]models.Feature, error) {
	var features []models.Feature
	err := s.db.WithContext(ctx).Order("created_at DESC").Find(&features).Error
	if err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, err)
	}
	return features, nil
}

// GetFeature 获取功能详情
func (s *FeatureService) GetFeature(ctx context.Context, featureKey string) (*models.Feature, error) {
	var feature models.Feature
	err := s.db.WithContext(ctx).Where("feature_key = ?", featureKey).First(&feature).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.New(constant.FeatureNotFound)
		}
		return nil, apperr.Wrap(constant.CommonInternal, err)
	}
	return &feature, nil
}

// CreateFeature 创建功能
func (s *FeatureService) CreateFeature(ctx context.Context, feature *models.Feature) error {
	var existing models.Feature
	err := s.db.WithContext(ctx).Where("feature_key = ?", feature.FeatureKey).First(&existing).Error
	if err == nil {
		return apperr.New(constant.FeatureIdentifierExists)
	}
	if err != gorm.ErrRecordNotFound {
		return apperr.Wrap(constant.CommonInternal, err)
	}

	// 确保 JSON 字段默认值为空数组
	if feature.UserIDs == nil {
		feature.UserIDs = datatypes.JSON([]byte("[]"))
	}
	if feature.RoleIDs == nil {
		feature.RoleIDs = datatypes.JSON([]byte("[]"))
	}

	err = s.db.WithContext(ctx).Create(feature).Error
	if err != nil {
		return apperr.Wrap(constant.CommonInternal, err)
	}

	s.clearFeatureEnabledCache(ctx, feature.FeatureKey)
	return nil
}

// UpdateFeature 更新功能
func (s *FeatureService) UpdateFeature(ctx context.Context, featureKey string, updates map[string]interface{}) error {
	result := s.db.WithContext(ctx).
		Model(&models.Feature{}).
		Where("feature_key = ?", featureKey).
		Updates(updates)

	if result.Error != nil {
		return apperr.Wrap(constant.CommonInternal, result.Error)
	}
	if result.RowsAffected == 0 {
		var count int64
		if err := s.db.WithContext(ctx).
			Model(&models.Feature{}).
			Where("feature_key = ?", featureKey).
			Count(&count).Error; err != nil {
			return apperr.Wrap(constant.CommonInternal, err)
		}
		if count == 0 {
			return apperr.New(constant.FeatureNotFound)
		}
	}

	s.clearFeatureEnabledCache(ctx, featureKey)
	return nil
}

// DeleteFeature 删除功能（软删除）
func (s *FeatureService) DeleteFeature(ctx context.Context, featureKey string) error {
	result := s.db.WithContext(ctx).
		Where("feature_key = ?", featureKey).
		Delete(&models.Feature{})

	if result.Error != nil {
		return apperr.Wrap(constant.CommonInternal, result.Error)
	}
	if result.RowsAffected == 0 {
		return apperr.New(constant.FeatureNotFound)
	}

	s.clearFeatureEnabledCache(ctx, featureKey)
	return nil
}

// GetEnabledProjectFeatures 获取所有启用的项目灰度 feature_key
// 查询 SELECT ... LIKE 'review.project.%' 走 B-tree 索引前缀匹配，~1ms，无需缓存
func (s *FeatureService) GetEnabledProjectFeatures(ctx context.Context) (map[string]bool, error) {
	var keys []string
	err := s.db.WithContext(ctx).
		Table("features").
		Where("feature_key LIKE ? AND is_enabled = ? AND deleted_at IS NULL", "review.project.%", true).
		Pluck("feature_key", &keys).Error
	if err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, err)
	}
	set := make(map[string]bool, len(keys))
	for _, k := range keys {
		set[k] = true
	}
	return set, nil
}

// GetUserIDsByFeature 查询某功能已授权的用户ID列表
func (s *FeatureService) GetUserIDsByFeature(ctx context.Context, featureKey string) ([]uint, error) {
	feature, err := s.GetFeature(ctx, featureKey)
	if err != nil {
		return nil, err
	}
	var ids []uint
	if feature.UserIDs != nil {
		_ = json.Unmarshal(feature.UserIDs, &ids)
	}
	return ids, nil
}

// GetRoleIDsByFeature 查询某功能已授权的角色ID列表
func (s *FeatureService) GetRoleIDsByFeature(ctx context.Context, featureKey string) ([]uint, error) {
	feature, err := s.GetFeature(ctx, featureKey)
	if err != nil {
		return nil, err
	}
	var ids []uint
	if feature.RoleIDs != nil {
		_ = json.Unmarshal(feature.RoleIDs, &ids)
	}
	return ids, nil
}

// ==================== 缓存管理 ====================

func (s *FeatureService) clearUserFeaturesCache(ctx context.Context, userID uint) {
	if s.cache != nil {
		cacheKey := fmt.Sprintf(constant.CacheKeyUserFeatures, userID)
		_ = s.cache.Delete(ctx, cacheKey)
	}
}

func (s *FeatureService) clearFeatureEnabledCache(ctx context.Context, featureKey string) {
	if s.cache != nil {
		cacheKey := fmt.Sprintf(constant.CacheKeyFeatureEnabled, featureKey)
		_ = s.cache.Delete(ctx, cacheKey)
	}
}
