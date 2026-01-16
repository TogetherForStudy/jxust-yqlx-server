package services

import (
	"context"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"

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
	return s.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).
		UpdateColumn("pomodoro_count", gorm.Expr("pomodoro_count + ?", 1)).Error
}

// GetPomodoroRanking 获取番茄钟排名（前20名）
func (s *PomodoroService) GetPomodoroRanking(ctx context.Context) ([]response.PomodoroRankingItem, error) {
	var results []response.PomodoroRankingItem
	err := s.db.WithContext(ctx).Model(&models.User{}).
		Select("nickname, pomodoro_count").
		Where("pomodoro_count > 0").
		Order("pomodoro_count DESC").
		Limit(20).
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	// 为每个结果计算排名
	for i := range results {
		results[i].Rank = i + 1
	}

	return results, nil
}
