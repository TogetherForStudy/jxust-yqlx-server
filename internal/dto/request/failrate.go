package request

// SearchFailRateRequest 搜索挂科率请求
type SearchFailRateRequest struct {
	Keyword string `form:"keyword" json:"keyword"`      // 课程名关键词
	Page    int    `form:"page,default=1" json:"page"`  // 页码
	Size    int    `form:"size,default=10" json:"size"` // 每页数量
}

type AdminListFailRatesRequest struct {
	Keyword    string `form:"keyword" json:"keyword"`
	Department string `form:"department" json:"department"`
	Semester   string `form:"semester" json:"semester"`
	Page       int    `form:"page" json:"page"`
	PageSize   int    `form:"page_size" json:"page_size" binding:"min=1,max=100"`
}

type AdminCreateFailRateRequest struct {
	CourseName string  `json:"course_name" binding:"required"`
	Department string  `json:"department" binding:"required"`
	Semester   string  `json:"semester" binding:"required"`
	FailRate   float64 `json:"failrate" binding:"required"`
}

type AdminUpdateFailRateRequest struct {
	CourseName *string  `json:"course_name"`
	Department *string  `json:"department"`
	Semester   *string  `json:"semester"`
	FailRate   *float64 `json:"failrate"`
}
