package response

import (
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
)

// QuestionProjectResponse 项目响应
type QuestionProjectResponse struct {
	ID            uint      `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Version       int       `json:"version"`
	Sort          int       `json:"sort"`
	IsActive      bool      `json:"is_active"`
	QuestionCount int64     `json:"question_count"` // 题目总数
	UserCount     int64     `json:"user_count"`     // 使用过该项目的用户数
	UsageCount    int64     `json:"usage_count"`    // 项目内题目总刷题次数（学习+练习）
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// QuestionListResponse 题目列表响应（只返回ID数组）
type QuestionListResponse struct {
	QuestionIDs []uint `json:"question_ids"` // 题目ID数组（顺序或乱序）
}

// QuestionResponse 题目响应（包含完整信息）
type QuestionResponse struct {
	ID            uint               `json:"id"`
	ProjectID     uint               `json:"project_id"`
	ParentID      *uint              `json:"parent_id,omitempty"` // 父题目ID
	Type          int8               `json:"type"`                // 1=选择题，2=简答题
	Title         string             `json:"title"`
	Options       []string           `json:"options,omitempty"`       // 选择题选项
	Answer        string             `json:"answer"`                  // 正确答案
	Sort          int                `json:"sort"`                    // 排序
	StudyCount    int                `json:"study_count"`             // 学习次数
	PracticeCount int                `json:"practice_count"`          // 做题次数
	SubQuestions  []QuestionResponse `json:"sub_questions,omitempty"` // 子题（题目分组）
}

// ToQuestionProjectResponse 转换为项目响应
func ToQuestionProjectResponse(project *models.QuestionProject, questionCount int64, userCount, usageCount int64) QuestionProjectResponse {
	return QuestionProjectResponse{
		ID:            project.ID,
		Name:          project.Name,
		Description:   project.Description,
		Version:       project.Version,
		Sort:          project.Sort,
		IsActive:      project.IsActive,
		QuestionCount: questionCount,
		UserCount:     userCount,
		UsageCount:    usageCount,
		CreatedAt:     project.CreatedAt,
		UpdatedAt:     project.UpdatedAt,
	}
}
