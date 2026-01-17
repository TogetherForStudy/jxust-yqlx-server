// Package worker provides a generic framework for asynchronous task processing.
//
// The worker package decouples business logic from async processing infrastructure,
// allowing services to push tasks to queues without managing worker lifecycle.
//
// Key Components:
//   - Task: Interface for task data that can be serialized and processed
//   - TaskProcessor: Interface for task-specific processing logic
//   - QueueProvider: Interface for queue backends (Redis, etc.)
//   - Worker: Core worker that polls queues and processes tasks
//   - WorkerManager: Manages multiple workers' lifecycle
//
// Example Usage:
//
//	// Create queue provider
//	queueProvider := worker.NewRedisQueueProvider(cache.GlobalCache)
//
//	// Create task processor
//	processor := processors.NewQuestionTaskProcessor(db)
//
//	// Configure worker
//	config := worker.WorkerConfig{
//	    QueueKey:        "sync:question:usage",
//	    ProcessInterval: 5 * time.Second,
//	    MaxRetries:      3,
//	    WorkerName:      "question-sync-worker",
//	}
//
//	// Create and start worker
//	w := worker.NewWorker(config, processor, queueProvider)
//	ctx := context.Background()
//	w.Start(ctx)
//
//	// Stop worker gracefully
//	w.Stop(10 * time.Second)
package worker
