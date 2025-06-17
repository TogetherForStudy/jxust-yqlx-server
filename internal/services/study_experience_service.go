package services

import (
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/utils"

	"gorm.io/gorm"
)

type StudyExperienceService struct {
	db *gorm.DB
}

func NewStudyExperienceService(db *gorm.DB) *StudyExperienceService {
	return &StudyExperienceService{
		db: db,
	}
}

// CreateExperienceRequest 创建备考经验请求
type CreateExperienceRequest struct {
	Campus     string `json:"campus" binding:"required"`
	CourseName string `json:"course_name" binding:"required"`
	Content    string `json:"content" binding:"required"`
}

// CreateExperience 创建备考经验-User
func (s *StudyExperienceService) CreateExperience(userID uint, req *CreateExperienceRequest) error {
	// 创建经验分享
	experience := &models.StudyExperience{
		UserID:     userID,
		Campus:     req.Campus,
		CourseName: req.CourseName,
		Content:    req.Content,
		Status:     models.StudyExperienceStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	return s.db.Create(experience).Error
}

// GetExperiences 获取备考经验列表-Admin
func (s *StudyExperienceService) GetExperiences(page, size int, status models.StudyExperienceStatus) ([]models.StudyExperience, int64, error) {
	var experiences []models.StudyExperience
	var total int64

	query := s.db.Model(&models.StudyExperience{})

	// 按状态筛选
	if status > 0 {
		query = query.Where("status = ?", status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	pagination := utils.GetPagination(page, size)
	err := query.Order("created_at DESC").
		Offset(pagination.Offset).
		Limit(pagination.Size).
		Find(&experiences).Error

	return experiences, total, err
}

// GetApprovedExperiences 获取已审核通过的备考经验-User
func (s *StudyExperienceService) GetApprovedExperiences(page, size int, course string) ([]models.StudyExperience, int64, error) {
	var experiences []models.StudyExperience
	var total int64

	query := s.db.Model(&models.StudyExperience{}).
		Where("status = ?", models.StudyExperienceStatusApproved)

	// 按课程筛选
	if course != "" {
		query = query.Where("course LIKE ?", "%"+course+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	pagination := utils.GetPagination(page, size)
	err := query.Order("created_at DESC").
		Offset(pagination.Offset).
		Limit(pagination.Size).
		Find(&experiences).Error

	return experiences, total, err
}

// GetUserExperiences 获取用户的经验分享记录-Admin
func (s *StudyExperienceService) GetUserExperiences(userID uint, page, size int) ([]models.StudyExperience, int64, error) {
	var experiences []models.StudyExperience
	var total int64

	query := s.db.Model(&models.StudyExperience{}).
		Where("user_id = ?", userID)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	pagination := utils.GetPagination(page, size)
	err := query.Order("created_at DESC").
		Offset(pagination.Offset).
		Limit(pagination.Size).
		Find(&experiences).Error

	return experiences, total, err
}

// GetExperienceByID 根据ID获取备考经验-Admin
func (s *StudyExperienceService) GetExperienceByID(experienceID uint) (*models.StudyExperience, error) {
	var experience models.StudyExperience
	err := s.db.First(&experience, experienceID).Error
	if err != nil {
		return nil, err
	}

	// 增加查看次数
	s.db.Model(&models.StudyExperience{}).Where("id = ?", experienceID).
		UpdateColumn("view_count", gorm.Expr("view_count + 1"))

	return &experience, nil
}

// ApproveExperience 审核通过经验分享-Admin
func (s *StudyExperienceService) ApproveExperience(experienceID uint, adminNote string) error {
	return s.db.Model(&models.StudyExperience{}).
		Where("id = ?", experienceID).
		Updates(map[string]interface{}{
			"status":     models.StudyExperienceStatusApproved,
			"admin_note": adminNote,
			"updated_at": time.Now(),
		}).Error
}

// RejectExperience 审核拒绝经验分享-Admin
func (s *StudyExperienceService) RejectExperience(experienceID uint, adminNote string) error {
	return s.db.Model(&models.StudyExperience{}).
		Where("id = ?", experienceID).
		Updates(map[string]interface{}{
			"status":     models.StudyExperienceStatusRejected,
			"admin_note": adminNote,
			"updated_at": time.Now(),
		}).Error
}

// LikeExperience 点赞经验分享-User
func (s *StudyExperienceService) LikeExperience(experienceID uint) error {
	return s.db.Model(&models.StudyExperience{}).
		Where("id = ?", experienceID).
		UpdateColumn("like_count", gorm.Expr("like_count + 1")).Error
}
