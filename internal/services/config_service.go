package services

import (
	"context"
	"fmt"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/utils"
	"gorm.io/gorm"
)

type ConfigService struct {
	db *gorm.DB
}

func NewConfigService(db *gorm.DB) *ConfigService {
	return &ConfigService{db: db}
}

// Create 创建配置项（key唯一）
func (s *ConfigService) Create(ctx context.Context, key, value, valueType, description string) (*models.SystemConfig, error) {
	var cnt int64
	if err := s.db.WithContext(ctx).Model(&models.SystemConfig{}).Where("`key` = ?", key).Count(&cnt).Error; err != nil {
		return nil, err
	}
	if cnt > 0 {
		return nil, fmt.Errorf("key已存在")
	}

	if valueType == "" {
		valueType = "string"
	}

	m := &models.SystemConfig{
		Key:         key,
		Value:       value,
		ValueType:   valueType,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := s.db.WithContext(ctx).Create(m).Error; err != nil {
		return nil, err
	}
	return m, nil
}

// Update 根据key更新配置项
func (s *ConfigService) Update(ctx context.Context, key, value, valueType, description string) error {
	if valueType == "" {
		valueType = "string"
	}
	updates := map[string]any{
		"value":       value,
		"value_type":  valueType,
		"description": description,
		"updated_at":  time.Now(),
	}
	tx := s.db.WithContext(ctx).Model(&models.SystemConfig{}).Where("`key` = ?", key).Updates(updates)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return fmt.Errorf("未找到key")
	}
	return nil
}

// Delete 软删除（按key）
func (s *ConfigService) Delete(ctx context.Context, key string) error {
	tx := s.db.WithContext(ctx).Where("`key` = ?", key).Delete(&models.SystemConfig{})
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return fmt.Errorf("未找到key")
	}
	return nil
}

// GetByKey 通过key获取配置项
func (s *ConfigService) GetByKey(ctx context.Context, key string) (*models.SystemConfig, error) {
	var m models.SystemConfig
	if err := s.db.WithContext(ctx).Where("`key` = ?", key).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("未找到key")
		}
		return nil, err
	}
	return &m, nil
}

// SearchConfigs 搜索配置项，支持按key搜索，空query返回全部（分页版本）
func (s *ConfigService) SearchConfigs(ctx context.Context, query string, page, size int) ([]models.SystemConfig, int64, error) {
	var list []models.SystemConfig
	var total int64

	queryBuilder := s.db.WithContext(ctx).Model(&models.SystemConfig{})

	if query != "" {
		// 模糊搜索key或description
		queryBuilder = queryBuilder.WithContext(ctx).Where("`key` LIKE ? OR description LIKE ?", "%"+query+"%", "%"+query+"%")
	}

	// 先获取总数
	if err := queryBuilder.WithContext(ctx).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	pagination := utils.GetPagination(page, size)
	if err := queryBuilder.WithContext(ctx).Order("created_at DESC").
		Offset(pagination.Offset).
		Limit(pagination.Size).
		Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}
