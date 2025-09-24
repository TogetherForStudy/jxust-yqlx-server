package response

import (
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
)

// StudyTaskResponse 学习任务响应
type StudyTaskResponse struct {
	ID          uint                     `json:"id"`
	UserID      uint                     `json:"user_id"`
	Title       string                   `json:"title"`
	Description string                   `json:"description"`
	DueDate     *time.Time               `json:"due_date"`
	Priority    models.StudyTaskPriority `json:"priority"`
	Status      models.StudyTaskStatus   `json:"status"`
	CompletedAt *time.Time               `json:"completed_at"`
	DaysLeft    *int                     `json:"days_left"`  // 剩余天数（负数表示已过期）
	IsOverdue   bool                     `json:"is_overdue"` // 是否过期
	CreatedAt   time.Time                `json:"created_at"`
	UpdatedAt   time.Time                `json:"updated_at"`
}

// StudyTaskStatsResponse 学习任务统计响应
type StudyTaskStatsResponse struct {
	TotalCount     int `json:"total_count"`     // 总任务数
	PendingCount   int `json:"pending_count"`   // 待完成数量
	CompletedCount int `json:"completed_count"` // 已完成数量
}
