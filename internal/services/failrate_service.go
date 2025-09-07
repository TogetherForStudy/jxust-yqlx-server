package services

import (
	"math/rand"

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

// Rand 获取随机 N 条（无筛选）- 高性能版本，避免使用ORDER BY RAND()
func (s *FailRateService) Rand(limit int) ([]models.FailRate, error) {
	if limit <= 0 {
		limit = 10
	}

	// 1. 获取总记录数
	var total int64
	if err := s.db.Model(&models.FailRate{}).Count(&total).Error; err != nil {
		return nil, err
	}

	// 如果总数不够，直接返回所有记录
	if total <= int64(limit) {
		var list []models.FailRate
		if err := s.db.Model(&models.FailRate{}).Find(&list).Error; err != nil {
			return nil, err
		}
		return list, nil
	}

	// 2. 生成随机偏移量，确保有足够的记录可以获取
	maxOffset := int(total) - limit
	if maxOffset < 0 {
		maxOffset = 0
	}

	// Go 1.20+ 全局随机数生成器已自动初始化，无需手动设置种子
	randomOffset := rand.Intn(maxOffset + 1)

	// 3. 使用随机偏移量获取记录
	var list []models.FailRate
	if err := s.db.Model(&models.FailRate{}).
		Offset(randomOffset).
		Limit(limit).
		Find(&list).Error; err != nil {
		return nil, err
	}

	return list, nil
}
