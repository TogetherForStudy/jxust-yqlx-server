package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/cache"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// Scheduler 定时任务调度器
type Scheduler struct {
	cron                *cron.Cron
	materialService     *services.MaterialService
	userActivityService *services.UserActivityService
}

// NewScheduler 创建新的调度器实例
func NewScheduler(db *gorm.DB) *Scheduler {
	// 使用中国时区
	c := cron.New(cron.WithLogger(cron.VerbosePrintfLogger(log.New(log.Writer(), "cron: ", log.LstdFlags))))

	rbacService := services.NewRBACService(db)
	userActivityService := services.NewUserActivityService(db, rbacService)

	return &Scheduler{
		cron:                c,
		materialService:     services.NewMaterialService(db),
		userActivityService: userActivityService,
	}
}

// Start 启动定时任务调度器
func (s *Scheduler) Start() error {
	// 添加每天凌晨2点执行热度计算的任务
	// cron表达式: "0 2 * * *" 表示每天凌晨2点0分执行
	_, err := s.cron.AddFunc("0 2 * * *", func() {
		log.Println("开始执行热度计算定时任务...")
		if err := s.materialService.CalculateHotness(); err != nil {
			log.Printf("热度计算任务执行失败: %v", err)
		} else {
			log.Println("热度计算任务执行成功")
		}
	})

	if err != nil {
		return err
	}

	// 添加每天凌晨3点执行活跃用户角色更新任务
	// cron表达式: "0 3 * * *" 表示每天凌晨3点0分执行
	_, err = s.cron.AddFunc("0 3 * * *", func() {
		log.Println("开始执行活跃用户角色更新任务...")
		ctx := context.Background()
		if err := s.userActivityService.UpdateActiveUserRoles(ctx); err != nil {
			log.Printf("活跃用户角色更新任务执行失败: %v", err)
		} else {
			log.Println("活跃用户角色更新任务执行成功")
		}
	})

	if err != nil {
		return err
	}

	// 添加每小时执行在线用户数据清理任务
	// cron表达式: "0 * * * *" 表示每小时0分执行
	_, err = s.cron.AddFunc("0 * * * *", func() {
		log.Println("开始执行在线用户数据清理任务...")
		ctx := context.Background()
		if err := cleanupOnlineUserData(ctx); err != nil {
			log.Printf("在线用户数据清理任务执行失败: %v", err)
		} else {
			log.Println("在线用户数据清理任务执行成功")
		}
	})

	if err != nil {
		return err
	}

	log.Println("定时任务调度器已启动，热度计算任务将在每天凌晨2点执行，活跃用户角色更新任务将在每天凌晨3点执行，在线用户数据清理任务将每小时执行")
	s.cron.Start()
	return nil
}

// cleanupOnlineUserData 清理过期的在线用户数据
// 清理1小时前（超过1分钟过期时间很久）的过期数据，保持Redis数据整洁
func cleanupOnlineUserData(ctx context.Context) error {
	if cache.GlobalCache == nil {
		return nil
	}

	now := float64(time.Now().Unix())
	// 清理1小时前的过期数据（1分钟过期时间 + 1小时缓冲）
	expiredTime := now - 3600 // 1小时前

	// 清理系统在线用户数据
	systemOnlineKey := "online:system"
	if _, err := cache.GlobalCache.ZRemRangeByScore(ctx, systemOnlineKey, 0, expiredTime); err != nil {
		log.Printf("清理系统在线用户数据失败: %v", err)
	}

	// 注意：项目在线用户数据的key是动态的（online:project:{project_id}）
	// 由于无法预先知道所有项目ID，这里只清理系统级别的数据
	// 项目级别的数据会在查询时自动过滤，不会影响统计结果

	return nil
}

// Stop 停止定时任务调度器
func (s *Scheduler) Stop() {
	log.Println("正在停止定时任务调度器...")
	s.cron.Stop()
	log.Println("定时任务调度器已停止")
}

// GetScheduler 获取cron实例（用于添加其他定时任务）
func (s *Scheduler) GetScheduler() *cron.Cron {
	return s.cron
}
