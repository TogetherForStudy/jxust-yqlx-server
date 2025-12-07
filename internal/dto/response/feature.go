package response

import "time"

// UserFeaturesResponse 用户功能列表响应
type UserFeaturesResponse struct {
	Features []string `json:"features"` // ["beta_ai_chat", "beta_study_plan"]
}

// FeatureResponse 功能详情响应
type FeatureResponse struct {
	ID          uint      `json:"id"`
	FeatureKey  string    `json:"feature_key"`
	FeatureName string    `json:"feature_name"`
	Description string    `json:"description"`
	IsEnabled   bool      `json:"is_enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// WhitelistUserInfo 白名单用户信息响应
type WhitelistUserInfo struct {
	ID        uint       `json:"id"`
	UserID    uint       `json:"user_id"`
	StudentID string     `json:"student_id"`
	RealName  string     `json:"real_name"`
	GrantedBy uint       `json:"granted_by"`
	GrantedAt time.Time  `json:"granted_at"`
	ExpiresAt *time.Time `json:"expires_at"`
	IsExpired bool       `json:"is_expired"`
	CreatedAt time.Time  `json:"created_at"`
}

// UserFeatureInfo 用户功能权限详情（管理员查看）
type UserFeatureInfo struct {
	FeatureKey  string     `json:"feature_key"`
	FeatureName string     `json:"feature_name"`
	GrantedBy   uint       `json:"granted_by"`
	GrantedAt   time.Time  `json:"granted_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	IsExpired   bool       `json:"is_expired"`
}
