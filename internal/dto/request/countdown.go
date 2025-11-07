package request

// CreateCountdownRequest 创建倒数日请求
type CreateCountdownRequest struct {
	Title       string `json:"title" binding:"required,max=100"` // 倒数日标题
	Description string `json:"description"`                      // 描述
	TargetDate  string `json:"target_date" binding:"required"`   // 目标日期 YYYY-MM-DD
}

// UpdateCountdownRequest 更新倒数日请求
type UpdateCountdownRequest struct {
	Title       *string `json:"title" binding:"omitempty,max=100"` // 倒数日标题
	Description *string `json:"description"`                       // 描述
	TargetDate  *string `json:"target_date"`                       // 目标日期 YYYY-MM-DD
}
