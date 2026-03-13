package bootstrap

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/config"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/database"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/router"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/scheduler"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/worker"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// App 封装了应用的全部基础设施组件及其生命周期。
type App struct {
	cfg           *config.Config
	db            *gorm.DB
	scheduler     *scheduler.Scheduler
	workerManager *worker.WorkerManager
	workerCancel  context.CancelFunc
}

// New 按依赖顺序初始化所有组件并返回 App 实例。
// 任何关键组件初始化失败都会通过 logger.Fatalf 终止进程。
func New() *App {
	cfg := config.NewConfig()

	db, err := database.NewDatabase(cfg)
	if err != nil {
		logger.Fatalf("Failed to initialize database: %v", err)
	}

	if err := logger.TencentClsLoggerInit(
		context.Background(),
		cfg.ClsEnable, cfg.ClsEndpoint, cfg.ClsTopicID,
		cfg.ClsSecretID, cfg.ClsSecretKey,
	); err != nil {
		logger.Fatalf("Failed to initialize Tencent CLS logger: %v", err)
	}

	if err := database.AutoMigrate(db); err != nil {
		logger.Fatalf("Failed to migrate database: %v", err)
	}

	InitRedisCache(cfg)
	InitProjectRedisData(db)

	taskScheduler := scheduler.NewScheduler(db)
	if err := taskScheduler.Start(); err != nil {
		logger.Fatalf("Failed to start scheduler: %v", err)
	}

	workerManager := InitializeWorkers(db)
	workerCtx, workerCancel := context.WithCancel(context.Background())
	if err := workerManager.StartAll(workerCtx); err != nil {
		logger.Fatalf("Failed to start workers: %v", err)
	}

	return &App{
		cfg:           cfg,
		db:            db,
		scheduler:     taskScheduler,
		workerManager: workerManager,
		workerCancel:  workerCancel,
	}
}

// Run 启动HTTP服务器并阻塞等待关闭信号，收到信号后执行优雅关闭。
func (a *App) Run() {
	if a.cfg.GinMode == gin.ReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	}

	r := router.NewRouter(a.db, a.cfg)

	port := a.cfg.ServerPort
	if port == "" {
		port = "8085"
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Infof("Server starting on port %s", port)
		if err := r.Run(":" + port); err != nil {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	<-quit
	logger.Info("Server is shutting down...")
	a.Shutdown()
}

// Shutdown 按依赖逆序优雅关闭所有组件。
func (a *App) Shutdown() {
	logger.Info("Flushing remaining logs...")
	logger.ShutdownLogger(10 * time.Second)

	a.scheduler.Stop()

	logger.Info("Stopping workers...")
	a.workerCancel()
	if err := a.workerManager.StopAll(10 * time.Second); err != nil {
		logger.Warnf("Worker shutdown error: %v", err)
	}

	logger.Info("Server shutdown complete")
}
