package services

import (
	"fmt"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"gorm.io/gorm"
)

type ConfigService struct {
	db *gorm.DB
}

func NewConfigService(db *gorm.DB) *ConfigService {
	return &ConfigService{db: db}
}

// Create 创建配置项（key唯一）
func (s *ConfigService) Create(key, value, valueType, description string) (*models.SystemConfig, error) {
	var cnt int64
	if err := s.db.Model(&models.SystemConfig{}).Where("`key` = ?", key).Count(&cnt).Error; err != nil {
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
	if err := s.db.Create(m).Error; err != nil {
		return nil, err
	}
	return m, nil
}

// Update 根据key更新配置项
func (s *ConfigService) Update(key, value, valueType, description string) error {
	if valueType == "" {
		valueType = "string"
	}
	updates := map[string]any{
		"value":       value,
		"value_type":  valueType,
		"description": description,
		"updated_at":  time.Now(),
	}
	tx := s.db.Model(&models.SystemConfig{}).Where("`key` = ?", key).Updates(updates)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return fmt.Errorf("未找到key")
	}
	return nil
}

// Delete 软删除（按key）
func (s *ConfigService) Delete(key string) error {
	tx := s.db.Where("`key` = ?", key).Delete(&models.SystemConfig{})
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return fmt.Errorf("未找到key")
	}
	return nil
}

// GetByKey 通过key获取配置项
func (s *ConfigService) GetByKey(key string) (*models.SystemConfig, error) {
	var m models.SystemConfig
	if err := s.db.Where("`key` = ?", key).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("未找到key")
		}
		return nil, err
	}
	return &m, nil
}
