package response

import (
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
)

// ContributionResponse 投稿响应
type ContributionResponse struct {
	ID             uint                           `json:"id"`
	UserID         uint                           `json:"user_id"`
	User           *UserSimpleResponse            `json:"user,omitempty"` // 投稿用户信息
	Title          string                         `json:"title"`
	Content        string                         `json:"content"`
	Categories     []NotificationCategoryResponse `json:"categories"` // 建议分类
	Status         models.UserContributionStatus  `json:"status"`
	ReviewerID     *uint                          `json:"reviewer_id"`
	Reviewer       *UserSimpleResponse            `json:"reviewer,omitempty"` // 审核者信息
	ReviewNote     string                         `json:"review_note"`
	NotificationID *uint                          `json:"notification_id"`
	Notification   *NotificationSimpleResponse    `json:"notification,omitempty"` // 关联通知信息
	PointsAwarded  uint                           `json:"points_awarded"`
	ReviewedAt     *time.Time                     `json:"reviewed_at"`
	CreatedAt      time.Time                      `json:"created_at"`
	UpdatedAt      time.Time                      `json:"updated_at"`
}

// ContributionSimpleResponse 投稿简单响应
type ContributionSimpleResponse struct {
	ID            uint                          `json:"id"`
	Title         string                        `json:"title"`
	Status        models.UserContributionStatus `json:"status"`
	PointsAwarded uint                          `json:"points_awarded"`
	CreatedAt     time.Time                     `json:"created_at"`
}
