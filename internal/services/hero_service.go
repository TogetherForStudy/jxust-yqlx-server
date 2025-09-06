package services

import (
	"fmt"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
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

// ListAll 返回仅名称的字符串数组，按 sort 升序
func (s *HeroService) ListAll() ([]string, error) {
	var list []models.Hero
	if err := s.db.Model(&models.Hero{}).Order("sort ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	names := make([]string, 0, len(list))
	for _, it := range list {
		names = append(names, it.Name)
	}
	return names, nil
}
