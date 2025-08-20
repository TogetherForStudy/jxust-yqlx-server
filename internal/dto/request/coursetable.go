package request

// GetCourseTableRequest 获取课程表请求
type GetCourseTableRequest struct {
	Semester string `form:"semester" binding:"required" json:"semester"` // 学期
}

// SearchClassRequest 搜索班级请求
type SearchClassRequest struct {
	Keyword string `form:"keyword" binding:"required" json:"keyword"` // 搜索关键字
	Page    int    `form:"page,default=1" json:"page"`                // 页码
	Size    int    `form:"size,default=10" json:"size"`               // 每页数量
}

// UpdateUserClassRequest 更新用户班级请求
type UpdateUserClassRequest struct {
	ClassID string `json:"class_id" binding:"required"` // 班级ID
}
