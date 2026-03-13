package logger

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/utils"
)

// Context keys for storing request metadata
type contextKey string

const (
	ctxKeyUserID    contextKey = "user_id"
	ctxKeyUserRoles contextKey = "user_roles"

	ctxKeyClientIP       contextKey = "client_ip"
	ctxKeyMethod         contextKey = "method"
	ctxKeyPath           contextKey = "path"
	ctxKeyProtocol       contextKey = "protocol"
	ctxKeyUserAgent      contextKey = "user_agent"
	ctxKeyTimestamp      contextKey = "timestamp"
	ctxKeyIdempotencyKey contextKey = "idempotency_key"
)

func parserMapStringAnyToMapStringString(m map[string]any) map[string]string {
	result := make(map[string]string)
	for k, v := range m {
		result[k] = fmt.Sprintf("%v", v)
	}
	return result
}

// parserCtxToMapStringString extracts structured logging fields from context
func parserCtxToMapStringString(ctx context.Context) map[string]string {
	result := make(map[string]string)

	// Extract RequestID
	result["request_id"] = utils.GetRequestID(ctx)

	// Extract Timestamp
	if timestamp, ok := ctx.Value(ctxKeyTimestamp).(time.Time); ok {
		result["timestamp"] = timestamp.Format(time.RFC3339)
	} else {
		result["timestamp"] = time.Now().Format(time.RFC3339)
	}

	// Extract UserID
	if userID, ok := ctx.Value(ctxKeyUserID).(uint); ok && userID > 0 {
		result["user_id"] = fmt.Sprintf("%d", userID)
	}

	// Extract UserRoles
	if userRoles, ok := ctx.Value(ctxKeyUserRoles).([]string); ok && len(userRoles) > 0 {
		result["user_roles"] = strings.Join(userRoles, ",")
	}

	// Extract ClientIP
	if clientIP, ok := ctx.Value(ctxKeyClientIP).(string); ok && clientIP != "" {
		result["client_ip"] = clientIP
	}

	// Extract Method
	if method, ok := ctx.Value(ctxKeyMethod).(string); ok && method != "" {
		result["method"] = method
	}

	// Extract Path
	if path, ok := ctx.Value(ctxKeyPath).(string); ok && path != "" {
		result["path"] = path
	}

	// Extract Protocol
	if protocol, ok := ctx.Value(ctxKeyProtocol).(string); ok && protocol != "" {
		result["protocol"] = protocol
	}

	// Extract UserAgent
	if userAgent, ok := ctx.Value(ctxKeyUserAgent).(string); ok && userAgent != "" {
		result["user_agent"] = userAgent
	}

	// Extract IdempotencyKey
	if idempotencyKey, ok := ctx.Value(constant.IdempotencyKeyCtx).(string); ok && idempotencyKey != "" {
		result["idempotency_key"] = idempotencyKey
	}

	return result
}

// EnrichContext enriches context with request metadata from various sources
// This helper function can be called to populate context with logging fields
func EnrichContext(ctx context.Context, fields map[string]any) context.Context {
	if requestID, ok := fields["request_id"].(string); ok {
		ctx = context.WithValue(ctx, constant.RequestID, requestID)
	}
	if userID, ok := fields["user_id"].(uint); ok {
		ctx = context.WithValue(ctx, ctxKeyUserID, userID)
	}

	if userRoles, ok := fields["user_roles"].([]string); ok {
		ctx = context.WithValue(ctx, ctxKeyUserRoles, userRoles)
	}

	if clientIP, ok := fields["client_ip"].(string); ok {
		ctx = context.WithValue(ctx, ctxKeyClientIP, clientIP)
	}
	if method, ok := fields["method"].(string); ok {
		ctx = context.WithValue(ctx, ctxKeyMethod, method)
	}
	if path, ok := fields["path"].(string); ok {
		ctx = context.WithValue(ctx, ctxKeyPath, path)
	}
	if protocol, ok := fields["protocol"].(string); ok {
		ctx = context.WithValue(ctx, ctxKeyProtocol, protocol)
	}
	if userAgent, ok := fields["user_agent"].(string); ok {
		ctx = context.WithValue(ctx, ctxKeyUserAgent, userAgent)
	}
	if timestamp, ok := fields["timestamp"].(time.Time); ok {
		ctx = context.WithValue(ctx, ctxKeyTimestamp, timestamp)
	}
	if idempotencyKey, ok := fields["idempotency_key"].(string); ok {
		ctx = context.WithValue(ctx, ctxKeyIdempotencyKey, idempotencyKey)
	}
	return ctx
}
