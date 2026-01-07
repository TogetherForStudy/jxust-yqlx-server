package request

import "time"

// CreateFeatureRequest 创建功能请求
type CreateFeatureRequest struct {
	FeatureKey  string `json:"feature_key" binding:"required,max=50"`
	FeatureName string `json:"feature_name" binding:"required,max=100"`
	Description string `json:"description" binding:"max=500"`
	IsEnabled   *bool  `json:"is_enabled"`
}

// UpdateFeatureRequest 更新功能请求
type UpdateFeatureRequest struct {
	FeatureName *string `json:"feature_name" binding:"omitempty,max=100"`
	Description *string `json:"description" binding:"omitempty,max=500"`
	IsEnabled   *bool   `json:"is_enabled"`
}

// GrantFeatureRequest 授予功能权限请求
type GrantFeatureRequest struct {
	UserID    uint       `json:"user_id" binding:"required"`
	ExpiresAt *time.Time `json:"expires_at"` // 可选，NULL表示永久
}

// BatchGrantFeatureRequest 批量授予功能权限请求
type BatchGrantFeatureRequest struct {
	UserIDs   []uint     `json:"user_ids" binding:"required,min=1"`
	ExpiresAt *time.Time `json:"expires_at"`
}
