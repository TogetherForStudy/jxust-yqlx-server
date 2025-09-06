package services

import (
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/utils"
	"gorm.io/gorm"
)

type FailRateService struct {
	db *gorm.DB
}

func NewFailRateService(db *gorm.DB) *FailRateService {
	return &FailRateService{db: db}
}

// Search 按课程名关键词分页查询，默认按 failrate 降序
func (s *FailRateService) Search(keyword string, page, size int) ([]models.FailRate, int64, error) {
	var list []models.FailRate
	var total int64

	query := s.db.Model(&models.FailRate{})
	if keyword != "" {
		query = query.Where("course_name LIKE ?", "%"+keyword+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	pagination := utils.GetPagination(page, size)
	if err := query.Order("fail_rate DESC").
		Offset(pagination.Offset).
		Limit(pagination.Size).
		Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

// Top 获取随机 N 条（无筛选）
func (s *FailRateService) Top(limit int) ([]models.FailRate, error) {
	if limit <= 0 {
		limit = 10
	}
	var list []models.FailRate
	if err := s.db.Model(&models.FailRate{}).
		Order("RAND()").
		Limit(limit).
		Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
