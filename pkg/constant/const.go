package constant

import (
	"time"
)

const (
	RequestID      = "request_id"
	DefaultExpired = 30 * time.Minute

	// 幂等性相关常量
	IdempotencyKey           = "X-Idempotency-Key"
	IdempotencyKeyCtx        = "idempotency_key"
	IdempotencyCachePrefix   = "idempotent:"
	IdempotencyExpiration    = 24 * time.Hour
	IdempotencyLockTimeout   = 30 * time.Second
	IdempotencyStatusPending = "pending"
)
