package scheduler

import (
	"log"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// Scheduler 定时任务调度器
type Scheduler struct {
	cron            *cron.Cron
	materialService *services.MaterialService
}

// NewScheduler 创建新的调度器实例
func NewScheduler(db *gorm.DB) *Scheduler {
	// 使用中国时区
	c := cron.New(cron.WithLogger(cron.VerbosePrintfLogger(log.New(log.Writer(), "cron: ", log.LstdFlags))))

	return &Scheduler{
		cron:            c,
		materialService: services.NewMaterialService(db),
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

	log.Println("定时任务调度器已启动，热度计算任务将在每天凌晨2点执行")
	s.cron.Start()
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
