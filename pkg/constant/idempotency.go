package constant

import "time"

const (
	IdempotencyKey           = "X-Idempotency-Key"
	IdempotencyKeyCtx        = "idempotency_key"
	IdempotencyCachePrefix   = "idempotent:"
	IdempotencyExpiration    = 10 * time.Minute
	IdempotencyLockTimeout   = 30 * time.Second
	IdempotencyStatusPending = "pending"
)
