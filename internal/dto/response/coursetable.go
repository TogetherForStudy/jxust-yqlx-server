package response

import "gorm.io/datatypes"

// CourseTableResponse 课程表响应
type CourseTableResponse struct {
	ClassID    string         `json:"class_id"`    // 班级ID
	Semester   string         `json:"semester"`    // 学期
	CourseData datatypes.JSON `json:"course_data"` // 课程数据
}

// ClassInfo 班级信息
type ClassInfo struct {
	ClassID  string `json:"class_id"` // 班级ID
	Semester string `json:"semester"` // 学期
}

// SearchClassResponse 搜索班级响应
type SearchClassResponse struct {
	List  []ClassInfo `json:"list"`  // 班级列表
	Total int64       `json:"total"` // 总数
	Page  int         `json:"page"`  // 当前页
	Size  int         `json:"size"`  // 每页数量
}
