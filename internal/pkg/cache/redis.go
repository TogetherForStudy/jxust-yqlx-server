package cache

import (
	"context"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/redis"
)

var _ Cache = (*redisCache)(nil)

type redisCache struct {
	cli redis.RedisClient
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
func (r *redisCache) Close() error {
	return r.cli.GetRedisCli().Close()
}
