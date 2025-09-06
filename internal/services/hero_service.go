package services

import (
	"fmt"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/utils"
	"gorm.io/gorm"
)

type HeroService struct {
	db *gorm.DB
}

func NewHeroService(db *gorm.DB) *HeroService {
	return &HeroService{db: db}
}

func (s *HeroService) Create(name string, sort int, isShow bool) (*models.Hero, error) {
	// name 唯一
	var cnt int64
	if err := s.db.Model(&models.Hero{}).Where("name = ?", name).Count(&cnt).Error; err != nil {
		return nil, err
	}
	if cnt > 0 {
		return nil, fmt.Errorf("名称已存在")
	}

	hero := &models.Hero{
		Name:      name,
		Sort:      sort,
		IsShow:    isShow,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.db.Create(hero).Error; err != nil {
		return nil, err
	}
	return hero, nil
}

func (s *HeroService) Update(id uint, name string, sort int, isShow bool) error {
	// 如果修改 name，需要确保唯一（排除自身）
	var cnt int64
	if err := s.db.Model(&models.Hero{}).Where("name = ? AND id <> ?", name, id).Count(&cnt).Error; err != nil {
		return err
	}
	if cnt > 0 {
		return fmt.Errorf("名称已存在")
	}
	return s.db.Model(&models.Hero{}).Where("id = ?", id).Updates(map[string]any{
		"name":       name,
		"sort":       sort,
		"is_show":    isShow,
		"updated_at": time.Now(),
	}).Error
}

func (s *HeroService) Delete(id uint) error {
	return s.db.Unscoped().Delete(&models.Hero{}, id).Error
}

func (s *HeroService) Get(id uint) (*models.Hero, error) {
	var m models.Hero
	if err := s.db.First(&m, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("未找到")
		}
		return nil, err
	}
	return &m, nil
}

// ListAll 返回仅名称的字符串数组，按 sort 升序（只返回is_show=true的）
func (s *HeroService) ListAll() ([]string, error) {
	var list []models.Hero
	if err := s.db.Model(&models.Hero{}).Where("is_show = ?", true).Order("sort ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	names := make([]string, 0, len(list))
	for _, it := range list {
		names = append(names, it.Name)
	}
	return names, nil
}

// SearchHeroes 搜索英雄，支持按名称搜索和是否显示过滤，空query返回全部（分页版本）
func (s *HeroService) SearchHeroes(query string, isShow *bool, page, size int) ([]models.Hero, int64, error) {
	var list []models.Hero
	var total int64

	queryBuilder := s.db.Model(&models.Hero{})

	if query != "" {
		// 模糊搜索名称
		queryBuilder = queryBuilder.Where("name LIKE ?", "%"+query+"%")
	}

	if isShow != nil {
		// 根据is_show字段过滤
		queryBuilder = queryBuilder.Where("is_show = ?", *isShow)
	}

	// 先获取总数
	if err := queryBuilder.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	pagination := utils.GetPagination(page, size)
	if err := queryBuilder.Order("sort ASC").
		Offset(pagination.Offset).
		Limit(pagination.Size).
		Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}
