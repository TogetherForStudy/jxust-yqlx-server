package request

// GetQuestionRequest 获取题目请求
type GetQuestionRequest struct {
	ProjectID uint `form:"project_id" binding:"required"`
	Random    bool `form:"random"` // true=乱序，false=顺序（默认顺序）
}

// RecordStudyRequest 记录学习请求（学习模式）
type RecordStudyRequest struct {
	QuestionID uint `json:"question_id" binding:"required"`
	ProjectID  uint `json:"project_id"`
}

// SubmitPracticeRequest 提交做题请求（仅记录做题次数）
type SubmitPracticeRequest struct {
	QuestionID uint `json:"question_id" binding:"required"`
	ProjectID  uint `json:"project_id"`
}

// GetProjectUsageRequest 获取项目使用统计请求
type GetProjectUsageRequest struct {
	ProjectID uint `form:"project_id"`
}

type AdminListQuestionProjectsRequest struct {
	Keyword  string `form:"keyword" json:"keyword"`
	IsActive *bool  `form:"is_active" json:"is_active"`
	Page     int    `form:"page" json:"page"`
	PageSize int    `form:"page_size" json:"page_size" binding:"min=1,max=100"`
}

type AdminCreateQuestionProjectRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Version     int    `json:"version"`
	Sort        int    `json:"sort"`
	IsActive    *bool  `json:"is_active"`
}

type AdminUpdateQuestionProjectRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Version     *int    `json:"version"`
	Sort        *int    `json:"sort"`
	IsActive    *bool   `json:"is_active"`
}

type AdminListQuestionsRequest struct {
	ProjectID   uint   `form:"project_id" json:"project_id"`
	Keyword     string `form:"keyword" json:"keyword"`
	IsActive    *bool  `form:"is_active" json:"is_active"`
	ParentID    *uint  `form:"parent_id" json:"parent_id"`
	Type        *int8  `form:"type" json:"type"`
	SortMin     *int   `form:"sort_min" json:"sort_min"`
	SortMax     *int   `form:"sort_max" json:"sort_max"`
	CreatedFrom string `form:"created_from" json:"created_from"`
	CreatedTo   string `form:"created_to" json:"created_to"`
	Page        int    `form:"page" json:"page"`
	PageSize    int    `form:"page_size" json:"page_size" binding:"min=1,max=100"`
}

type AdminCreateQuestionRequest struct {
	ProjectID uint     `json:"project_id" binding:"required"`
	ParentID  *uint    `json:"parent_id"`
	Type      int8     `json:"type" binding:"required"`
	Title     string   `json:"title" binding:"required"`
	Options   []string `json:"options"`
	Answer    string   `json:"answer" binding:"required"`
	Sort      int      `json:"sort"`
	IsActive  *bool    `json:"is_active"`
}

type AdminUpdateQuestionRequest struct {
	ProjectID *uint     `json:"project_id"`
	ParentID  *uint     `json:"parent_id"`
	Type      *int8     `json:"type"`
	Title     *string   `json:"title"`
	Options   *[]string `json:"options"`
	Answer    *string   `json:"answer"`
	Sort      *int      `json:"sort"`
	IsActive  *bool     `json:"is_active"`
}
