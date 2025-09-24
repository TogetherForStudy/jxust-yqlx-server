package response

import (
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
)

// UserPointsResponse 用户积分响应
type UserPointsResponse struct {
	UserID uint                `json:"user_id"`
	User   *UserSimpleResponse `json:"user,omitempty"` // 用户信息（管理员查看时）
	Points uint                `json:"points"`
}

// PointsTransactionResponse 积分交易记录响应
type PointsTransactionResponse struct {
	ID          uint                           `json:"id"`
	UserID      uint                           `json:"user_id"`
	User        *UserSimpleResponse            `json:"user,omitempty"` // 用户信息（管理员查看时）
	Type        models.PointsTransactionType   `json:"type"`
	Source      models.PointsTransactionSource `json:"source"`
	Points      int                            `json:"points"`
	Description string                         `json:"description"`
	RelatedID   *uint                          `json:"related_id"`
	CreatedAt   time.Time                      `json:"created_at"`
}
