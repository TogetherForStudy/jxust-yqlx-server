package worker

import (
	"context"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/cache"
)

// QueueProvider defines the interface for queue backends.
// Implementations can use Redis, RabbitMQ, or other message queues.
type QueueProvider interface {
	// Push adds a task to the queue (left push for FIFO with RPop)
	Push(ctx context.Context, queueKey string, taskData string) error

	// Pop removes and returns a task from the queue (right pop for FIFO)
	Pop(ctx context.Context, queueKey string) (string, error)

	// Length returns the current queue length
	Length(ctx context.Context, queueKey string) (int64, error)
}

// RedisQueueProvider implements QueueProvider using Redis lists.
type RedisQueueProvider struct {
	cache cache.Cache
}

// NewRedisQueueProvider creates a new Redis-backed queue provider.
func NewRedisQueueProvider(c cache.Cache) *RedisQueueProvider {
	return &RedisQueueProvider{
		cache: c,
	}
}

// Push adds a task to the queue using LPUSH (left push).
func (r *RedisQueueProvider) Push(ctx context.Context, queueKey string, taskData string) error {
	_, err := r.cache.LPush(ctx, queueKey, taskData)
	return err
}

// Pop removes and returns a task from the queue using RPOP (right pop).
// Returns empty string if queue is empty.
func (r *RedisQueueProvider) Pop(ctx context.Context, queueKey string) (string, error) {
	return r.cache.RPop(ctx, queueKey)
}

// Length returns the current queue length using LLEN.
func (r *RedisQueueProvider) Length(ctx context.Context, queueKey string) (int64, error) {
	return r.cache.LLen(ctx, queueKey)
}
