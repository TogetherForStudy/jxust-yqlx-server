package services

import (
	"context"
	"errors"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"

	"gorm.io/gorm"
)

type CountdownService struct {
	db *gorm.DB
}

func NewCountdownService(db *gorm.DB) *CountdownService {
	return &CountdownService{
		db: db,
	}
}

// CreateCountdown 创建倒数日
func (s *CountdownService) CreateCountdown(ctx context.Context, userID uint, req *request.CreateCountdownRequest) (*response.CountdownResponse, error) {
	// 解析目标日期
	targetDate, err := time.Parse("2006-01-02", req.TargetDate)
	if err != nil {
		return nil, errors.New("目标日期格式错误")
	}

	// 创建倒数日
	countdown := models.Countdown{
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		TargetDate:  targetDate,
	}

	if err := s.db.WithContext(ctx).Create(&countdown).Error; err != nil {
		return nil, err
	}

	return &response.CountdownResponse{
		ID:          countdown.ID,
		UserID:      countdown.UserID,
		Title:       countdown.Title,
		Description: countdown.Description,
		TargetDate:  countdown.TargetDate,
		CreatedAt:   countdown.CreatedAt,
		UpdatedAt:   countdown.UpdatedAt,
	}, nil
}

// GetCountdowns 获取用户倒数日列表
func (s *CountdownService) GetCountdowns(ctx context.Context, userID uint) ([]response.CountdownResponse, error) {
	var countdowns []models.Countdown

	// 构建查询
	query := s.db.WithContext(ctx).Where("user_id = ?", userID)

	// 查询数据
	if err := query.Order("created_at DESC").Find(&countdowns).Error; err != nil {
		return nil, err
	}

	// 转换为响应格式
	var countdownResponses []response.CountdownResponse
	for _, countdown := range countdowns {
		countdownResponses = append(countdownResponses, response.CountdownResponse{
			ID:          countdown.ID,
			UserID:      countdown.UserID,
			Title:       countdown.Title,
			Description: countdown.Description,
			TargetDate:  countdown.TargetDate,
			CreatedAt:   countdown.CreatedAt,
			UpdatedAt:   countdown.UpdatedAt,
		})
	}

	return countdownResponses, nil
}

// GetCountdownByID 根据ID获取倒数日详情
func (s *CountdownService) GetCountdownByID(ctx context.Context, countdownID uint, userID uint) (*response.CountdownResponse, error) {
	var countdown models.Countdown
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", countdownID, userID).First(&countdown).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("倒数日不存在或无权限访问")
		}
		return nil, err
	}

	return &response.CountdownResponse{
		ID:          countdown.ID,
		UserID:      countdown.UserID,
		Title:       countdown.Title,
		Description: countdown.Description,
		TargetDate:  countdown.TargetDate,
		CreatedAt:   countdown.CreatedAt,
		UpdatedAt:   countdown.UpdatedAt,
	}, nil
}

// UpdateCountdown 更新倒数日
func (s *CountdownService) UpdateCountdown(ctx context.Context, countdownID uint, userID uint, req *request.UpdateCountdownRequest) (*response.CountdownResponse, error) {
	// 查找倒数日
	var countdown models.Countdown
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", countdownID, userID).First(&countdown).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("倒数日不存在或无权限访问")
		}
		return nil, err
	}

	// 更新字段
	updates := make(map[string]interface{})

	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.TargetDate != nil && *req.TargetDate != "" {
		targetDate, err := time.Parse("2006-01-02", *req.TargetDate)
		if err != nil {
			return nil, errors.New("目标日期格式错误")
		}
		updates["target_date"] = targetDate
	}

	if len(updates) > 0 {
		if err := s.db.WithContext(ctx).Model(&countdown).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	return s.GetCountdownByID(ctx, countdownID, userID)
}

// DeleteCountdown 删除倒数日
func (s *CountdownService) DeleteCountdown(ctx context.Context, countdownID uint, userID uint) error {
	// 查找倒数日
	var countdown models.Countdown
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", countdownID, userID).First(&countdown).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("倒数日不存在或无权限访问")
		}
		return err
	}

	// 软删除
	return s.db.WithContext(ctx).Delete(&countdown).Error
}
