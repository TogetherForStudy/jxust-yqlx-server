// Package redis provides a Redis client for interacting with Redis databases.
package redis

import (
	"github.com/redis/go-redis/v9"
)

type RedisClient interface {
	GetRedisCli() redis.UniversalClient
}

var GlobalRedisCli RedisClient
