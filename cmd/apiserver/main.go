package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/config"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/database"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/cache"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/redis"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/router"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/scheduler"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/worker"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/worker/processors"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

func main() {
	// 加载环境变量
	if err := godotenv.Load(); err != nil {
		logger.Warnf(".env file not found: %v", err)
	}

	// 初始化配置
	cfg := config.NewConfig()

	// 初始化数据库
	db, err := database.NewDatabase(cfg)
	if err != nil {
		logger.Fatalf("Failed to initialize database: %v", err)
	}
	err = logger.TencentClsLoggerInit(context.Background(), cfg.ClsEnable, cfg.ClsEndpoint, cfg.ClsTopicID, cfg.ClsSecretID, cfg.ClsSecretKey)
	if err != nil {
		logger.Fatalf("Failed to initialize Tencent CLS logger: %v", err)
	}

	// 自动迁移数据库表
	if err := database.AutoMigrate(db); err != nil {
		logger.Fatalf("Failed to migrate database: %v", err)
	}

	// 初始化Redis和缓存
	initRedisCache(cfg)
	// 初始化项目相关的Redis数据
	if cache.GlobalCache != nil {
		initProjectRedisData(db)
	}
	// 初始化并启动定时任务调度器
	taskScheduler := scheduler.NewScheduler(db)
	if err := taskScheduler.Start(); err != nil {
		logger.Fatalf("Failed to start scheduler: %v", err)
	}

	// 初始化并启动异步工作器
	workerManager := initializeWorkers(db)
	workerCtx, workerCancel := context.WithCancel(context.Background())
	if err := workerManager.StartAll(workerCtx); err != nil {
		logger.Fatalf("Failed to start workers: %v", err)
	}

	// 设置生产模式
	if os.Getenv(constant.ENV_GIN_MODE) == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 初始化路由
	r := router.NewRouter(db, cfg)

	// 启动服务器
	port := cfg.ServerPort
	if port == "" {
		port = "8085"
	}

	// 设置优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 在goroutine中启动gin服务器
	go func() {
		logger.Infof("Server starting on port %s", port)
		if err := r.Run(":" + port); err != nil {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 等待关闭信号
	<-quit
	logger.Info("Server is shutting down...")

	// 优雅关闭日志系统（确保所有日志上报完成）
	logger.Info("Flushing remaining logs...")
	logger.ShutdownLogger(10 * time.Second)

	// 停止定时任务调度器
	taskScheduler.Stop()

	// 停止异步工作器
	logger.Info("Stopping workers...")
	workerCancel()
	if err := workerManager.StopAll(10 * time.Second); err != nil {
		logger.Warnf("Worker shutdown error: %v", err)
	}

	logger.Info("Server shutdown complete")
}

// initRedisCache 初始化Redis缓存
func initRedisCache(cfg *config.Config) {
	// 检查Redis配置是否有效
	if cfg.RedisHost == "" {
		logger.Warnf("Redis host not configured, idempotency feature will be disabled")
		return
	}

	// 初始化Redis客户端
	redisCli := redis.NewRedisCli(cfg)
	if redisCli == nil {
		logger.Warnf("Failed to create Redis client, idempotency feature will be disabled")
		return
	}

	// 测试Redis连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisCli.GetRedisCli().Ping(ctx).Err(); err != nil {
		logger.Warnf("Redis connection failed: %v, idempotency feature will be disabled", err)
		return
	}

	// 初始化缓存
	cache.NewRedisCache(redisCli)
	logger.Infof("Redis cache initialized successfully, idempotency feature enabled")
}

// initProjectRedisData 初始化项目相关的Redis数据
func initProjectRedisData(db *gorm.DB) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if cache.GlobalCache == nil {
		return
	}

	logger.Info("Initializing project Redis data...")

	// 获取所有项目
	var projects []models.QuestionProject
	if err := db.Where("is_active = ?", true).Select("id").Find(&projects).Error; err != nil {
		logger.Warnf("Failed to load projects: %v", err)
		return
	}

	// 初始化每个项目的用户集合和刷题次数
	for _, project := range projects {
		projectID := project.ID
		userSetKey := fmt.Sprintf("project:users:%d", projectID)
		usageKey := fmt.Sprintf("project:usage:%d", projectID)

		// 加载用户集合
		var userIDs []uint
		if err := db.Model(&models.UserProjectUsage{}).
			Where("project_id = ?", projectID).
			Pluck("user_id", &userIDs).Error; err == nil {
			if len(userIDs) > 0 {
				// 清空现有集合（如果存在）
				_ = cache.GlobalCache.Delete(ctx, userSetKey)
				// 添加所有用户ID到集合
				members := make([]interface{}, len(userIDs))
				for i, id := range userIDs {
					members[i] = strconv.FormatUint(uint64(id), 10)
				}
				if _, err := cache.GlobalCache.SAdd(ctx, userSetKey, members...); err != nil {
					logger.Warnf("Failed to initialize user set for project %d: %v", projectID, err)
				}
			}
		} else {
			logger.Warnf("Failed to load users for project %d: %v", projectID, err)
		}

		// 加载刷题次数
		var questionIDs []uint
		if err := db.Model(&models.Question{}).
			Where("project_id = ? AND is_active = ?", projectID, true).
			Pluck("id", &questionIDs).Error; err == nil {
			if len(questionIDs) > 0 {
				var usageCount int64
				if err := db.Model(&models.UserQuestionUsage{}).
					Where("question_id IN ?", questionIDs).
					Select("COALESCE(SUM(study_count + practice_count), 0)").
					Scan(&usageCount).Error; err == nil {
					// 设置Redis中的刷题次数
					noExpiration := time.Duration(0)
					if err := cache.GlobalCache.Set(ctx, usageKey, strconv.FormatInt(usageCount, 10), &noExpiration); err != nil {
						logger.Warnf("Failed to initialize usage count for project %d: %v", projectID, err)
					}
				} else {
					logger.Warnf("Failed to calculate usage count for project %d: %v", projectID, err)
				}
			}
		} else {
			logger.Warnf("Failed to load questions for project %d: %v", projectID, err)
		}
	}

	logger.Info("Project Redis data initialization completed")
}

// initializeWorkers initializes and registers all async workers
func initializeWorkers(db *gorm.DB) *worker.WorkerManager {
	manager := worker.NewWorkerManager()

	// Only initialize workers if Redis is available
	if cache.GlobalCache != nil {
		// Create queue provider
		queueProvider := worker.NewRedisQueueProvider(cache.GlobalCache)

		// Create question task processor
		questionProcessor := processors.NewQuestionTaskProcessor(db)

		// Configure question sync worker
		config := worker.WorkerConfig{
			QueueKey:        "sync:question:usage",
			ProcessInterval: 5 * time.Second,
			MaxRetries:      3,
			WorkerName:      "question-sync-worker",
		}

		// Create and register worker
		questionWorker := worker.NewWorker(config, questionProcessor, queueProvider)
		if err := manager.RegisterWorker("question-sync", questionWorker); err != nil {
			logger.Warnf("Failed to register question sync worker: %v", err)
		} else {
			logger.Info("Question sync worker registered")
		}
	} else {
		logger.Warn("Redis not available, workers will not be started")
	}

	return manager
}
