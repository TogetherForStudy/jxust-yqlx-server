package services

import (
	"context"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/apperr"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"

	"gorm.io/gorm"
)

type PomodoroService struct {
	db *gorm.DB
}

func NewPomodoroService(db *gorm.DB) *PomodoroService {
	return &PomodoroService{
		db: db,
	}
}

// IncrementPomodoroCount 增加番茄钟次数
func (s *PomodoroService) IncrementPomodoroCount(ctx context.Context, userID uint) error {
	if err := s.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).
		UpdateColumn("pomodoro_count", gorm.Expr("pomodoro_count + ?", 1)).Error; err != nil {
		return apperr.Wrap(constant.CommonInternal, err)
	}
	return nil
}

// GetPomodoroCount 获取用户番茄钟次数
func (s *PomodoroService) GetPomodoroCount(ctx context.Context, userID uint) (uint, error) {
	var user models.User
	if err := s.db.WithContext(ctx).
		Select("id", "pomodoro_count").
		Where("id = ?", userID).
		First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, apperr.FromCode(constant.CommonUserNotFound)
		}
		return 0, apperr.Wrap(constant.CommonInternal, err)
	}

	return user.PomodoroCount, nil
}

// GetPomodoroRanking 获取番茄钟排名（前20名）
func (s *PomodoroService) GetPomodoroRanking(ctx context.Context) ([]response.PomodoroRankingItem, error) {
	var results []response.PomodoroRankingItem
	err := s.db.WithContext(ctx).Model(&models.User{}).
		Select("id AS rank, nickname, pomodoro_count").
		Where("pomodoro_count > 0").
		Order("pomodoro_count DESC").
		Limit(20).
		Scan(&results).Error

	if err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, err)
	}

	return results, nil
}
