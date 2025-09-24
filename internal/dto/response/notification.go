package response

import (
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
)

// NotificationResponse 通知响应
type NotificationResponse struct {
	ID              uint                             `json:"id"`
	Title           string                           `json:"title"`
	Content         string                           `json:"content"`
	PublisherID     uint                             `json:"publisher_id"`
	PublisherType   models.NotificationPublisherType `json:"publisher_type"`
	Publisher       *UserSimpleResponse              `json:"publisher,omitempty"` // 发布者信息
	ContributorID   *uint                            `json:"contributor_id"`
	Contributor     *UserSimpleResponse              `json:"contributor,omitempty"` // 投稿者信息
	Categories      []NotificationCategoryResponse   `json:"categories"`
	Status          models.NotificationStatus        `json:"status"`
	Schedule        *models.ScheduleData             `json:"schedule,omitempty"` // 日程信息
	ViewCount       uint                             `json:"view_count"`
	PublishedAt     *time.Time                       `json:"published_at"`
	CreatedAt       time.Time                        `json:"created_at"`
	UpdatedAt       time.Time                        `json:"updated_at"`
	Approvals       []NotificationApprovalResponse   `json:"approvals,omitempty"`        // 审核记录
	ApprovalSummary *NotificationApprovalSummary     `json:"approval_summary,omitempty"` // 审核进度汇总
}

// NotificationApprovalSummary 通知审核进度汇总
type NotificationApprovalSummary struct {
	TotalReviewers int64   `json:"total_reviewers"` // 总审核人数（管理员+运营）
	ApprovedCount  int64   `json:"approved_count"`  // 已通过人数
	RejectedCount  int64   `json:"rejected_count"`  // 已拒绝人数
	PendingCount   int64   `json:"pending_count"`   // 待审核人数
	ApprovalRate   float64 `json:"approval_rate"`   // 通过率
	RequiredRate   float64 `json:"required_rate"`   // 所需通过率（0.5）
	CanPublish     bool    `json:"can_publish"`     // 是否可以发布
}

// NotificationSimpleResponse 通知简单响应
type NotificationSimpleResponse struct {
	ID              uint                           `json:"id"`
	Title           string                         `json:"title"`
	Categories      []NotificationCategoryResponse `json:"categories"`
	Status          models.NotificationStatus      `json:"status"`             // 添加状态字段
	Schedule        *models.ScheduleData           `json:"schedule,omitempty"` // 日程信息
	ViewCount       uint                           `json:"view_count"`
	PublishedAt     *time.Time                     `json:"published_at"`
	CreatedAt       time.Time                      `json:"created_at"`
	ApprovalSummary *NotificationApprovalSummary   `json:"approval_summary,omitempty"` // 审核进度汇总
}

// NotificationApprovalResponse 通知审核响应
type NotificationApprovalResponse struct {
	ID        uint                              `json:"id"`
	Reviewer  UserSimpleResponse                `json:"reviewer"`   // 审核者信息
	Status    models.NotificationApprovalStatus `json:"status"`     // 审核状态
	Note      string                            `json:"note"`       // 审核备注
	CreatedAt time.Time                         `json:"created_at"` // 审核时间
}

// NotificationCategoryResponse 通知分类响应
type NotificationCategoryResponse struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Sort      int       `json:"sort"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserSimpleResponse 用户简单响应
type UserSimpleResponse struct {
	ID       uint   `json:"id"`
	Nickname string `json:"nickname"`
}
