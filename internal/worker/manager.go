package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"
)

// WorkerManager manages the lifecycle of multiple workers.
type WorkerManager struct {
	workers map[string]*Worker
	mu      sync.RWMutex
}

// NewWorkerManager creates a new worker manager.
func NewWorkerManager() *WorkerManager {
	return &WorkerManager{
		workers: make(map[string]*Worker),
	}
}

// RegisterWorker adds a worker to the manager.
// The name must be unique.
func (m *WorkerManager) RegisterWorker(name string, worker *Worker) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.workers[name]; exists {
		return fmt.Errorf("worker '%s' already registered", name)
	}

	m.workers[name] = worker
	return nil
}

// StartAll starts all registered workers.
func (m *WorkerManager) StartAll(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, worker := range m.workers {
		worker.Start(ctx)
		logger.Info(fmt.Sprintf("Started worker: %s", name))
	}

	logger.Info(fmt.Sprintf("All workers started (%d total)", len(m.workers)))
	return nil
}

// StopAll stops all workers gracefully within the given timeout.
// Returns error if any worker fails to stop within timeout.
func (m *WorkerManager) StopAll(timeout time.Duration) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var wg sync.WaitGroup
	errChan := make(chan error, len(m.workers))

	for name, worker := range m.workers {
		wg.Add(1)
		go func(n string, w *Worker) {
			defer wg.Done()
			if err := w.Stop(timeout); err != nil {
				errChan <- fmt.Errorf("worker '%s': %w", n, err)
			}
		}(name, worker)
	}

	wg.Wait()
	close(errChan)

	// Collect any errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to stop %d worker(s): %v", len(errors), errors)
	}

	logger.Info("All workers stopped successfully")
	return nil
}

// GetWorker returns a worker by name.
func (m *WorkerManager) GetWorker(name string) (*Worker, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	worker, exists := m.workers[name]
	return worker, exists
}

// Count returns the number of registered workers.
func (m *WorkerManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.workers)
}
