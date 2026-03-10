package services

import (
	"context"
	stdjson "encoding/json"
	"fmt"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/apperr"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const maxGPABackupsPerUser int64 = 6

type GPABackupService struct {
	db *gorm.DB
}

func NewGPABackupService(db *gorm.DB) *GPABackupService {
	return &GPABackupService{db: db}
}

func (s *GPABackupService) CreateBackup(ctx context.Context, userID uint, rawData []byte) (*response.GPABackupResponse, error) {
	if len(rawData) == 0 || !stdjson.Valid(rawData) {
		err := apperr.New(constant.CommonBadRequest)
		err.Message = "请求体必须是有效 JSON"
		return nil, err
	}

	var count int64
	if err := s.db.WithContext(ctx).Model(&models.GPABackup{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询绩点备份数量失败: %w", err))
	}
	if count >= maxGPABackupsPerUser {
		err := apperr.New(constant.CommonConflict)
		err.Message = "每个用户最多只能保存 6 组绩点数据"
		return nil, err
	}

	backup := models.GPABackup{
		UserID: userID,
		Data:   datatypes.JSON(rawData),
	}
	if err := s.db.WithContext(ctx).Create(&backup).Error; err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("创建绩点备份失败: %w", err))
	}

	resp := toGPABackupResponse(backup)
	return &resp, nil
}

func (s *GPABackupService) ListBackups(ctx context.Context, userID uint) ([]response.GPABackupResponse, error) {
	var backups []models.GPABackup
	if err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC, id DESC").
		Find(&backups).Error; err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询绩点备份列表失败: %w", err))
	}

	result := make([]response.GPABackupResponse, 0, len(backups))
	for _, item := range backups {
		result = append(result, toGPABackupResponse(item))
	}
	return result, nil
}

func (s *GPABackupService) GetBackupByID(ctx context.Context, userID, backupID uint) (*response.GPABackupResponse, error) {
	var backup models.GPABackup
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", backupID, userID).First(&backup).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.New(constant.CommonNotFound)
		}
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询绩点备份详情失败: %w", err))
	}

	resp := toGPABackupResponse(backup)
	return &resp, nil
}

func (s *GPABackupService) DeleteBackup(ctx context.Context, userID, backupID uint) error {
	result := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", backupID, userID).Delete(&models.GPABackup{})
	if result.Error != nil {
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("删除绩点备份失败: %w", result.Error))
	}
	if result.RowsAffected == 0 {
		return apperr.New(constant.CommonNotFound)
	}
	return nil
}

func toGPABackupResponse(backup models.GPABackup) response.GPABackupResponse {
	return response.GPABackupResponse{
		ID:        backup.ID,
		Data:      backup.Data,
		CreatedAt: backup.CreatedAt,
		UpdatedAt: backup.UpdatedAt,
	}
}
