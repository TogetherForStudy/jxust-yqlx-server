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

// InitProjectRedisData 将项目相关的热数据预热到Redis中，
// 包括每个活跃项目的用户集合和刷题次数统计。
func InitProjectRedisData(db *gorm.DB) {
	if cache.GlobalCache == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger.Info("Initializing project Redis data...")

	var projects []models.QuestionProject
	if err := db.Where("is_active = ?", true).Select("id").Find(&projects).Error; err != nil {
		logger.Warnf("Failed to load projects: %v", err)
		return
	}

	for _, project := range projects {
		projectID := project.ID
		userSetKey := fmt.Sprintf("project:users:%d", projectID)
		usageKey := fmt.Sprintf("project:usage:%d", projectID)

		initProjectUserSet(ctx, db, projectID, userSetKey)
		initProjectUsageCount(ctx, db, projectID, usageKey)
	}

	logger.Info("Project Redis data initialization completed")
}

func initProjectUserSet(ctx context.Context, db *gorm.DB, projectID uint, key string) {
	var userIDs []uint
	if err := db.Model(&models.UserProjectUsage{}).
		Where("project_id = ?", projectID).
		Pluck("user_id", &userIDs).Error; err != nil {
		logger.Warnf("Failed to load users for project %d: %v", projectID, err)
		return
	}

	if len(userIDs) == 0 {
		return
	}

	_ = cache.GlobalCache.Delete(ctx, key)
	members := make([]interface{}, len(userIDs))
	for i, id := range userIDs {
		members[i] = strconv.FormatUint(uint64(id), 10)
	}
	if _, err := cache.GlobalCache.SAdd(ctx, key, members...); err != nil {
		logger.Warnf("Failed to initialize user set for project %d: %v", projectID, err)
	}
}

func initProjectUsageCount(ctx context.Context, db *gorm.DB, projectID uint, key string) {
	var questionIDs []uint
	if err := db.Model(&models.Question{}).
		Where("project_id = ? AND is_active = ?", projectID, true).
		Pluck("id", &questionIDs).Error; err != nil {
		logger.Warnf("Failed to load questions for project %d: %v", projectID, err)
		return
	}

	if len(questionIDs) == 0 {
		return
	}

	var usageCount int64
	if err := db.Model(&models.UserQuestionUsage{}).
		Where("question_id IN ?", questionIDs).
		Select("COALESCE(SUM(study_count + practice_count), 0)").
		Scan(&usageCount).Error; err != nil {
		logger.Warnf("Failed to calculate usage count for project %d: %v", projectID, err)
		return
	}

	noExpiration := time.Duration(0)
	if err := cache.GlobalCache.Set(ctx, key, strconv.FormatInt(usageCount, 10), &noExpiration); err != nil {
		logger.Warnf("Failed to initialize usage count for project %d: %v", projectID, err)
	}
}
