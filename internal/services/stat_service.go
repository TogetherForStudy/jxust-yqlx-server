package services

import (
	"context"
	"fmt"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/cache"
)

type StatService struct{}

func NewStatService() *StatService {
	return &StatService{}
}

// GetSystemOnlineCount 获取系统在线人数
// 使用 Sorted Set，只统计最近1分钟内有活动的用户
// 不主动清理过期数据，只统计有效范围内的用户，过期数据不影响统计结果
func (s *StatService) GetSystemOnlineCount(ctx context.Context) (int64, error) {
	if cache.GlobalCache == nil {
		return 0, fmt.Errorf("Redis缓存未初始化")
	}

	key := "online:system"
	now := float64(time.Now().Unix())
	// 只统计最近1分钟内的用户（score >= now - 60）
	minScore := now - 60
	maxScore := now + 1 // 包含当前时间

	// 统计最近1分钟内的用户数量（不清理过期数据，只统计有效范围）
	count, err := cache.GlobalCache.ZCount(ctx, key, minScore, maxScore)
	if err != nil {
		return 0, fmt.Errorf("获取系统在线人数失败: %w", err)
	}

	return count, nil
}

// GetProjectOnlineCount 获取项目在线人数
// 使用 Sorted Set，只统计最近1分钟内有活动的用户
// 不主动清理过期数据，只统计有效范围内的用户，过期数据不影响统计结果
// 如果传入了userID（>0），则在统计前先更新该用户的在线状态
func (s *StatService) GetProjectOnlineCount(ctx context.Context, projectID uint, userID ...uint) (int64, error) {
	if cache.GlobalCache == nil {
		return 0, fmt.Errorf("Redis缓存未初始化")
	}

	key := fmt.Sprintf("online:project:%d", projectID)
	now := float64(time.Now().Unix())

	// 如果传入了userID，先更新该用户的在线状态
	if len(userID) > 0 && userID[0] > 0 {
		userIDStr := fmt.Sprintf("%d", userID[0])
		_ = cache.GlobalCache.ZAdd(ctx, key, now, userIDStr)
	}

	// 只统计最近1分钟内的用户（score >= now - 60）
	minScore := now - 60
	maxScore := now + 1 // 包含当前时间

	// 统计最近1分钟内的用户数量（不清理过期数据，只统计有效范围）
	count, err := cache.GlobalCache.ZCount(ctx, key, minScore, maxScore)
	if err != nil {
		return 0, fmt.Errorf("获取项目在线人数失败: %w", err)
	}

	return count, nil
}
