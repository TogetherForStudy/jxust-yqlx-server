package logger

import (
	"context"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"github.com/gin-gonic/gin"
)

// GinContextToContext extracts metadata from gin.Context and enriches context.Context
// This function bridges gin.Context and context.Context for structured logging
func GinContextToContext(c *gin.Context) context.Context {
	ctx := c.Request.Context()

	// Extract RequestID (already in context from middleware)
	if requestID, exists := c.Get(constant.RequestID); exists {
		if reqID, ok := requestID.(string); ok {
			ctx = context.WithValue(ctx, constant.RequestID, reqID)
		}
	}

	// Extract UserID
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(uint); ok {
			ctx = context.WithValue(ctx, ctxKeyUserID, uid)
		}
	}

	// Extract UserRoles
	if userRoles, exists := c.Get("user_roles"); exists {
		if roles, ok := userRoles.([]string); ok {
			ctx = context.WithValue(ctx, ctxKeyUserRoles, roles)
		}
	}

	// Extract IdempotencyKey
	if idempotencyKey, exists := c.Get(constant.IdempotencyKeyCtx); exists {
		if key, ok := idempotencyKey.(string); ok {
			ctx = context.WithValue(ctx, ctxKeyIdempotencyKey, key)
		}
	}

	// Extract request metadata
	ctx = context.WithValue(ctx, ctxKeyClientIP, c.ClientIP())
	ctx = context.WithValue(ctx, ctxKeyMethod, c.Request.Method)
	ctx = context.WithValue(ctx, ctxKeyPath, c.Request.URL.Path)
	ctx = context.WithValue(ctx, ctxKeyProtocol, c.Request.Proto)
	ctx = context.WithValue(ctx, ctxKeyUserAgent, c.Request.UserAgent())
	ctx = context.WithValue(ctx, ctxKeyTimestamp, time.Now())

	return ctx
}

// InfoGin logs an info level message from gin.Context
func InfoGin(c *gin.Context, msg map[string]any) {
	ctx := GinContextToContext(c)
	merged := mergeContextAndMessage(ctx, msg)
	l := NewStructuredClsLogging(constant.InfoLevel, merged)
	zlog.Infoln(l.String())
	safeSendLog(l)
}

// DebugGin logs a debug level message from gin.Context
func DebugGin(c *gin.Context, msg map[string]any) {
	ctx := GinContextToContext(c)
	merged := mergeContextAndMessage(ctx, msg)
	l := NewStructuredClsLogging(constant.DebugLevel, merged)
	zlog.Debugln(l.String())
	safeSendLog(l)
}

// WarnGin logs a warning level message from gin.Context
func WarnGin(c *gin.Context, msg map[string]any) {
	ctx := GinContextToContext(c)
	merged := mergeContextAndMessage(ctx, msg)
	l := NewStructuredClsLogging(constant.WarnLevel, merged)
	zlog.Warnln(l.String())
	safeSendLog(l)
}

// ErrorGin logs an error level message from gin.Context
func ErrorGin(c *gin.Context, msg map[string]any) {
	ctx := GinContextToContext(c)
	merged := mergeContextAndMessage(ctx, msg)
	l := NewStructuredClsLogging(constant.ErrorLevel, merged)
	zlog.Errorln(l.String())
	safeSendLog(l)
}

// FatalGin logs a fatal level message from gin.Context
func FatalGin(c *gin.Context, msg map[string]any) {
	ctx := GinContextToContext(c)
	merged := mergeContextAndMessage(ctx, msg)
	l := NewStructuredClsLogging(constant.FatalLevel, merged)
	safeSendLog(l)
	// Wait for log to be sent before exiting
	ShutdownLogger(5 * time.Second)
	zlog.Fatalln(l.String())
}

// PanicGin logs a panic level message from gin.Context
func PanicGin(c *gin.Context, msg map[string]any) {
	ctx := GinContextToContext(c)
	merged := mergeContextAndMessage(ctx, msg)
	l := NewStructuredClsLogging(constant.PanicLevel, merged)
	safeSendLog(l)
	// Wait for log to be sent before panicking
	ShutdownLogger(5 * time.Second)
	zlog.Errorln(l.String()) // Log to zap before panicking
	panic(l.String())
}
