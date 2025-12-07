package models

import (
	"time"

	"gorm.io/gorm"
)

// Feature 功能定义模型
type Feature struct {
	ID          uint           `json:"id" gorm:"type:int unsigned;primaryKey;comment:功能ID"`
	FeatureKey  string         `json:"feature_key" gorm:"type:varchar(50);uniqueIndex:idx_feature_key;not null;comment:功能唯一标识"`
	FeatureName string         `json:"feature_name" gorm:"type:varchar(100);not null;comment:功能显示名称"`
	Description string         `json:"description" gorm:"type:varchar(500);comment:功能描述"`
	IsEnabled   bool           `json:"is_enabled" gorm:"type:tinyint;default:1;index:idx_is_enabled;comment:全局开关：1=启用 0=禁用"`
	CreatedAt   time.Time      `json:"created_at" gorm:"type:datetime;comment:创建时间"`
	UpdatedAt   time.Time      `json:"updated_at" gorm:"type:datetime;comment:更新时间"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index;comment:软删除时间"`
}

// TableName 指定表名
func (Feature) TableName() string {
	return "features"
}

// UserFeatureWhitelist 用户功能白名单模型
type UserFeatureWhitelist struct {
	ID         uint       `json:"id" gorm:"type:int unsigned;primaryKey;comment:白名单ID"`
	UserID     uint       `json:"user_id" gorm:"type:int unsigned;not null;index:idx_user_id;comment:用户ID"`
	FeatureKey string     `json:"feature_key" gorm:"type:varchar(50);not null;index:idx_feature_key;comment:功能标识"`
	GrantedBy  uint       `json:"granted_by" gorm:"type:int unsigned;comment:授权人ID（管理员）"`
	GrantedAt  time.Time  `json:"granted_at" gorm:"type:datetime;comment:授权时间"`
	ExpiresAt  *time.Time `json:"expires_at" gorm:"type:datetime;index:idx_expires_at;comment:过期时间，NULL表示永久有效"`
	CreatedAt  time.Time  `json:"created_at" gorm:"type:datetime;comment:创建时间"`
	UpdatedAt  time.Time  `json:"updated_at" gorm:"type:datetime;comment:更新时间"`
}

// TableName 指定表名
func (UserFeatureWhitelist) TableName() string {
	return "user_feature_whitelist"
}

// IsExpired 检查白名单是否已过期
func (w *UserFeatureWhitelist) IsExpired() bool {
	if w.ExpiresAt == nil {
		return false // 永久有效
	}
	return time.Now().After(*w.ExpiresAt)
}
