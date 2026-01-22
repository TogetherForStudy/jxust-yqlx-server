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
	IdempotencyExpiration    = 10 * time.Minute
	IdempotencyLockTimeout   = 30 * time.Second
	IdempotencyStatusPending = "pending"

	LLMModel = "gpt-3.5-turbo"
)
