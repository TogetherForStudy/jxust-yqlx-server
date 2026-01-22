package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID            uint           `json:"id" gorm:"type:int unsigned;primaryKey;comment:用户ID"`
	OpenID        string         `json:"open_id" gorm:"type:varchar(256);uniqueIndex:idx_openid;not null;comment:微信OpenID"`
	UnionID       string         `json:"union_id" gorm:"type:varchar(256);index:idx_unionid;comment:微信UnionID"`
	Nickname      string         `json:"nickname" gorm:"type:varchar(256);comment:用户昵称"`
	Avatar        string         `json:"avatar" gorm:"type:varchar(500);comment:头像URL"`
	Phone         string         `json:"phone" gorm:"type:varchar(20);comment:手机号"`
	Password      string         `json:"-" gorm:"type:varchar(100);comment:密码哈希"`
	StudentID     string         `json:"student_id" gorm:"type:varchar(20);index:idx_student_id;comment:学号"`
	RealName      string         `json:"real_name" gorm:"type:varchar(20);comment:真实姓名"`
	College       string         `json:"college" gorm:"type:varchar(50);comment:学院"`
	Major         string         `json:"major" gorm:"type:varchar(50);comment:专业"`
	ClassID       string         `json:"class_id" gorm:"type:varchar(256);comment:班级标识"`
	Role          int8           `json:"role" gorm:"type:tinyint;default:1;comment:用户角色：1=普通用户，2=管理员，3=运营（向前兼容字段）"`
	Status        UserStatus     `json:"status" gorm:"type:tinyint;default:1;comment:用户状态：1=正常，2=禁用"`
	Points        uint           `json:"points" gorm:"type:int unsigned;default:0;comment:积分"`
	PomodoroCount uint           `json:"pomodoro_count" gorm:"type:int unsigned;default:0;comment:番茄钟次数"`
	CreatedAt     time.Time      `json:"created_at" gorm:"type:datetime;comment:创建时间"`
	UpdatedAt     time.Time      `json:"updated_at" gorm:"type:datetime;comment:更新时间"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"comment:软删除时间"`
}

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

	// 关联关系
	Publisher   *User `json:"publisher" gorm:"foreignKey:PublisherID;references:ID;constraint:-"`
	Contributor *User `json:"contributor" gorm:"foreignKey:ContributorID;references:ID;constraint:-"`
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
	ID          uint                  `json:"id" gorm:"type:int unsigned;primaryKey;comment:交易ID"`
	UserID      uint                  `json:"user_id" gorm:"not null;index:idx_user_created;comment:用户ID"`
	Type        PointsTransactionType `json:"type" gorm:"type:tinyint;not null;index:idx_type_source;comment:类型：1=获得，2=消耗"`
	Source      string                `json:"source" gorm:"type:varchar(50);not null;index:idx_type_source;comment:来源"`
	Points      int                   `json:"points" gorm:"type:int;not null;comment:积分数量"`
	Description string                `json:"description" gorm:"type:varchar(200);comment:描述"`
	RelatedID   *uint                 `json:"related_id" gorm:"comment:关联ID(投稿ID/奖品ID等)"`
	CreatedAt   time.Time             `json:"created_at" gorm:"type:datetime;index:idx_user_created;comment:创建时间"`
}

// PointsTransactionType 积分交易类型
type PointsTransactionType int8

const (
	PointsTransactionTypeEarn  PointsTransactionType = 1 // 获得
	PointsTransactionTypeSpend PointsTransactionType = 2 // 消耗
)

// PointsTransactionSource 积分交易来源常量
const (
	PointsTransactionSourceDailyLogin   = "daily_login"  // 每日登录
	PointsTransactionSourceReview       = "review"       // 发布评价并审核通过
	PointsTransactionSourceContribution = "contribution" // 投稿信息并审核通过
	PointsTransactionSourceRedeem       = "redeem"       // 兑换奖品
	PointsTransactionSourceAdminGrant   = "admin_grant"  // 管理员手动赋予
)

// UserActivity 用户活动记录模型
type UserActivity struct {
	ID         uint      `json:"id" gorm:"type:int unsigned;primaryKey;comment:记录ID"`
	UserID     uint      `json:"user_id" gorm:"not null;uniqueIndex:idx_user_date;comment:用户ID"`
	Date       time.Time `json:"date" gorm:"type:date;not null;uniqueIndex:idx_user_date;comment:活动日期"`
	VisitCount int       `json:"visit_count" gorm:"type:int;default:1;comment:访问次数"`
	CreatedAt  time.Time `json:"created_at" gorm:"type:datetime;comment:创建时间"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"type:datetime;comment:更新时间"`
}

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

	// 关联关系
	User         *User         `json:"user" gorm:"foreignKey:UserID;references:ID;constraint:-"`
	Reviewer     *User         `json:"reviewer" gorm:"foreignKey:ReviewerID;references:ID;constraint:-"`
	Notification *Notification `json:"notification" gorm:"foreignKey:NotificationID;references:ID;constraint:-"`
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

// ==================== 资料管理系统模型 ====================

// Material 资料表
type Material struct {
	ID         uint           `json:"id" gorm:"type:int unsigned;primaryKey;comment:资料ID"`
	MD5        string         `json:"md5" gorm:"type:varchar(32);not null;comment:文件MD5"`
	FileName   string         `json:"file_name" gorm:"type:varchar(255);not null;comment:文件名"`
	FileSize   int64          `json:"file_size" gorm:"type:bigint;not null;comment:文件大小(字节)"`
	CategoryID uint           `json:"category_id" gorm:"not null;index:idx_material_category;comment:分类ID"`
	CreatedAt  time.Time      `json:"created_at" gorm:"type:datetime;comment:创建时间"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"comment:软删除时间"`
	// 关联（仅定义关联关系，不创建数据库外键约束）
	Category *MaterialCategory `json:"category,omitempty" gorm:"foreignKey:CategoryID;references:ID;constraint:-"`
	Desc     *MaterialDesc     `json:"desc,omitempty" gorm:"foreignKey:MD5;references:MD5;constraint:-"`
}

// MaterialDesc 资料描述表
type MaterialDesc struct {
	MD5           string         `json:"md5" gorm:"type:varchar(32);primaryKey;comment:文件MD5"`
	Tags          string         `json:"tags" gorm:"type:varchar(500);comment:标签,用逗号分隔"`
	Description   string         `json:"description" gorm:"type:text;comment:描述"`
	ExternalLink  string         `json:"external_link" gorm:"type:varchar(1000);comment:外部下载链接"`
	TotalHotness  int            `json:"total_hotness" gorm:"type:int;default:0;comment:总热度"`
	PeriodHotness int            `json:"period_hotness" gorm:"type:int;default:0;comment:期间热度"`
	IsRecommended bool           `json:"is_recommended" gorm:"type:tinyint(1);default:0;comment:人工推荐"`
	ViewCount     int            `json:"view_count" gorm:"type:int;default:0;comment:查看次数"`
	DownloadCount int            `json:"download_count" gorm:"type:int;default:0;comment:下载次数"`
	Rating        float64        `json:"rating" gorm:"type:decimal(3,2);default:0.00;comment:平均评分(0-5)"`
	RatingCount   int            `json:"rating_count" gorm:"type:int;default:0;comment:评分人数"`
	UpdatedAt     time.Time      `json:"updated_at" gorm:"type:datetime;comment:更新时间"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"comment:软删除时间"`
}

// MaterialCategory 分类表
type MaterialCategory struct {
	ID        uint      `json:"id" gorm:"type:int unsigned;primaryKey;comment:分类ID"`
	Name      string    `json:"name" gorm:"type:varchar(500);not null;comment:分类名称"`
	ParentID  uint      `json:"parent_id" gorm:"type:int unsigned;default:0;index:idx_category_parent;comment:上级分类ID,0表示根级别"`
	Level     int       `json:"level" gorm:"type:tinyint;default:1;comment:层级级别"`
	Sort      int       `json:"sort" gorm:"type:int;default:0;comment:排序"`
	CreatedAt time.Time `json:"created_at" gorm:"type:datetime;comment:创建时间"`
}

// MaterialLog 记录表
type MaterialLog struct {
	ID          uint            `json:"id" gorm:"type:int unsigned;primaryKey;comment:日志ID"`
	UserID      uint            `json:"user_id" gorm:"not null;index:idx_log_user;comment:用户ID"`
	Type        MaterialLogType `json:"type" gorm:"type:tinyint;not null;index:idx_log_type;comment:记录类型：1=搜索，2=查看，3=评分，4=下载"`
	Keywords    string          `json:"keywords" gorm:"type:varchar(200);comment:搜索关键词"`
	MaterialMD5 string          `json:"material_md5" gorm:"type:varchar(32);index:idx_log_material;comment:资料MD5"`
	Rating      *int            `json:"rating" gorm:"type:tinyint;comment:评分(1-5)"`
	Count       int             `json:"count" gorm:"type:int;default:0;comment:次数"`
	CreatedAt   time.Time       `json:"created_at" gorm:"type:datetime;comment:创建时间"`
	DeletedAt   gorm.DeletedAt  `json:"-" gorm:"comment:软删除时间"`
}

// MaterialLogType 记录类型
type MaterialLogType int8

const (
	MaterialLogTypeSearch   MaterialLogType = 1 // 搜索
	MaterialLogTypeView     MaterialLogType = 2 // 查看
	MaterialLogTypeRating   MaterialLogType = 3 // 评分
	MaterialLogTypeDownload MaterialLogType = 4 // 下载
)

// =============== 刷题功能相关模型 ===============

// QuestionProject 题目项目（题库分类）
type QuestionProject struct {
	ID          uint           `json:"id" gorm:"type:int unsigned;primaryKey;comment:项目ID"`
	Name        string         `json:"name" gorm:"type:varchar(100);not null;comment:项目名称"`
	Description string         `json:"description" gorm:"type:text;comment:项目描述"`
	Version     int            `json:"version" gorm:"type:int;default:1;comment:版本号"`
	Sort        int            `json:"sort" gorm:"type:int;default:0;comment:排序"`
	IsActive    bool           `json:"is_active" gorm:"type:tinyint(1);default:1;comment:是否启用"`
	CreatedAt   time.Time      `json:"created_at" gorm:"type:datetime;comment:创建时间"`
	UpdatedAt   time.Time      `json:"updated_at" gorm:"type:datetime;comment:更新时间"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"comment:软删除时间"`
}

// Question 题目
type Question struct {
	ID        uint           `json:"id" gorm:"type:int unsigned;primaryKey;comment:题目ID"`
	ProjectID uint           `json:"project_id" gorm:"not null;index:idx_project_question;comment:项目ID"`
	ParentID  *uint          `json:"parent_id" gorm:"type:int unsigned;index:idx_parent_question;comment:父题目ID（用于题目分组，null表示主题或独立题）"`
	Type      QuestionType   `json:"type" gorm:"type:tinyint;not null;comment:题目类型：1=选择题，2=简答题"`
	Title     string         `json:"title" gorm:"type:text;not null;comment:题目标题"`
	Options   datatypes.JSON `json:"options" gorm:"type:json;comment:选项（JSON数组，仅选择题使用）"`
	Answer    string         `json:"answer" gorm:"type:text;not null;comment:答案"`
	Sort      int            `json:"sort" gorm:"type:int;default:0;comment:排序"`
	IsActive  bool           `json:"is_active" gorm:"type:tinyint(1);default:1;comment:是否启用"`
	CreatedAt time.Time      `json:"created_at" gorm:"type:datetime;comment:创建时间"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"type:datetime;comment:更新时间"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"type:datetime;index;comment:删除时间"`
	// 关联
	Project      *QuestionProject `json:"project,omitempty" gorm:"foreignKey:ProjectID;references:ID;constraint:-"`
	Parent       *Question        `json:"parent,omitempty" gorm:"foreignKey:ParentID;references:ID;constraint:-"`
	SubQuestions []Question       `json:"sub_questions,omitempty" gorm:"foreignKey:ParentID;references:ID;constraint:-"`
}

// QuestionType 题目类型
type QuestionType int8

const (
	QuestionTypeChoice QuestionType = 1 // 选择题
	QuestionTypeEssay  QuestionType = 2 // 简答题
)

// UserProjectUsage 用户对项目的使用记录
type UserProjectUsage struct {
	ID         uint      `json:"id" gorm:"type:int unsigned;primaryKey;comment:记录ID"`
	UserID     uint      `json:"user_id" gorm:"not null;uniqueIndex:idx_user_project_usage;comment:用户ID"`
	ProjectID  uint      `json:"project_id" gorm:"not null;uniqueIndex:idx_user_project_usage;comment:项目ID"`
	UsageCount int       `json:"usage_count" gorm:"type:int;default:0;comment:使用次数"`
	LastUsedAt time.Time `json:"last_used_at" gorm:"type:datetime;comment:最后使用时间"`
	CreatedAt  time.Time `json:"created_at" gorm:"type:datetime;comment:创建时间"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"type:datetime;comment:更新时间"`
	// 关联
	User    *User            `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID;constraint:-"`
	Project *QuestionProject `json:"project,omitempty" gorm:"foreignKey:ProjectID;references:ID;constraint:-"`
}

// UserQuestionUsage 用户对题目的使用记录
type UserQuestionUsage struct {
	ID              uint       `json:"id" gorm:"type:int unsigned;primaryKey;comment:记录ID"`
	UserID          uint       `json:"user_id" gorm:"not null;uniqueIndex:idx_user_question_usage;comment:用户ID"`
	QuestionID      uint       `json:"question_id" gorm:"not null;uniqueIndex:idx_user_question_usage;comment:题目ID"`
	StudyCount      int        `json:"study_count" gorm:"type:int;default:0;comment:学习次数"`
	PracticeCount   int        `json:"practice_count" gorm:"type:int;default:0;comment:做题次数"`
	LastStudiedAt   *time.Time `json:"last_studied_at" gorm:"type:datetime;comment:最后学习时间"`
	LastPracticedAt *time.Time `json:"last_practiced_at" gorm:"type:datetime;comment:最后做题时间"`
	CreatedAt       time.Time  `json:"created_at" gorm:"type:datetime;comment:创建时间"`
	UpdatedAt       time.Time  `json:"updated_at" gorm:"type:datetime;comment:更新时间"`
	// 关联
	User     *User     `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID;constraint:-"`
	Question *Question `json:"question,omitempty" gorm:"foreignKey:QuestionID;references:ID;constraint:-"`
}

// =============== 词典功能相关模型 ===============

// Dictionary 词典模型
type Dictionary struct {
	ID         uint           `json:"id" gorm:"type:int;primaryKey;autoIncrement;comment:记录ID"`
	Word       string         `json:"word" gorm:"type:varchar(100);comment:单词"`
	PhoneticUK string         `json:"phonetic_uk" gorm:"type:text;comment:英标"`
	PhoneticUS string         `json:"phonetic_us" gorm:"type:text;comment:美标"`
	Trans      datatypes.JSON `json:"trans" gorm:"type:json;comment:翻译"`
	Sentences  datatypes.JSON `json:"sentences" gorm:"type:json;comment:例句"`
	Phrases    datatypes.JSON `json:"phrases" gorm:"type:json;comment:短语"`
	Synos      datatypes.JSON `json:"synos" gorm:"type:json;comment:同义词"`
	RelWords   datatypes.JSON `json:"rel_words" gorm:"type:json;comment:派生词"`
	Source     string         `json:"source" gorm:"type:varchar(20);comment:来源"`
}

func (Dictionary) TableName() string {
	return "dictionary"
}

// ==================== 聊天对话系统模型 ====================

// Conversation 对话会话模型
type Conversation struct {
	ID            uint           `json:"id" gorm:"type:int unsigned;primaryKey;comment:会话ID"`
	UserID        uint           `json:"user_id" gorm:"not null;index:idx_user_updated;comment:用户ID"`
	Title         string         `json:"title" gorm:"type:varchar(200);not null;comment:会话标题"`
	Messages      datatypes.JSON `json:"messages" gorm:"type:json;comment:完整会话消息[]*schema.Message"`
	CreatedAt     time.Time      `json:"created_at" gorm:"type:datetime;comment:创建时间"`
	UpdatedAt     time.Time      `json:"updated_at" gorm:"type:datetime;index:idx_user_updated;comment:更新时间"`
	LastMessageAt *time.Time     `json:"last_message_at" gorm:"type:datetime;comment:最后消息时间"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"comment:软删除时间"`
}
