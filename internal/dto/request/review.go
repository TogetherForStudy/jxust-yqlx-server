package request

import "github.com/TogetherForStudy/jxust-yqlx-server/internal/models"

// CreateReviewRequest 创建评价请求
type CreateReviewRequest struct {
	TeacherName string                 `json:"teacher_name" binding:"required"`
	Campus      string                 `json:"campus" binding:"required"`
	CourseName  string                 `json:"course_name" binding:"required"`
	Content     string                 `json:"content" binding:"required,max=200"`
	Attitude    models.TeacherAttitude `json:"attitude" binding:"required,oneof=1 2 3"`
}
