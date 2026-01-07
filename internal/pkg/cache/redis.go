package cache

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/redis"
	rediscache "github.com/redis/go-redis/v9"
)

var _ Cache = (*redisCache)(nil)

type redisCache struct {
	cli redis.RedisClient
}

// NewRedisCache 创建Redis缓存实例并设置为全局缓存
func NewRedisCache(cli redis.RedisClient) Cache {
	cache := &redisCache{cli: cli}
	GlobalCache = cache
	return cache
}

func (r *redisCache) Get(ctx context.Context, key string) (string, error) {
	return r.cli.GetRedisCli().Get(ctx, key).Result()
}

func (r *redisCache) Set(ctx context.Context, key string, value string, expiration *time.Duration) error {
	if expiration == nil {
		expiration = &DefaultExpiration
	}
	return r.cli.GetRedisCli().Set(ctx, key, value, *expiration).Err()
}

func (r *redisCache) Delete(ctx context.Context, key string) error {
	return r.cli.GetRedisCli().Del(ctx, key).Err()
}

func (r *redisCache) Exists(ctx context.Context, key string) (bool, error) {
	exists, err := r.cli.GetRedisCli().Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

func (r *redisCache) Incr(ctx context.Context, key string) (int64, error) {
	return r.cli.GetRedisCli().Incr(ctx, key).Result()
}

func (r *redisCache) Decr(ctx context.Context, key string) (int64, error) {
	return r.cli.GetRedisCli().Decr(ctx, key).Result()
}

func (r *redisCache) Lock(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	lockKey := "lock:" + key
	return r.cli.GetRedisCli().SetNX(ctx, lockKey, "1", expiration).Result()
}

func (r *redisCache) Unlock(ctx context.Context, key string) error {
	lockKey := "lock:" + key
	return r.cli.GetRedisCli().Del(ctx, lockKey).Err()
}

func (r *redisCache) SetNX(ctx context.Context, key string, value string, expiration time.Duration) (bool, error) {
	return r.cli.GetRedisCli().SetNX(ctx, key, value, expiration).Result()
}

func (r *redisCache) SAdd(ctx context.Context, key string, members ...interface{}) (int64, error) {
	return r.cli.GetRedisCli().SAdd(ctx, key, members...).Result()
}

func (r *redisCache) SCard(ctx context.Context, key string) (int64, error) {
	return r.cli.GetRedisCli().SCard(ctx, key).Result()
}

func (r *redisCache) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	return r.cli.GetRedisCli().SIsMember(ctx, key, member).Result()
}

func (r *redisCache) GetInt(ctx context.Context, key string) (int64, error) {
	val, err := r.cli.GetRedisCli().Get(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	result, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, err
	}
	return result, nil
}

func (r *redisCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return r.cli.GetRedisCli().Expire(ctx, key, expiration).Err()
}

func (r *redisCache) ZAdd(ctx context.Context, key string, score float64, member interface{}) error {
	return r.cli.GetRedisCli().ZAdd(ctx, key, rediscache.Z{
		Score:  score,
		Member: member,
	}).Err()
}

func (r *redisCache) ZCount(ctx context.Context, key string, min, max float64) (int64, error) {
	return r.cli.GetRedisCli().ZCount(ctx, key, fmt.Sprintf("%.0f", min), fmt.Sprintf("%.0f", max)).Result()
}

func (r *redisCache) ZRemRangeByScore(ctx context.Context, key string, min, max float64) (int64, error) {
	return r.cli.GetRedisCli().ZRemRangeByScore(ctx, key, fmt.Sprintf("%.0f", min), fmt.Sprintf("%.0f", max)).Result()
}

func (r *redisCache) Close() error {
	return r.cli.GetRedisCli().Close()
}
