package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/cache"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// Scheduler 定时任务调度器
type Scheduler struct {
	cron                *cron.Cron
	db                  *gorm.DB
	materialService     *services.MaterialService
	userActivityService *services.UserActivityService
}

// NewScheduler 创建新的调度器实例
func NewScheduler(db *gorm.DB) *Scheduler {
	// 使用中国时区
	// 使用内置日志
	c := cron.New()

	rbacService := services.NewRBACService(db)
	userActivityService := services.NewUserActivityService(db, rbacService)

	return &Scheduler{
		cron:                c,
		db:                  db,
		materialService:     services.NewMaterialService(db),
		userActivityService: userActivityService,
	}
}

// Start 启动定时任务调度器
func (s *Scheduler) Start() error {
	// 添加每天凌晨2点执行热度计算的任务
	// cron表达式: "0 2 * * *" 表示每天凌晨2点0分执行
	_, err := s.cron.AddFunc("0 2 * * *", func() {
		ctx := context.Background()
		logger.DebugCtx(ctx, map[string]any{
			"task":   "material_hotness_calculation",
			"status": "started",
		})
		if err := s.materialService.CalculateHotness(ctx); err != nil {
			logger.ErrorCtx(ctx, map[string]any{
				"task":   "material_hotness_calculation",
				"status": "failed",
				"error":  err.Error(),
			})
		} else {
			logger.DebugCtx(ctx, map[string]any{
				"task":   "material_hotness_calculation",
				"status": "success",
			})
		}
	})

	if err != nil {
		return err
	}

	// 添加每天凌晨3点执行活跃用户角色更新任务
	// cron表达式: "0 3 * * *" 表示每天凌晨3点0分执行
	_, err = s.cron.AddFunc("0 3 * * *", func() {
		ctx := context.Background()
		logger.DebugCtx(ctx, map[string]any{
			"task":   "active_user_role_update",
			"status": "started",
		})
		if err := s.userActivityService.UpdateActiveUserRoles(ctx); err != nil {
			logger.ErrorCtx(ctx, map[string]any{
				"task":   "active_user_role_update",
				"status": "failed",
				"error":  err.Error(),
			})
		} else {
			logger.DebugCtx(ctx, map[string]any{
				"task":   "active_user_role_update",
				"status": "success",
			})
		}
	})

	if err != nil {
		return err
	}

	// 添加每小时执行在线用户数据清理任务
	// cron表达式: "0 * * * *" 表示每小时0分执行
	_, err = s.cron.AddFunc("0 * * * *", func() {
		ctx := context.Background()
		logger.Info("任务开始：在线用户数据清理")
		if err := s.cleanupOnlineUserData(ctx); err != nil {
			logger.Error("任务失败：在线用户数据清理")
		} else {
			logger.Info("任务成功：在线用户数据清理")
		}
	})

	if err != nil {
		return err
	}

	logger.Info("定时任务调度器已启动，热度计算任务将在每天凌晨2点执行，活跃用户角色更新任务将在每天凌晨3点执行，在线用户数据清理任务将每小时执行")
	s.cron.Start()
	return nil
}

// cleanupOnlineUserData 清理过期的在线用户数据
// 清理1小时前（超过1分钟过期时间很久）的过期数据，保持Redis数据整洁
func (s *Scheduler) cleanupOnlineUserData(ctx context.Context) error {
	if cache.GlobalCache == nil {
		return nil
	}

	now := float64(time.Now().Unix())
	// 清理1小时前的过期数据（1分钟过期时间 + 1小时缓冲）
	expiredTime := now - 3600 // 1小时前

	// 1. 清理系统在线用户数据
	systemOnlineKey := "online:system"
	if _, err := cache.GlobalCache.ZRemRangeByScore(ctx, systemOnlineKey, 0, expiredTime); err != nil {
		logger.Errorf("清理系统在线用户数据失败: %v", err)
	}

	// 2. 清理所有项目的在线用户数据
	// 查询所有激活的项目ID
	var projectIDs []uint
	if err := s.db.Table("question_projects").Where("deleted_at IS NULL").Pluck("id", &projectIDs).Error; err != nil {
		logger.Errorf("获取项目ID列表失败: %v", err)
		return err
	}

	for _, pid := range projectIDs {
		projectOnlineKey := fmt.Sprintf("online:project:%d", pid)
		if _, err := cache.GlobalCache.ZRemRangeByScore(ctx, projectOnlineKey, 0, expiredTime); err != nil {
			logger.Errorf("清理项目[%d]在线用户数据失败: %v", pid, err)
		}
	}

	return nil
}

// Stop 停止定时任务调度器
func (s *Scheduler) Stop() {
	logger.Info("正在停止定时任务调度器...")
	s.cron.Stop()
	logger.Info("定时任务调度器已停止")
}

// GetScheduler 获取cron实例（用于添加其他定时任务）
func (s *Scheduler) GetScheduler() *cron.Cron {
	return s.cron
}
