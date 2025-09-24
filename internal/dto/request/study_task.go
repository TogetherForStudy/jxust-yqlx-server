package request

// CreateStudyTaskRequest 创建学习任务请求
type CreateStudyTaskRequest struct {
	Title       string `json:"title" binding:"required,max=200"`         // 任务标题
	Description string `json:"description"`                              // 任务描述
	DueDate     string `json:"due_date"`                                 // 截止日期 YYYY-MM-DD
	Priority    uint8  `json:"priority" binding:"omitempty,oneof=1 2 3"` // 优先级：1=高，2=中，3=低
}

// UpdateStudyTaskRequest 更新学习任务请求
type UpdateStudyTaskRequest struct {
	Title       string `json:"title" binding:"omitempty,max=200"`        // 任务标题
	Description string `json:"description"`                              // 任务描述
	DueDate     string `json:"due_date"`                                 // 截止日期 YYYY-MM-DD
	Priority    *uint8 `json:"priority" binding:"omitempty,oneof=1 2 3"` // 优先级：1=高，2=中，3=低
	Status      *uint8 `json:"status" binding:"omitempty,oneof=1 2"`     // 状态：1=待完成，2=已完成
}

// GetStudyTasksRequest 获取学习任务列表请求
type GetStudyTasksRequest struct {
	Page     int    `form:"page" binding:"min=1"`         // 页码
	Size     int    `form:"size" binding:"min=1,max=100"` // 每页数量
	Status   *uint8 `form:"status" binding:"omitempty"`   // 状态过滤：1=待完成，2=已完成
	Priority *uint8 `form:"priority" binding:"omitempty"` // 优先级过滤：1=高，2=中，3=低
	Keyword  string `form:"keyword" binding:"omitempty"`  // 关键词搜索
}

// CompleteTaskRequest 完成任务请求
type CompleteTaskRequest struct {
	IsCompleted bool `json:"is_completed"` // 是否已完成
}
