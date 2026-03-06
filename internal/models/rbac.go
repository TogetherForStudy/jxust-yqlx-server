package models

import (
	"time"

	"gorm.io/gorm"
)

// Role 角色模型
type Role struct {
	ID          uint           `json:"id" gorm:"type:int unsigned;primaryKey;comment:角色ID"`
	RoleTag     string         `json:"role_tag" gorm:"type:varchar(64);uniqueIndex;not null;comment:角色标识"`
	Name        string         `json:"name" gorm:"type:varchar(100);not null;comment:角色名称"`
	Description string         `json:"description" gorm:"type:varchar(255);comment:角色描述"`
	CreatedAt   time.Time      `json:"created_at" gorm:"type:datetime;comment:创建时间"`
	UpdatedAt   time.Time      `json:"updated_at" gorm:"type:datetime;comment:更新时间"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"comment:软删除时间"`
}

// Permission 权限模型
type Permission struct {
	ID            uint           `json:"id" gorm:"type:int unsigned;primaryKey;comment:权限ID"`
	PermissionTag string         `json:"permission_tag" gorm:"type:varchar(128);uniqueIndex;not null;comment:权限标识"`
	Name          string         `json:"name" gorm:"type:varchar(100);not null;comment:权限名称"`
	Description   string         `json:"description" gorm:"type:varchar(255);comment:权限描述"`
	CreatedAt     time.Time      `json:"created_at" gorm:"type:datetime;comment:创建时间"`
	UpdatedAt     time.Time      `json:"updated_at" gorm:"type:datetime;comment:更新时间"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"comment:软删除时间"`
}

// UserRole 用户与角色关联
type UserRole struct {
	ID        uint           `json:"id" gorm:"type:int unsigned;primaryKey;comment:记录ID"`
	UserID    uint           `json:"user_id" gorm:"type:int unsigned;not null;comment:用户ID"`
	RoleID    uint           `json:"role_id" gorm:"type:int unsigned;not null;comment:角色ID"`
	CreatedAt time.Time      `json:"created_at" gorm:"type:datetime;comment:创建时间"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"type:datetime;comment:更新时间"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"comment:软删除时间"`
}

// RolePermission 角色与权限关联
type RolePermission struct {
	ID           uint           `json:"id" gorm:"type:int unsigned;primaryKey;comment:记录ID"`
	RoleID       uint           `json:"role_id" gorm:"type:int unsigned;not null;comment:角色ID"`
	PermissionID uint           `json:"permission_id" gorm:"type:int unsigned;not null;comment:权限ID"`
	CreatedAt    time.Time      `json:"created_at" gorm:"type:datetime;comment:创建时间"`
	UpdatedAt    time.Time      `json:"updated_at" gorm:"type:datetime;comment:更新时间"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"comment:软删除时间"`
}
