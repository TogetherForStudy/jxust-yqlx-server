package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Feature 功能定义模型（灰度规则唯一入口）
// 授权逻辑：is_enabled=true 且 (user_id IN user_ids OR user_role IN role_ids)
type Feature struct {
	ID          uint           `json:"id" gorm:"type:int unsigned;primaryKey;comment:功能ID"`
	FeatureKey  string         `json:"feature_key" gorm:"type:varchar(50);uniqueIndex:idx_feature_key;not null;comment:功能唯一标识"`
	FeatureName string         `json:"feature_name" gorm:"type:varchar(100);not null;comment:功能显示名称"`
	Description string         `json:"description" gorm:"type:varchar(500);comment:功能描述"`
	IsEnabled   bool           `json:"is_enabled" gorm:"type:tinyint;default:1;index:idx_is_enabled;comment:全局开关：1=启用 0=禁用"`
	UserIDs     datatypes.JSON `json:"user_ids" gorm:"type:json;comment:授权用户ID列表，如 [1,2,3]"`
	RoleIDs     datatypes.JSON `json:"role_ids" gorm:"type:json;comment:授权角色ID列表，如 [1,2,3]"`
	CreatedAt   time.Time      `json:"created_at" gorm:"type:datetime;comment:创建时间"`
	UpdatedAt   time.Time      `json:"updated_at" gorm:"type:datetime;comment:更新时间"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index;comment:软删除时间"`
}

// TableName 指定表名
func (Feature) TableName() string {
	return "features"
}
