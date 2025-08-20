package services

import (
	"fmt"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"

	"gorm.io/gorm"
)

type CourseTableService struct {
	db *gorm.DB
}

func NewCourseTableService(db *gorm.DB) *CourseTableService {
	return &CourseTableService{
		db: db,
	}
}

// GetUserCourseTable 获取用户课程表
func (s *CourseTableService) GetUserCourseTable(userID uint, semester string) (*response.CourseTableResponse, error) {
	// 先获取用户信息，获取其班级ID
	var user models.User
	if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, fmt.Errorf("查询用户信息失败: %v", err)
	}

	// 检查用户是否设置了班级ID
	if user.ClassID == "" {
		return nil, fmt.Errorf("用户尚未设置班级信息")
	}

	// 根据班级ID和学期查询课程表
	var courseTable models.CourseTable
	if err := s.db.Where("class_id = ? AND semester = ?", user.ClassID, semester).First(&courseTable).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("未找到该班级在指定学期的课程表")
		}
		return nil, fmt.Errorf("查询课程表失败: %v", err)
	}

	return &response.CourseTableResponse{
		ClassID:    courseTable.ClassID,
		Semester:   courseTable.Semester,
		CourseData: courseTable.CourseData,
	}, nil
}

// SearchClasses 模糊搜索班级
func (s *CourseTableService) SearchClasses(keyword string, page, size int) (*response.SearchClassResponse, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}

	offset := (page - 1) * size

	// 查询总数
	var total int64
	if err := s.db.Model(&models.CourseTable{}).
		Where("class_id LIKE ?", "%"+keyword+"%").
		Distinct("class_id").
		Count(&total).Error; err != nil {
		return nil, fmt.Errorf("查询班级总数失败: %v", err)
	}

	// 查询班级列表（去重）
	var courseTables []models.CourseTable
	if err := s.db.Select("DISTINCT class_id").
		Where("class_id LIKE ?", "%"+keyword+"%").
		Order("class_id ASC").
		Offset(offset).
		Limit(size).
		Find(&courseTables).Error; err != nil {
		return nil, fmt.Errorf("查询班级列表失败: %v", err)
	}

	// 转换为响应格式
	var classList []response.ClassInfo
	for _, ct := range courseTables {
		classList = append(classList, response.ClassInfo{
			ClassID:  ct.ClassID,
			Semester: ct.Semester,
		})
	}

	return &response.SearchClassResponse{
		List:  classList,
		Total: total,
		Page:  page,
		Size:  size,
	}, nil
}

// UpdateUserClass 更新用户班级
func (s *CourseTableService) UpdateUserClass(userID uint, classID string) (err error) {
	// 检查班级是否存在
	var count int64
	if err := s.db.Model(&models.CourseTable{}).Where("class_id = ?", classID).Count(&count).Error; err != nil {
		return fmt.Errorf("查询班级信息失败: %v", err)
	}

	if count == 0 {
		return fmt.Errorf("指定的班级不存在")
	}

	// 更新用户的班级ID
	if err := s.db.Model(&models.User{}).Where("id = ?", userID).Update("class_id", classID).Error; err != nil {
		return fmt.Errorf("更新用户班级失败: %v", err)
	}

	return nil
}
