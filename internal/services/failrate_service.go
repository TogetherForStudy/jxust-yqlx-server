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

// Rand 获取随机 N 条（无筛选）- 全表均匀抽样，避免使用 ORDER BY RAND()
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

	// 2. 生成不重复的随机偏移集合，从全表均匀抽样 N 条
	uniqueOffsets := make(map[int]struct{}, limit)
	maxIndex := int(total) - 1
	for len(uniqueOffsets) < limit {
		idx := rand.Intn(maxIndex + 1)
		uniqueOffsets[idx] = struct{}{}
	}

	list := make([]models.FailRate, 0, limit)
	for offset := range uniqueOffsets {
		var item models.FailRate
		if err := s.db.Model(&models.FailRate{}).
			Order("id").
			Offset(offset).
			Limit(1).
			Find(&item).Error; err != nil {
			return nil, err
		}
		list = append(list, item)
	}

	return list, nil
}
