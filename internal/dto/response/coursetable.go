package response

import (
	"time"

	"gorm.io/datatypes"
)

// CourseTableResponse 课程表响应
type CourseTableResponse struct {
	ClassID      string         `json:"class_id"`              // 班级ID
	Semester     string         `json:"semester"`              // 学期
	CourseData   datatypes.JSON `json:"course_data,omitempty"` // 课程数据 (有变化时才返回)
	LastModified int64          `json:"last_modified"`         // 数据最后修改时间戳
	HasChanges   bool           `json:"has_changes"`           // 是否有数据变化
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

type AdminCourseTableResponse struct {
	ID         uint           `json:"id"`
	ClassID    string         `json:"class_id"`
	Semester   string         `json:"semester"`
	CourseData datatypes.JSON `json:"course_data"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}
