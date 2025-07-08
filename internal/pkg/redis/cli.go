package redis

import (
	"fmt"
	"sync"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/config"
	"github.com/redis/go-redis/v9"
)

type redisCli struct {
	c *redis.Client
}

func (r *redisCli) GetRedisCli() redis.UniversalClient {
	return r.c
}

var _once sync.Once

// NewRedisCli initializes and returns a RedisClient instance.
func NewRedisCli(cfg *config.Config) RedisClient {
	_once.Do(func() {
		redisClient := redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		})
		GlobalRedisCli = &redisCli{redisClient}
	})

	return GlobalRedisCli
}
