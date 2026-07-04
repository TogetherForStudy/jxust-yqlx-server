package worker

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"
)

// WorkerConfig holds configuration for a worker instance.
type WorkerConfig struct {
	// QueueKey is the Redis key for the task queue
	QueueKey string

	// ProcessInterval is how often to poll the queue
	ProcessInterval time.Duration

	// BatchSize is the maximum number of tasks to process per poll cycle.
	// 0 means unlimited (drain entire queue). >0 enables load leveling:
	// a burst of tasks is spread across multiple intervals, smoothing DB write pressure.
	BatchSize int

	// MaxRetries is the maximum number of retry attempts for failed tasks
	MaxRetries int

	// MaxConcurrentRetries limits the number of concurrent retry goroutines.
	// 0 (default) means 1 — the first failure doesn't block other tasks,
	// but a second concurrent failure will block the main loop, providing
	// automatic backpressure under systemic failure.
	MaxConcurrentRetries int

	// WorkerName is a human-readable identifier for logging
	WorkerName string
}

// Worker polls a queue and processes tasks using a TaskProcessor.
type Worker struct {
	config    WorkerConfig
	processor TaskProcessor
	queue     QueueProvider

	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	retryLimit chan struct{}
}

// NewWorker creates a new worker instance.
func NewWorker(config WorkerConfig, processor TaskProcessor, queue QueueProvider) *Worker {
	if config.MaxConcurrentRetries <= 0 {
		config.MaxConcurrentRetries = 1
	}
	return &Worker{
		config:     config,
		processor:  processor,
		queue:      queue,
		retryLimit: make(chan struct{}, config.MaxConcurrentRetries),
	}
}

// Start begins processing tasks from the queue.
// This method is non-blocking and returns immediately.
func (w *Worker) Start(ctx context.Context) {
	w.ctx, w.cancel = context.WithCancel(ctx)
	w.wg.Add(1)

	go w.run()

	logger.Info(fmt.Sprintf("Worker '%s' started", w.config.WorkerName))
}

// Stop gracefully stops the worker within the given timeout.
// Returns error if shutdown exceeds timeout.
func (w *Worker) Stop(timeout time.Duration) error {
	if w.cancel == nil {
		return nil
	}

	w.cancel()

	// Wait for worker to finish with timeout
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info(fmt.Sprintf("Worker '%s' stopped gracefully", w.config.WorkerName))
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("worker '%s' shutdown timeout exceeded", w.config.WorkerName)
	}
}

// run is the main worker loop that polls the queue at regular intervals.
func (w *Worker) run() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.config.ProcessInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			logger.Info(fmt.Sprintf("Worker '%s' received shutdown signal", w.config.WorkerName))
			return
		case <-ticker.C:
			w.processQueue()
		}
	}
}

// processQueue processes tasks from the queue, up to BatchSize per poll cycle.
// When BatchSize > 0, it acts as a rate limiter: a burst of incoming tasks is
// spread across multiple intervals instead of being written to DB all at once.
func (w *Worker) processQueue() {
	processed := 0
	for {
		// Check if context is cancelled before processing next task
		select {
		case <-w.ctx.Done():
			return
		default:
		}

		// Batch limit: stop after processing BatchSize tasks, leave the rest
		// for the next poll cycle (load leveling)
		if w.config.BatchSize > 0 && processed >= w.config.BatchSize {
			remaining, _ := w.queue.Length(w.ctx, w.config.QueueKey)
			logger.InfoCtx(w.ctx, map[string]any{
				"action":      "batch_limit_reached",
				"worker_name": w.config.WorkerName,
				"batch_size":  w.config.BatchSize,
				"remaining":   remaining,
			})
			break
		}

		// Pop a task from the queue
		taskData, err := w.queue.Pop(w.ctx, w.config.QueueKey)
		if err != nil || taskData == "" {
			// Queue is empty or error occurred
			break
		}

		// Unmarshal the task
		task, err := w.processor.Unmarshal([]byte(taskData))
		if err != nil {
			logger.ErrorCtx(w.ctx, map[string]any{
				"action":      "unmarshal_task_failed",
				"worker_name": w.config.WorkerName,
				"error":       err.Error(),
				"task_data":   taskData,
			})
			continue
		}

		// Process the task
		if err := w.processor.ProcessTask(w.ctx, task); err != nil {
			w.handleTaskError(task, taskData, err)
		} else {
			logger.InfoCtx(w.ctx, map[string]any{
				"action":      "task_processed_successfully",
				"worker_name": w.config.WorkerName,
				"task_type":   task.GetType(),
			})
		}
		processed++
	}
}

// handleTaskError handles task processing errors with retry logic.
// 使用指数退避策略，退避等待在异步 goroutine 中执行，避免阻塞主处理循环。
// 通过信号量限制并发重试 goroutine 数量（默认 1）：
//   - 第 1 个失败任务：主循环不阻塞，goroutine 异步等待后推回队列
//   - 若 goroutine 未完成时第 2 个任务也失败：信号量耗尽，主循环阻塞 → 自动背压
func (w *Worker) handleTaskError(task Task, taskData string, err error) {
	retryCount := task.GetRetryCount()

	if retryCount < w.config.MaxRetries {
		// 指数退避：1s → 2s → 4s（第1/2/3次重试）
		backoff := time.Duration(math.Pow(2, float64(retryCount))) * time.Second
		logger.WarnCtx(w.ctx, map[string]any{
			"action":       "task_retry_waiting",
			"worker_name":  w.config.WorkerName,
			"task_type":    task.GetType(),
			"retry_count":  retryCount + 1,
			"backoff_secs": backoff.Seconds(),
			"error":        err.Error(),
		})

		// Acquire retry slot; blocks main loop when all slots are occupied → automatic backpressure
		select {
		case w.retryLimit <- struct{}{}:
		case <-w.ctx.Done():
			return
		}

		// Execute backoff + re-enqueue in background, so the main loop can continue
		// processing other tasks (or block here if the slot was already occupied).
		go func() {
			defer func() { <-w.retryLimit }()

			select {
			case <-time.After(backoff):
			case <-w.ctx.Done():
				return
			}

			task.IncrementRetry()
			retryData, marshalErr := task.Marshal()
			if marshalErr != nil {
				logger.ErrorCtx(w.ctx, map[string]any{
					"action":      "marshal_retry_task_failed",
					"worker_name": w.config.WorkerName,
					"error":       marshalErr.Error(),
				})
				return
			}

			if pushErr := w.queue.Push(w.ctx, w.config.QueueKey, string(retryData)); pushErr != nil {
				logger.ErrorCtx(w.ctx, map[string]any{
					"action":      "push_retry_task_failed",
					"worker_name": w.config.WorkerName,
					"error":       pushErr.Error(),
				})
				return
			}

			logger.WarnCtx(w.ctx, map[string]any{
				"action":      "task_retry_pushed",
				"worker_name": w.config.WorkerName,
				"task_type":   task.GetType(),
				"retry_count": retryCount + 1,
				"error":       err.Error(),
			})
		}()
	} else {
		// Max retries exceeded, log and discard
		logger.ErrorCtx(w.ctx, map[string]any{
			"action":      "task_failed_final",
			"worker_name": w.config.WorkerName,
			"task_type":   task.GetType(),
			"retry_count": retryCount,
			"error":       err.Error(),
			"task_data":   taskData,
		})
	}
}
