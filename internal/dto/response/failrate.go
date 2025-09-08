package response

// FailRateItem 挂科率列表项
type FailRateItem struct {
	ID         uint    `json:"id"`
	CourseName string  `json:"course_name"`
	Department string  `json:"department"`
	Semester   string  `json:"semester"`
	FailRate   float64 `json:"failrate"`
}

// FailRateListResponse 搜索/Top 返回结构
type FailRateListResponse struct {
	List  []FailRateItem `json:"list"`
	Total int64          `json:"total,omitempty"`
	Page  int            `json:"page,omitempty"`
	Size  int            `json:"size,omitempty"`
}
