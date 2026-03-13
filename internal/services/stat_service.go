package services

import (
	"context"
	"fmt"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/apperr"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/cache"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/utils"
	"gorm.io/gorm"
)

type StatService struct {
	db *gorm.DB
}

func NewStatService(db *gorm.DB) *StatService {
	return &StatService{db: db}
}

// GetSystemOnlineCount 获取系统在线人数
// 使用 Sorted Set，只统计最近1分钟内有活动的用户
// 不主动清理过期数据，只统计有效范围内的用户，过期数据不影响统计结果
func (s *StatService) GetSystemOnlineCount(ctx context.Context) (int64, error) {
	if cache.GlobalCache == nil {
		return 0, apperr.New(constant.StatServiceUnavailable)
	}

	key := "online:system"
	now := float64(time.Now().Unix())
	// 只统计最近1分钟内的用户（score >= now - 60）
	minScore := now - 60
	maxScore := now + 1 // 包含当前时间

	// 统计最近1分钟内的用户数量（不清理过期数据，只统计有效范围）
	count, err := cache.GlobalCache.ZCount(ctx, key, minScore, maxScore)
	if err != nil {
		return 0, apperr.Wrap(constant.StatServiceUnavailable, err)
	}

	return count, nil
}

// GetProjectOnlineCount 获取项目在线人数
// 使用 Sorted Set，只统计最近1分钟内有活动的用户
// 不主动清理过期数据，只统计有效范围内的用户，过期数据不影响统计结果
// 如果传入了userID（>0），则在统计前先更新该用户的在线状态
func (s *StatService) GetProjectOnlineCount(ctx context.Context, projectID uint, userID ...uint) (int64, error) {
	if cache.GlobalCache == nil {
		return 0, apperr.New(constant.StatServiceUnavailable)
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
		return 0, apperr.Wrap(constant.StatServiceUnavailable, err)
	}

	return count, nil
}

// GetAllProjectsOnlineCount 获取所有启用项目的在线人数
func (s *StatService) GetAllProjectsOnlineCount(ctx context.Context) ([]response.AllProjectsOnlineStatItem, error) {
	if cache.GlobalCache == nil {
		return nil, apperr.New(constant.StatServiceUnavailable)
	}

	// 查询所有启用的项目
	var projects []struct {
		ID   uint
		Name string
	}
	if err := s.db.WithContext(ctx).Table("question_projects").
		Select("id, name").
		Where("is_active = ? AND deleted_at IS NULL", true).
		Order("sort ASC, id ASC").
		Scan(&projects).Error; err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询启用项目列表失败: %w", err))
	}

	now := float64(time.Now().Unix())
	minScore := now - 60
	maxScore := now + 1

	items := make([]response.AllProjectsOnlineStatItem, 0, len(projects))
	for _, p := range projects {
		key := fmt.Sprintf("online:project:%d", p.ID)
		count, err := cache.GlobalCache.ZCount(ctx, key, minScore, maxScore)
		if err != nil {
			count = 0 // Redis 查询失败时降级为0
		}
		items = append(items, response.AllProjectsOnlineStatItem{
			ProjectID:   p.ID,
			ProjectName: p.Name,
			OnlineCount: count,
		})
	}

	return items, nil
}

func (s *StatService) GetCountdownCountsByUser(ctx context.Context, page, pageSize int) ([]response.AdminUserCountStatResponse, int64, error) {
	return s.getUserCountStats(ctx, "countdowns", page, pageSize)
}

func (s *StatService) GetStudyTaskCountsByUser(ctx context.Context, page, pageSize int) ([]response.AdminUserCountStatResponse, int64, error) {
	return s.getUserCountStats(ctx, "study_tasks", page, pageSize)
}

func (s *StatService) GetGPABackupCountsByUser(ctx context.Context, page, pageSize int) ([]response.AdminUserCountStatResponse, int64, error) {
	return s.getUserCountStats(ctx, "gpa_backups", page, pageSize)
}

func (s *StatService) getUserCountStats(ctx context.Context, tableName string, page, pageSize int) ([]response.AdminUserCountStatResponse, int64, error) {
	if s.db == nil {
		return nil, 0, apperr.New(constant.StatServiceUnavailable)
	}

	var total int64
	groupedQuery := s.db.WithContext(ctx).Table(tableName).Select("user_id").Group("user_id")
	if err := s.db.WithContext(ctx).Table("(?) AS grouped", groupedQuery).Count(&total).Error; err != nil {
		return nil, 0, apperr.Wrap(constant.CommonInternal, fmt.Errorf("统计 %s 用户分组总数失败: %w", tableName, err))
	}

	pagination := utils.GetPagination(page, pageSize)
	var result []response.AdminUserCountStatResponse
	if err := s.db.WithContext(ctx).Table(tableName).
		Select("user_id, COUNT(*) AS count").
		Group("user_id").
		Order("count DESC, user_id ASC").
		Offset(pagination.Offset).
		Limit(pagination.Size).
		Scan(&result).Error; err != nil {
		return nil, 0, apperr.Wrap(constant.CommonInternal, fmt.Errorf("统计 %s 用户分组失败: %w", tableName, err))
	}

	return result, total, nil
}
