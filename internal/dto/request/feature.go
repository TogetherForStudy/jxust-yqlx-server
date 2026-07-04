package request

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

// GrantFeatureRequest 授予功能权限请求（支持单个/批量）
type GrantFeatureRequest struct {
	UserID  uint   `json:"user_id"`  // 单个用户ID
	UserIDs []uint `json:"user_ids"` // 批量用户ID
}
