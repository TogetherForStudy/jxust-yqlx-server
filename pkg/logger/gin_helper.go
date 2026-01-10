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

	// Extract IsAdmin
	if isAdmin, exists := c.Get("is_admin"); exists {
		if admin, ok := isAdmin.(bool); ok {
			ctx = context.WithValue(ctx, ctxKeyIsAdmin, admin)
		}
	}

	// Extract UserRoles
	if userRoles, exists := c.Get("user_roles"); exists {
		if roles, ok := userRoles.([]string); ok {
			ctx = context.WithValue(ctx, ctxKeyUserRoles, roles)
		}
	}

	// Extract UserPermissions
	if userPermissions, exists := c.Get("user_permissions"); exists {
		if perms, ok := userPermissions.([]string); ok {
			ctx = context.WithValue(ctx, ctxKeyUserPermissions, perms)
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
	InfoCtx(ctx, msg)
}

// DebugGin logs a debug level message from gin.Context
func DebugGin(c *gin.Context, msg map[string]any) {
	ctx := GinContextToContext(c)
	DebugCtx(ctx, msg)
}

// WarnGin logs a warning level message from gin.Context
func WarnGin(c *gin.Context, msg map[string]any) {
	ctx := GinContextToContext(c)
	WarnCtx(ctx, msg)
}

// ErrorGin logs an error level message from gin.Context
func ErrorGin(c *gin.Context, msg map[string]any) {
	ctx := GinContextToContext(c)
	ErrorCtx(ctx, msg)
}

// FatalGin logs a fatal level message from gin.Context
func FatalGin(c *gin.Context, msg map[string]any) {
	ctx := GinContextToContext(c)
	FatalCtx(ctx, msg)
}

// PanicGin logs a panic level message from gin.Context
func PanicGin(c *gin.Context, msg map[string]any) {
	ctx := GinContextToContext(c)
	PanicCtx(ctx, msg)
}
