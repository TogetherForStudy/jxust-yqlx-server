package bootstrap

import (
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/cache"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/worker"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/worker/processors"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"

	"gorm.io/gorm"
)

// InitializeWorkers 创建WorkerManager并注册所有异步任务Worker。
// 需要Redis可用才会注册实际的Worker，否则返回空的Manager。
func InitializeWorkers(db *gorm.DB) *worker.WorkerManager {
	manager := worker.NewWorkerManager()

	if cache.GlobalCache == nil {
		logger.Warn("Redis not available, workers will not be started")
		return manager
	}

	queueProvider := worker.NewRedisQueueProvider(cache.GlobalCache)
	questionProcessor := processors.NewQuestionTaskProcessor(db)

	cfg := worker.WorkerConfig{
		QueueKey:        "sync:question:usage",
		ProcessInterval: 5 * time.Second,
		MaxRetries:      3,
		WorkerName:      "question-sync-worker",
	}

	questionWorker := worker.NewWorker(cfg, questionProcessor, queueProvider)
	if err := manager.RegisterWorker("question-sync", questionWorker); err != nil {
		logger.Warnf("Failed to register question sync worker: %v", err)
	} else {
		logger.Info("Question sync worker registered")
	}

	return manager
}
