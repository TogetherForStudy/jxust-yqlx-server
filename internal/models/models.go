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
	Points    uint           `json:"points" gorm:"type:int unsigned;default:0;comment:积分"`
	CreatedAt time.Time      `json:"created_at" gorm:"type:datetime;comment:创建时间"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"type:datetime;comment:更新时间"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"comment:软删除时间"`
}
type UserRole int8

const (
	UserRoleNormal   UserRole = 1 // 普通用户
	UserRoleAdmin    UserRole = 2 // 管理员
	UserRoleOperator UserRole = 3 // 运营人员
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

// BindRecord 绑定记录表：记录用户访问绑定接口次数与成功绑定次数
type BindRecord struct {
	ID        uint           `json:"id" gorm:"type:int unsigned;primaryKey;comment:记录ID"`
	UserID    uint           `json:"user_id" gorm:"not null;uniqueIndex:idx_bind_user_id;comment:用户ID"`
	BindCount int            `json:"bind_count" gorm:"type:int;not null;default:0;comment:成功绑定次数"`
	CreatedAt time.Time      `json:"created_at" gorm:"type:datetime;comment:创建时间"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"type:datetime;comment:更新时间"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"comment:软删除时间"`
}

// ==================== 通知/日程系统模型 ====================

// NotificationCategory 通知分类模型
type NotificationCategory struct {
	ID        uint      `json:"id" gorm:"type:int unsigned;primaryKey;comment:分类ID"`
	Name      string    `json:"name" gorm:"type:varchar(20);not null;uniqueIndex;comment:分类名称"`
	Sort      int       `json:"sort" gorm:"type:int;default:0;comment:排序值"`
	IsActive  bool      `json:"is_active" gorm:"type:tinyint(1);default:1;comment:是否启用"`
	CreatedAt time.Time `json:"created_at" gorm:"type:datetime;comment:创建时间"`
	UpdatedAt time.Time `json:"updated_at" gorm:"type:datetime;comment:更新时间"`
}

// Notification 通知模型
type Notification struct {
	ID            uint                      `json:"id" gorm:"type:int unsigned;primaryKey;comment:通知ID"`
	Title         string                    `json:"title" gorm:"type:varchar(200);not null;comment:通知标题"`
	Content       string                    `json:"content" gorm:"type:text;comment:详细内容"`
	PublisherID   uint                      `json:"publisher_id" gorm:"not null;index:idx_publisher;comment:发布者ID(审核者ID)"`
	PublisherType NotificationPublisherType `json:"publisher_type" gorm:"type:tinyint;default:1;comment:发布者类型：1=运营，2=用户投稿"`
	ContributorID *uint                     `json:"contributor_id" gorm:"index:idx_contributor;comment:投稿者ID(用户投稿时)"`
	Categories    datatypes.JSON            `json:"categories" gorm:"type:json;not null;comment:分类ID数组"`
	Status        NotificationStatus        `json:"status" gorm:"type:tinyint;default:1;index:idx_status_published;comment:状态：1=草稿，2=待审核，3=已发布，4=已删除"`
	Schedule      datatypes.JSON            `json:"schedule" gorm:"type:json;comment:日程信息JSON"`
	ViewCount     uint                      `json:"view_count" gorm:"type:int unsigned;default:0;comment:查看次数"`
	IsPinned      bool                      `json:"is_pinned" gorm:"type:tinyint(1);default:0;index:idx_pinned;comment:是否置顶"`
	PinnedAt      *time.Time                `json:"pinned_at" gorm:"type:datetime;index:idx_pinned_at;comment:置顶时间"`
	PublishedAt   *time.Time                `json:"published_at" gorm:"type:datetime;index:idx_status_published;comment:发布时间"`
	CreatedAt     time.Time                 `json:"created_at" gorm:"type:datetime;comment:创建时间"`
	UpdatedAt     time.Time                 `json:"updated_at" gorm:"type:datetime;comment:更新时间"`
	DeletedAt     gorm.DeletedAt            `json:"-" gorm:"comment:软删除时间"`
}

// NotificationPublisherType 通知发布者类型
type NotificationPublisherType int8

const (
	NotificationPublisherOperator NotificationPublisherType = 1 // 运营发布
	NotificationPublisherUser     NotificationPublisherType = 2 // 用户投稿
)

// NotificationStatus 通知状态
type NotificationStatus int8

const (
	NotificationStatusDraft     NotificationStatus = 1 // 草稿
	NotificationStatusPending   NotificationStatus = 2 // 待审核
	NotificationStatusPublished NotificationStatus = 3 // 已发布
	NotificationStatusDeleted   NotificationStatus = 4 // 已删除
)

// NotificationApproval 通知审核记录模型
type NotificationApproval struct {
	ID             uint                       `json:"id" gorm:"type:int unsigned;primaryKey;comment:审核记录ID"`
	NotificationID uint                       `json:"notification_id" gorm:"not null;index:idx_notification;comment:通知ID"`
	ReviewerID     uint                       `json:"reviewer_id" gorm:"not null;index:idx_reviewer;comment:审核者ID"`
	Status         NotificationApprovalStatus `json:"status" gorm:"type:tinyint;not null;comment:审核状态：1=同意，2=拒绝"`
	Note           string                     `json:"note" gorm:"type:varchar(500);comment:审核备注"`
	CreatedAt      time.Time                  `json:"created_at" gorm:"type:datetime;comment:创建时间"`
	UpdatedAt      time.Time                  `json:"updated_at" gorm:"type:datetime;comment:更新时间"`
}

// NotificationApprovalStatus 通知审核状态
type NotificationApprovalStatus int8

const (
	NotificationApprovalStatusApproved NotificationApprovalStatus = 1 // 同意
	NotificationApprovalStatusRejected NotificationApprovalStatus = 2 // 拒绝
)

// ScheduleData 日程数据结构（用于JSON存储）
type ScheduleData struct {
	Title       string             `json:"title"`       // 总日程名称
	Description string             `json:"description"` // 日程描述
	TimeSlots   []ScheduleTimeSlot `json:"time_slots"`  // 时间段列表
}

// ScheduleTimeSlot 日程时间段
type ScheduleTimeSlot struct {
	Name      string `json:"name"`       // 时间段名称
	StartDate string `json:"start_date"` // 开始日期 YYYY-MM-DD
	EndDate   string `json:"end_date"`   // 结束日期 YYYY-MM-DD
	StartTime string `json:"start_time"` // 开始时间 HH:MM (可选)
	EndTime   string `json:"end_time"`   // 结束时间 HH:MM (可选)
	IsAllDay  bool   `json:"is_all_day"` // 是否全天
}

// PointsTransaction 积分变动记录模型
type PointsTransaction struct {
	ID          uint                    `json:"id" gorm:"type:int unsigned;primaryKey;comment:交易ID"`
	UserID      uint                    `json:"user_id" gorm:"not null;index:idx_user_created;comment:用户ID"`
	Type        PointsTransactionType   `json:"type" gorm:"type:tinyint;not null;index:idx_type_source;comment:类型：1=获得，2=消耗"`
	Source      PointsTransactionSource `json:"source" gorm:"type:tinyint;not null;index:idx_type_source;comment:来源：1=投稿采纳，2=兑换奖品"`
	Points      int                     `json:"points" gorm:"type:int;not null;comment:积分数量"`
	Description string                  `json:"description" gorm:"type:varchar(200);comment:描述"`
	RelatedID   *uint                   `json:"related_id" gorm:"comment:关联ID(投稿ID/奖品ID等)"`
	CreatedAt   time.Time               `json:"created_at" gorm:"type:datetime;index:idx_user_created;comment:创建时间"`
}

// PointsTransactionType 积分交易类型
type PointsTransactionType int8

const (
	PointsTransactionTypeEarn  PointsTransactionType = 1 // 获得
	PointsTransactionTypeSpend PointsTransactionType = 2 // 消耗
)

// PointsTransactionSource 积分交易来源
type PointsTransactionSource int8

const (
	PointsTransactionSourceContribution PointsTransactionSource = 1 // 投稿采纳
	PointsTransactionSourceRedeem       PointsTransactionSource = 2 // 兑换奖品
)

// UserContribution 用户投稿模型
type UserContribution struct {
	ID             uint                   `json:"id" gorm:"type:int unsigned;primaryKey;comment:投稿ID"`
	UserID         uint                   `json:"user_id" gorm:"not null;index:idx_user_status;comment:投稿用户ID"`
	Title          string                 `json:"title" gorm:"type:varchar(200);not null;comment:投稿标题"`
	Content        string                 `json:"content" gorm:"type:text;comment:投稿内容"`
	Categories     datatypes.JSON         `json:"categories" gorm:"type:json;not null;comment:建议分类"`
	Status         UserContributionStatus `json:"status" gorm:"type:tinyint;default:1;index:idx_user_status,idx_status_created;comment:状态：1=待审核，2=已采纳，3=已拒绝"`
	ReviewerID     *uint                  `json:"reviewer_id" gorm:"comment:审核者ID"`
	ReviewNote     string                 `json:"review_note" gorm:"type:varchar(500);comment:审核备注"`
	NotificationID *uint                  `json:"notification_id" gorm:"index:idx_notification;comment:采纳后的通知ID"`
	PointsAwarded  uint                   `json:"points_awarded" gorm:"type:int unsigned;default:0;comment:奖励积分"`
	ReviewedAt     *time.Time             `json:"reviewed_at" gorm:"type:datetime;comment:审核时间"`
	CreatedAt      time.Time              `json:"created_at" gorm:"type:datetime;index:idx_status_created;comment:创建时间"`
	UpdatedAt      time.Time              `json:"updated_at" gorm:"type:datetime;comment:更新时间"`
}

// UserContributionStatus 用户投稿状态
type UserContributionStatus int8

const (
	UserContributionStatusPending  UserContributionStatus = 1 // 待审核
	UserContributionStatusApproved UserContributionStatus = 2 // 已采纳
	UserContributionStatusRejected UserContributionStatus = 3 // 已拒绝
)

// ==================== 倒数日功能模型 ====================

// Countdown 倒数日模型
type Countdown struct {
	ID          uint           `json:"id" gorm:"type:int unsigned;primaryKey;comment:倒数日ID"`
	UserID      uint           `json:"user_id" gorm:"not null;index:idx_user_id;comment:用户ID"`
	Title       string         `json:"title" gorm:"type:varchar(100);not null;comment:倒数日标题"`
	Description string         `json:"description" gorm:"type:text;comment:描述"`
	TargetDate  time.Time      `json:"target_date" gorm:"type:date;not null;index:idx_target_date;comment:目标日期"`
	CreatedAt   time.Time      `json:"created_at" gorm:"type:datetime;comment:创建时间"`
	UpdatedAt   time.Time      `json:"updated_at" gorm:"type:datetime;comment:更新时间"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"comment:软删除时间"`
}

// ==================== 学习清单功能模型 ====================

// StudyTask 学习任务模型
type StudyTask struct {
	ID          uint              `json:"id" gorm:"type:int unsigned;primaryKey;comment:任务ID"`
	UserID      uint              `json:"user_id" gorm:"not null;index:idx_user_status;comment:用户ID"`
	Title       string            `json:"title" gorm:"type:varchar(200);not null;comment:任务标题"`
	Description string            `json:"description" gorm:"type:text;comment:任务描述"`
	DueDate     *time.Time        `json:"due_date" gorm:"type:datetime;comment:截止日期"`
	Priority    StudyTaskPriority `json:"priority" gorm:"type:tinyint;default:2;comment:优先级：1=高，2=中，3=低"`
	Status      StudyTaskStatus   `json:"status" gorm:"type:tinyint;default:1;index:idx_user_status;comment:状态：1=待完成，2=已完成"`
	CompletedAt *time.Time        `json:"completed_at" gorm:"type:datetime;comment:完成时间"`
	CreatedAt   time.Time         `json:"created_at" gorm:"type:datetime;comment:创建时间"`
	UpdatedAt   time.Time         `json:"updated_at" gorm:"type:datetime;comment:更新时间"`
	DeletedAt   gorm.DeletedAt    `json:"-" gorm:"comment:软删除时间"`
}

// StudyTaskPriority 学习任务优先级
type StudyTaskPriority int8

const (
	StudyTaskPriorityHigh   StudyTaskPriority = 1 // 高
	StudyTaskPriorityMedium StudyTaskPriority = 2 // 中
	StudyTaskPriorityLow    StudyTaskPriority = 3 // 低
)

// StudyTaskStatus 学习任务状态
type StudyTaskStatus int8

const (
	StudyTaskStatusPending   StudyTaskStatus = 1 // 待完成
	StudyTaskStatusCompleted StudyTaskStatus = 2 // 已完成
)
