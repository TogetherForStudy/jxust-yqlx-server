package bootstrap

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/config"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/cache"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/redis"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"

	"gorm.io/gorm"
)

// InitRedisCache 初始化Redis客户端并建立缓存连接。
// 当Redis不可用时会优雅降级，相关功能（幂等性等）将被禁用。
func InitRedisCache(cfg *config.Config) {
	if cfg.RedisHost == "" {
		logger.Warnf("Redis host not configured, idempotency feature will be disabled")
		return
	}

	redisCli := redis.NewRedisCli(cfg)
	if redisCli == nil {
		logger.Warnf("Failed to create Redis client, idempotency feature will be disabled")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisCli.GetRedisCli().Ping(ctx).Err(); err != nil {
		logger.Warnf("Redis connection failed: %v, idempotency feature will be disabled", err)
		return
	}

	cache.NewRedisCache(redisCli)
	logger.Infof("Redis cache initialized successfully, idempotency feature enabled")
}

// InitProjectRedisData 将项目相关的热数据从数据库同步到Redis中，
// 包括每个活跃项目的用户集合和刷题次数统计。
// 使用批量查询（2 次 DB 查询覆盖所有项目），避免 N+1 问题。
func InitProjectRedisData(db *gorm.DB) {
	if cache.GlobalCache == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	logger.Info("Initializing project Redis data from database...")

	// 1. 加载所有活跃项目 ID（带重试）
	var projects []models.QuestionProject
	const maxRetries = 3
	for i := 0; i < maxRetries; i++ {
		if err := db.Where("is_active = ?", true).Select("id").Find(&projects).Error; err != nil {
			if i < maxRetries-1 {
				wait := time.Duration(i+1) * time.Second
				logger.Warnf("Failed to load projects (attempt %d/%d): %v, retrying in %v...", i+1, maxRetries, err, wait)
				time.Sleep(wait)
				continue
			}
			logger.Warnf("Failed to load projects after %d attempts: %v", maxRetries, err)
			return
		}
		break
	}

	if len(projects) == 0 {
		logger.Info("No active projects found, skipping Redis data initialization")
		return
	}

	projectIDs := make([]uint, len(projects))
	for i, p := range projects {
		projectIDs[i] = p.ID
	}

	// 2. 批量查询所有活跃项目的用户集合（1 次 DB 查询替代 N 次）
	projectUserSets := syncProjectUserSets(ctx, db, projectIDs)

	// 3. 批量查询所有活跃项目的刷题次数（1 次 DB 查询替代 N 次）
	projectUsageCounts := syncProjectUsageCounts(ctx, db, projectIDs)

	// 4. 写入 Redis
	noExpiration := time.Duration(0)
	for _, projectID := range projectIDs {
		userSetKey := fmt.Sprintf("project:users:%d", projectID)
		usageKey := fmt.Sprintf("project:usage:%d", projectID)

		// 用户集合
		_ = cache.GlobalCache.Delete(ctx, userSetKey)
		if userIDs, ok := projectUserSets[projectID]; ok && len(userIDs) > 0 {
			members := make([]interface{}, len(userIDs))
			for i, uid := range userIDs {
				members[i] = strconv.FormatUint(uint64(uid), 10)
			}
			if _, err := cache.GlobalCache.SAdd(ctx, userSetKey, members...); err != nil {
				logger.Warnf("Failed to populate user set for project %d: %v", projectID, err)
			}
		}

		// 刷题次数
		usageCount := projectUsageCounts[projectID]
		if err := cache.GlobalCache.Set(ctx, usageKey, strconv.FormatInt(usageCount, 10), &noExpiration); err != nil {
			logger.Warnf("Failed to set usage count for project %d: %v", projectID, err)
		}
	}

	logger.Infof("Project Redis data initialized for %d projects", len(projects))
}

// syncProjectUserSets 批量查询所有活跃项目的用户集合。
// idx_user_project_usages_project_id 索引保证 WHERE project_id IN (...) 走索引。
func syncProjectUserSets(ctx context.Context, db *gorm.DB, projectIDs []uint) map[uint][]uint {
	result := make(map[uint][]uint, len(projectIDs))

	type row struct {
		ProjectID uint
		UserID    uint
	}
	var rows []row
	if err := db.Model(&models.UserProjectUsage{}).
		Select("project_id, user_id").
		Where("project_id IN ?", projectIDs).
		Find(&rows).Error; err != nil {
		logger.Warnf("Failed to load project user sets: %v", err)
		return result
	}

	for _, r := range rows {
		result[r.ProjectID] = append(result[r.ProjectID], r.UserID)
	}
	return result
}

// syncProjectUsageCounts 批量查询所有活跃项目的刷题总次数。
// 一次 JOIN + GROUP BY 替代原来的 N 次独立查询。
// idx_user_question_usages_question_id 索引保证 JOIN 走索引。
func syncProjectUsageCounts(ctx context.Context, db *gorm.DB, projectIDs []uint) map[uint]int64 {
	result := make(map[uint]int64, len(projectIDs))
	// 初始化所有项目为 0
	for _, id := range projectIDs {
		result[id] = 0
	}

	type row struct {
		ProjectID uint
		Total     int64
	}
	var rows []row
	if err := db.Table("questions q").
		Select("q.project_id, COALESCE(SUM(uq.study_count + uq.practice_count), 0) AS total").
		Joins("LEFT JOIN user_question_usages uq ON uq.question_id = q.id").
		Where("q.project_id IN ? AND q.is_active = ?", projectIDs, true).
		Group("q.project_id").
		Scan(&rows).Error; err != nil {
		logger.Warnf("Failed to load project usage counts: %v", err)
		return result
	}

	for _, r := range rows {
		result[r.ProjectID] = r.Total
	}
	return result
}
