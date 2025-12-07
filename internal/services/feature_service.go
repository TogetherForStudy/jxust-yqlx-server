package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/cache"

	"gorm.io/gorm"
)

const (
	// 缓存Key前缀
	cacheKeyUserFeatures   = "user_features:%d"   // 用户功能列表缓存
	cacheKeyFeatureEnabled = "feature_enabled:%s" // 功能全局开关缓存

	// 缓存过期时间
	userFeaturesCacheTTL   = 5 * time.Minute  // 用户功能列表缓存5分钟
	featureEnabledCacheTTL = 10 * time.Minute // 功能开关缓存10分钟
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
	// 1. 检查功能是否全局启用（带缓存）
	enabled, err := s.isFeatureEnabled(ctx, featureKey)
	if err != nil {
		return false, err
	}
	if !enabled {
		return false, nil
	}

	// 2. 检查用户是否在白名单中（带缓存）
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
func (s *FeatureService) GetUserFeatures(ctx context.Context, userID uint) ([]string, error) {
	// 1. 尝试从缓存获取
	if s.cache != nil {
		cacheKey := fmt.Sprintf(cacheKeyUserFeatures, userID)
		cachedData, err := s.cache.Get(ctx, cacheKey)
		if err == nil && cachedData != "" {
			var features []string
			if err := json.Unmarshal([]byte(cachedData), &features); err == nil {
				return features, nil
			}
		}
	}

	// 2. 从数据库查询用户的启用功能（通过JOIN优化）
	var features []string
	now := time.Now()

	err := s.db.WithContext(ctx).
		Table("user_feature_whitelists").
		Select("DISTINCT user_feature_whitelists.feature_key").
		Joins("INNER JOIN features ON user_feature_whitelists.feature_key = features.feature_key").
		Where("user_feature_whitelists.user_id = ? AND (user_feature_whitelists.expires_at IS NULL OR user_feature_whitelists.expires_at > ?) AND features.is_enabled = ?", userID, now, true).
		Pluck("feature_key", &features).Error

	if err != nil {
		return nil, err
	}

	// 5. 缓存结果
	if s.cache != nil {
		cacheKey := fmt.Sprintf(cacheKeyUserFeatures, userID)
		data, _ := json.Marshal(features)
		ttl := userFeaturesCacheTTL
		_ = s.cache.Set(ctx, cacheKey, string(data), &ttl)
	}

	return features, nil
}

// isFeatureEnabled 检查功能是否全局启用（带缓存）
func (s *FeatureService) isFeatureEnabled(ctx context.Context, featureKey string) (bool, error) {
	// 1. 尝试从缓存获取
	if s.cache != nil {
		cacheKey := fmt.Sprintf(cacheKeyFeatureEnabled, featureKey)
		cachedData, err := s.cache.Get(ctx, cacheKey)
		if err == nil && cachedData != "" {
			return cachedData == "1", nil
		}
	}

	// 2. 从数据库查询
	var feature models.Feature
	err := s.db.WithContext(ctx).
		Where("feature_key = ?", featureKey).
		First(&feature).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, err
	}

	// 3. 缓存结果
	if s.cache != nil {
		cacheKey := fmt.Sprintf(cacheKeyFeatureEnabled, featureKey)
		value := "0"
		if feature.IsEnabled {
			value = "1"
		}
		ttl := featureEnabledCacheTTL
		_ = s.cache.Set(ctx, cacheKey, value, &ttl)
	}

	return feature.IsEnabled, nil
}

// GrantFeatureToUser 授予用户功能权限
func (s *FeatureService) GrantFeatureToUser(ctx context.Context, userID, grantedBy uint, featureKey string, expiresAt *time.Time) error {
	// 1. 检查功能是否存在
	var feature models.Feature
	err := s.db.WithContext(ctx).Where("feature_key = ?", featureKey).First(&feature).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("功能不存在")
		}
		return err
	}

	// 2. 检查用户是否存在
	var user models.User
	err = s.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("用户不存在")
		}
		return err
	}

	// 3. 创建或更新白名单记录
	whitelist := models.UserFeatureWhitelist{
		UserID:     userID,
		FeatureKey: featureKey,
		GrantedBy:  grantedBy,
		GrantedAt:  time.Now(),
		ExpiresAt:  expiresAt,
	}

	err = s.db.WithContext(ctx).
		Where("user_id = ? AND feature_key = ?", userID, featureKey).
		Assign(whitelist).
		FirstOrCreate(&whitelist).Error

	if err != nil {
		return err
	}

	// 4. 清除用户功能缓存
	s.clearUserFeaturesCache(ctx, userID)

	return nil
}

// BatchGrantFeatureToUsers 批量授予用户功能权限
func (s *FeatureService) BatchGrantFeatureToUsers(ctx context.Context, userIDs []uint, grantedBy uint, featureKey string, expiresAt *time.Time) error {
	if len(userIDs) == 0 {
		return nil
	}

	// 1. 检查功能是否存在
	var feature models.Feature
	err := s.db.WithContext(ctx).Where("feature_key = ?", featureKey).First(&feature).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("功能不存在")
		}
		return err
	}

	// 2. 批量授予权限
	for _, userID := range userIDs {
		err := s.GrantFeatureToUser(ctx, userID, grantedBy, featureKey, expiresAt)
		if err != nil {
			// 记录错误但继续处理其他用户
			continue
		}
	}

	return nil
}

// RevokeFeatureFromUser 撤销用户功能权限
func (s *FeatureService) RevokeFeatureFromUser(ctx context.Context, userID uint, featureKey string) error {
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND feature_key = ?", userID, featureKey).
		Delete(&models.UserFeatureWhitelist{}).Error

	if err != nil {
		return err
	}

	// 清除用户功能缓存
	s.clearUserFeaturesCache(ctx, userID)

	return nil
}

// ListFeatures 获取所有功能列表
func (s *FeatureService) ListFeatures(ctx context.Context) ([]models.Feature, error) {
	var features []models.Feature
	err := s.db.WithContext(ctx).Order("created_at DESC").Find(&features).Error
	return features, err
}

// GetFeature 获取功能详情
func (s *FeatureService) GetFeature(ctx context.Context, featureKey string) (*models.Feature, error) {
	var feature models.Feature
	err := s.db.WithContext(ctx).Where("feature_key = ?", featureKey).First(&feature).Error
	if err != nil {
		return nil, err
	}
	return &feature, nil
}

// CreateFeature 创建功能
func (s *FeatureService) CreateFeature(ctx context.Context, feature *models.Feature) error {
	// 检查功能是否已存在
	var existing models.Feature
	err := s.db.WithContext(ctx).Where("feature_key = ?", feature.FeatureKey).First(&existing).Error
	if err == nil {
		return fmt.Errorf("功能标识已存在")
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}

	err = s.db.WithContext(ctx).Create(feature).Error
	if err != nil {
		return err
	}

	// 清除功能缓存
	s.clearFeatureEnabledCache(ctx, feature.FeatureKey)

	return nil
}

// UpdateFeature 更新功能
func (s *FeatureService) UpdateFeature(ctx context.Context, featureKey string, updates map[string]interface{}) error {
	err := s.db.WithContext(ctx).
		Model(&models.Feature{}).
		Where("feature_key = ?", featureKey).
		Updates(updates).Error

	if err != nil {
		return err
	}

	// 清除功能缓存
	s.clearFeatureEnabledCache(ctx, featureKey)

	return nil
}

// DeleteFeature 删除功能（软删除）
func (s *FeatureService) DeleteFeature(ctx context.Context, featureKey string) error {
	err := s.db.WithContext(ctx).
		Where("feature_key = ?", featureKey).
		Delete(&models.Feature{}).Error

	if err != nil {
		return err
	}

	// 清除相关缓存
	s.clearFeatureEnabledCache(ctx, featureKey)
	// 清除所有拥有该功能的用户缓存
	s.clearAllUsersCacheForFeature(ctx, featureKey)

	return nil
}

// ListWhitelist 获取某功能的白名单用户列表
func (s *FeatureService) ListWhitelist(ctx context.Context, featureKey string, page, pageSize int) ([]models.UserFeatureWhitelist, int64, error) {
	var whitelists []models.UserFeatureWhitelist
	var total int64

	query := s.db.WithContext(ctx).
		Where("feature_key = ?", featureKey)

	// 获取总数
	err := query.Model(&models.UserFeatureWhitelist{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err = query.
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&whitelists).Error

	return whitelists, total, err
}

// GetUserFeatureDetails 获取用户的功能权限详情（管理员查看）
func (s *FeatureService) GetUserFeatureDetails(ctx context.Context, userID uint) ([]models.UserFeatureWhitelist, error) {
	var whitelists []models.UserFeatureWhitelist
	err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&whitelists).Error
	return whitelists, err
}

// clearUserFeaturesCache 清除用户功能缓存
func (s *FeatureService) clearUserFeaturesCache(ctx context.Context, userID uint) {
	if s.cache != nil {
		cacheKey := fmt.Sprintf(cacheKeyUserFeatures, userID)
		_ = s.cache.Delete(ctx, cacheKey)
	}
}

// clearFeatureEnabledCache 清除功能启用状态缓存
func (s *FeatureService) clearFeatureEnabledCache(ctx context.Context, featureKey string) {
	if s.cache != nil {
		cacheKey := fmt.Sprintf(cacheKeyFeatureEnabled, featureKey)
		_ = s.cache.Delete(ctx, cacheKey)
	}
}

// clearAllUsersCacheForFeature 清除所有拥有指定功能的用户缓存
func (s *FeatureService) clearAllUsersCacheForFeature(ctx context.Context, featureKey string) {
	if s.cache == nil {
		return
	}

	// 查询所有拥有该功能的用户
	var whitelists []models.UserFeatureWhitelist
	err := s.db.WithContext(ctx).
		Where("feature_key = ?", featureKey).
		Select("DISTINCT user_id").
		Find(&whitelists).Error

	if err != nil {
		return
	}

	// 清除每个用户的缓存
	for _, w := range whitelists {
		s.clearUserFeaturesCache(ctx, w.UserID)
	}
}
