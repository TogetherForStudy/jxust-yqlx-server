package request

// GetQuestionRequest 获取题目请求
type GetQuestionRequest struct {
	ProjectID uint `form:"project_id" binding:"required"`
	Random    bool `form:"random"` // true=乱序，false=顺序（默认顺序）
}

// RecordStudyRequest 记录学习请求（学习模式）
type RecordStudyRequest struct {
	QuestionID uint `json:"question_id" binding:"required"`
}

// SubmitPracticeRequest 提交做题请求（仅记录做题次数）
type SubmitPracticeRequest struct {
	QuestionID uint `json:"question_id" binding:"required"`
}

// GetProjectUsageRequest 获取项目使用统计请求
type GetProjectUsageRequest struct {
	ProjectID uint `form:"project_id"`
}
