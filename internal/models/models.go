package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID        uint           `json:"id" gorm:"type:int unsigned;primaryKey;comment:用户ID"`
	OpenID    string         `json:"open_id" gorm:"type:varchar(256);uniqueIndex:idx_openid;not null;comment:微信OpenID"`
	UnionID   string         `json:"union_id" gorm:"type:varchar(256);index:idx_unionid;comment:微信UnionID"`
	Nickname  string         `json:"nickname" gorm:"type:varchar(256);comment:用户昵称"`
	Avatar    string         `json:"avatar" gorm:"type:varchar(500);comment:头像URL"`
	Phone     string         `json:"phone" gorm:"type:varchar(20);comment:手机号"`
	Password  string         `json:"-" gorm:"type:varchar(100);comment:密码哈希"`
	StudentID string         `json:"student_id" gorm:"type:varchar(20);index:idx_student_id;comment:学号"`
	RealName  string         `json:"real_name" gorm:"type:varchar(20);comment:真实姓名"`
	College   string         `json:"college" gorm:"type:varchar(50);comment:学院"`
	Major     string         `json:"major" gorm:"type:varchar(50);comment:专业"`
	ClassID   string         `json:"class_id" gorm:"type:varchar(256);comment:班级标识"`
	Role      UserRole       `json:"role" gorm:"type:tinyint;default:1;comment:用户角色：1=普通用户，2=管理员"`
	Status    UserStatus     `json:"status" gorm:"type:tinyint;default:1;comment:用户状态：1=正常，2=禁用"`
	CreatedAt time.Time      `json:"created_at" gorm:"type:datetime;comment:创建时间"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"type:datetime;comment:更新时间"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"comment:软删除时间"`
}
type UserRole int8

const (
	UserRoleNormal UserRole = 1 // 普通用户
	UserRoleAdmin  UserRole = 2 // 管理员
)

type UserStatus int8

const (
	UserStatusNormal   UserStatus = 1 // 正常
	UserStatusDisabled UserStatus = 2 // 禁用
)

// TeacherReview 教师评价模型
type TeacherReview struct {
	ID          uint                `json:"id" gorm:"type:int unsigned;primaryKey;comment:评价ID"`
	UserID      uint                `json:"user_id" gorm:"not null;index:idx_user_id;comment:评价用户ID"`
	TeacherName string              `json:"teacher_name" gorm:"type:varchar(50);not null;index:idx_teacher_name;comment:教师姓名"`
	CourseName  string              `json:"course_name" gorm:"type:varchar(100);comment:课程名称"`
	Campus      string              `json:"campus" gorm:"type:varchar(50);not null;comment:校区"`
	Content     string              `json:"content" gorm:"type:text;not null;comment:评价内容"`
	Attitude    TeacherAttitude     `json:"attitude" gorm:"type:tinyint;default:0;comment:评价态度：3=中立，1=推荐，2=避雷"`
	Status      TeacherReviewStatus `json:"status" gorm:"type:tinyint;default:1;comment:评价状态：1=待审核，2=已通过，3=已拒绝"`
	AdminNote   string              `json:"admin_note" gorm:"type:varchar(500);comment:管理员备注"`
	CreatedAt   time.Time           `json:"created_at" gorm:"type:datetime;index:idx_status_created_at;comment:创建时间"`
	UpdatedAt   time.Time           `json:"updated_at" gorm:"type:datetime;comment:更新时间"`
	DeletedAt   gorm.DeletedAt      `json:"-" gorm:"comment:软删除时间"`
}

// TeacherAttitude 评价态度（自定义类型）
type TeacherAttitude int8

// 评价态度常量定义（提升可读性）
const (
	AttitudeNeutral   TeacherAttitude = 3 // 中立
	AttitudeRecommend TeacherAttitude = 1 // 推荐
	AttitudeAvoid     TeacherAttitude = 2 // 避雷
)

type TeacherReviewStatus int8

const (
	TeacherReviewStatusPending  TeacherReviewStatus = 1 // 待审核
	TeacherReviewStatusApproved TeacherReviewStatus = 2 // 已通过
	TeacherReviewStatusRejected TeacherReviewStatus = 3 // 已拒绝
)

// 课程表模型
type CourseTable struct {
	ID         uint           `json:"id" gorm:"type:int unsigned;primaryKey;comment:课程表ID"`
	ClassID    string         `json:"class_id" gorm:"type:varchar(50);not null;comment:班级ID;index:idx_class_id"`
	CourseData datatypes.JSON `json:"course_data" gorm:"type:json;not null;comment:课程数据"`
	CreatedAt  time.Time      `json:"created_at" gorm:"type:datetime;comment:创建时间"`
	UpdatedAt  time.Time      `json:"updated_at" gorm:"type:datetime;comment:更新时间"`
	Semester   string         `json:"semester" gorm:"type:varchar(50);not null;comment:学期;index:idx_semester"`
}

// 用户个性化课程表模型
type ScheduleUser struct {
	ID        uint           `json:"id" gorm:"type:int unsigned;primaryKey;comment:记录ID"`
	UserID    uint           `json:"user_id" gorm:"not null;comment:用户ID"`
	ClassID   string         `json:"class_id" gorm:"type:varchar(50);not null;comment:班级ID"`
	Semester  string         `json:"semester" gorm:"type:varchar(50);not null;comment:学期"`
	Schedule  datatypes.JSON `json:"schedule" gorm:"type:json;not null;comment:个性化完整课程数据"`
	CreatedAt time.Time      `json:"created_at" gorm:"type:datetime;comment:创建时间"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"type:datetime;comment:更新时间"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"comment:软删除时间"`
}

// FailRate 挂科率模型
type FailRate struct {
	ID         uint      `json:"id" gorm:"type:int unsigned;primaryKey;comment:记录ID"`
	CourseName string    `json:"course_name" gorm:"type:varchar(150);not null;index:idx_failrate_course_name;comment:课程名称"`
	Department string    `json:"department" gorm:"type:varchar(150);not null;index:idx_failrate_unit;comment:开课单位"`
	Semester   string    `json:"semester" gorm:"type:varchar(20);not null;index:idx_failrate_semester;comment:学期(如2024-2025-1)"`
	FailRate   float64   `json:"failrate" gorm:"type:decimal(4,1);not null;default:0.0;comment:挂科率百分比(0-100.0)"`
	CreatedAt  time.Time `json:"created_at" gorm:"type:datetime;comment:创建时间"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"type:datetime;comment:更新时间"`
}

// Hero 英雄模型
type Hero struct {
	ID        uint      `json:"id" gorm:"type:int unsigned;primaryKey;comment:英雄ID"`
	Name      string    `json:"name" gorm:"type:varchar(100);uniqueIndex;not null;comment:英雄名称"`
	Sort      int       `json:"sort" gorm:"type:int;default:0;not null;comment:排序值"`
	IsShow    bool      `json:"is_show" gorm:"type:tinyint(1);not null;default:1;comment:是否展示"`
	CreatedAt time.Time `json:"created_at" gorm:"type:datetime;comment:创建时间"`
	UpdatedAt time.Time `json:"updated_at" gorm:"type:datetime;comment:更新时间"`
}

// SystemConfig 配置项模型（键值对）
type SystemConfig struct {
	ID          uint           `json:"id" gorm:"type:int unsigned;primaryKey;comment:配置ID"`
	Key         string         `json:"key" gorm:"type:varchar(191);uniqueIndex:idx_config_key;not null;comment:配置键(唯一)"`
	Value       string         `json:"value" gorm:"type:text;not null;comment:配置值(字符串/JSON/数字/布尔以字符串形式存储)"`
	ValueType   string         `json:"value_type" gorm:"type:varchar(20);not null;default:'string';comment:值类型: string|number|boolean|json"`
	Description string         `json:"description" gorm:"type:varchar(500);comment:描述"`
	CreatedAt   time.Time      `json:"created_at" gorm:"type:datetime;comment:创建时间"`
	UpdatedAt   time.Time      `json:"updated_at" gorm:"type:datetime;comment:更新时间"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"comment:软删除时间"`
}
