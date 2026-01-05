package models

import (
	"time"

	"gorm.io/gorm"
)

// 角色标识常量
const (
	RoleTagUserBasic    = "user_basic"
	RoleTagUserActive   = "user_active"
	RoleTagUserVerified = "user_verified"
	RoleTagOperator     = "operator"
	RoleTagAdmin        = "admin"
)

// PermissionTag 常量便于复用（仅存储需要鉴权的接口，开放接口不入库）
const (
	PermissionUserGet                   = "user.get"                     // basic_user
	PermissionUserUpdate                = "user.update"                  // basic_user
	PermissionOSSTokenGet               = "oss.token.get"                // basic_user
	PermissionReviewCreate              = "review.create"                // basic_user
	PermissionReviewGetSelf             = "review.get.self"              // basic_user
	PermissionCourseTableGet            = "coursetable.get"              // basic_user
	PermissionCourseTableClassSearch    = "coursetable.class.search"     // basic_user
	PermissionCourseTableClassUpdate    = "coursetable.class.update.own" // basic_user
	PermissionCourseTableClassUpdateAll = "coursetable.class.update.all" // active_user
	PermissionCourseTableUpdate         = "coursetable.update"           // basic_user
	PermissionFailRate                  = "failrate"                     // basic_user
	PermissionPointGet                  = "point.get"                    // basic_user
	PermissionPointSpend                = "point.spend"                  // basic_user
	PermissionStatisticGet              = "statistic.get"                // basic_user

	PermissionContributionGet     = "contribution.get"      // basic_user
	PermissionContributionCreate  = "contribution.create"   // basic_user
	PermissionCountdown           = "countdown"             // basic_user
	PermissionStudyTask           = "studytask"             // basic_user
	PermissionMaterialGet         = "material.get"          // basic_user
	PermissionMaterialRate        = "material.rate"         // basic_user
	PermissionMaterialDownload    = "material.download"     // basic_user
	PermissionMaterialCategoryGet = "material.category.get" // basic_user
	PermissionQuestion            = "question"              // basic_user

	PermissionReviewManage               = "review.manage"
	PermissionCourseTableManage          = "coursetable.manage"
	PermissionHeroManage                 = "hero.manage"
	PermissionConfigManage               = "config.manage"
	PermissionPointManage                = "point.manage"
	PermissionContributionManage         = "contribution.manage"    // operator
	PermissionNotificationGet            = "notification.get"       // basic_user
	PermissionNotificationGetAdmin       = "notification.get.admin" // operator
	PermissionNotificationCreate         = "notification.create"    // operator
	PermissionNotificationPublish        = "notification.publish"   // operator
	PermissionNotificationUpdate         = "notification.update"    // operator
	PermissionNotificationApprove        = "notification.approve"   // operator
	PermissionNotificationSchedule       = "notification.schedule"  // operator
	PermissionNotificationPin            = "notification.pin"
	PermissionNotificationDelete         = "notification.delete"
	PermissionNotificationPublishAdmin   = "notification.publish.admin"
	PermissionNotificationCategoryManage = "notification.category.manage"
	PermissionFeatureManage              = "feature.manage"
	PermissionUserManage                 = "user.manage"
	PermissionMaterialManage             = "material.manage"
	PermissionS3Manage                   = "s3.manage"
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
