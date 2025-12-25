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

	// 分布式锁
	// Lock 尝试获取分布式锁，返回是否成功获取
	Lock(ctx context.Context, key string, expiration time.Duration) (bool, error)
	// Unlock 释放分布式锁
	Unlock(ctx context.Context, key string) error
	// SetNX 仅当key不存在时设置值
	SetNX(ctx context.Context, key string, value string, expiration time.Duration) (bool, error)

	// Set 操作
	// SAdd 向集合中添加成员
	SAdd(ctx context.Context, key string, members ...interface{}) (int64, error)
	// SCard 获取集合的成员数量
	SCard(ctx context.Context, key string) (int64, error)
	// GetInt 获取整数值（用于计数器）
	GetInt(ctx context.Context, key string) (int64, error)

	Close() error
}

// GlobalCache is a global instance of Cache that can be used throughout the application.
var GlobalCache Cache

// DefaultExpiration is the default expiration time for cache items.
var DefaultExpiration = 30 * time.Minute
