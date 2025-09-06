package request

// SearchFailRateRequest 搜索挂科率请求
type SearchFailRateRequest struct {
	Keyword string `form:"keyword" json:"keyword"`      // 课程名关键词
	Page    int    `form:"page,default=1" json:"page"`  // 页码
	Size    int    `form:"size,default=10" json:"size"` // 每页数量
}
