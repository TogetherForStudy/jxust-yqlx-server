package constant

// User Status
const (
	UserStatusNormal   = 1 // 正常
	UserStatusDisabled = 2 // 禁用
)

// Teacher Attitude
const (
	AttitudeNeutral   = 3 // 中立
	AttitudeRecommend = 1 // 推荐
	AttitudeAvoid     = 2 // 避雷
)

// Teacher Review Status
const (
	TeacherReviewStatusPending  = 1 // 待审核
	TeacherReviewStatusApproved = 2 // 已通过
	TeacherReviewStatusRejected = 3 // 已拒绝
)

// Notification Publisher Type
const (
	NotificationPublisherOperator = 1 // 运营发布
	NotificationPublisherUser     = 2 // 用户投稿
)

// Notification Status
const (
	NotificationStatusDraft     = 1 // 草稿
	NotificationStatusPending   = 2 // 待审核
	NotificationStatusPublished = 3 // 已发布
	NotificationStatusDeleted   = 4 // 已删除
)

// Notification Approval Status
const (
	NotificationApprovalStatusApproved = 1 // 同意
	NotificationApprovalStatusRejected = 2 // 拒绝
)

// Points Transaction Type
const (
	PointsTransactionTypeEarn  = 1 // 获得
	PointsTransactionTypeSpend = 2 // 消耗
)

// Points Transaction Source
const (
	PointsTransactionSourceDailyLogin   = "daily_login"  // 每日登录
	PointsTransactionSourceReview       = "review"       // 发布评价并审核通过
	PointsTransactionSourceContribution = "contribution" // 投稿信息并审核通过
	PointsTransactionSourceRedeem       = "redeem"       // 兑换奖品
	PointsTransactionSourceAdminGrant   = "admin_grant"  // 管理员手动赋予
)

// User Contribution Status
const (
	UserContributionStatusPending  = 1 // 待审核
	UserContributionStatusApproved = 2 // 已采纳
	UserContributionStatusRejected = 3 // 已拒绝
)

// Study Task Priority
const (
	StudyTaskPriorityHigh   = 1 // 高
	StudyTaskPriorityMedium = 2 // 中
	StudyTaskPriorityLow    = 3 // 低
)

// Study Task Status
const (
	StudyTaskStatusPending   = 1 // 待完成
	StudyTaskStatusCompleted = 2 // 已完成
)

// Material Log Type
const (
	MaterialLogTypeSearch   = 1 // 搜索
	MaterialLogTypeView     = 2 // 查看
	MaterialLogTypeRating   = 3 // 评分
	MaterialLogTypeDownload = 4 // 下载
)

// Question Type
const (
	QuestionTypeChoice = 1 // 选择题
	QuestionTypeEssay  = 2 // 简答题
)
