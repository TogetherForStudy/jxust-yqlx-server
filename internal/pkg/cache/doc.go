// Package cache provides a simple in-memory cache implementation.
package cache

import (
	"context"
	"time"
)

type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, expiration *time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)

	// 计数器
	Incr(ctx context.Context, key string) (int64, error)
	Decr(ctx context.Context, key string) (int64, error)

	Close() error
}

// GlobalCache is a global instance of Cache that can be used throughout the application.
var GlobalCache Cache

// DefaultExpiration is the default expiration time for cache items.
var DefaultExpiration = 30 * time.Minute
