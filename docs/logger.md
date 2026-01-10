# 日志模块重构文档

## 概述

重构后的日志模块支持结构化日志记录，自动从 `context.Context` 和 `gin.Context` 中提取请求元数据，并将日志存储到腾讯云 CLS。

## 新增功能

### 1. Context 字段自动提取

`parserCtxToMapStringString` 函数现在会自动从 `context.Context` 中提取以下字段：

- **request_id**: 请求追踪ID
- **timestamp**: 请求时间戳
- **user_id**: 用户ID
- **is_admin**: 是否管理员
- **user_roles**: 用户角色列表（逗号分隔）
- **user_permissions**: 用户权限列表（逗号分隔）
- **client_ip**: 客户端IP
- **method**: HTTP方法
- **path**: 请求路径
- **protocol**: HTTP协议版本
- **user_agent**: User-Agent
- **idempotency_key**: 幂等性Key

### 2. 完整的日志级别支持

提供了所有日志级别的 Context 版本：

- `DebugCtx(ctx context.Context, msg map[string]any)`
- `InfoCtx(ctx context.Context, msg map[string]any)`
- `WarnCtx(ctx context.Context, msg map[string]any)`
- `ErrorCtx(ctx context.Context, msg map[string]any)`
- `FatalCtx(ctx context.Context, msg map[string]any)`
- `PanicCtx(ctx context.Context, msg map[string]any)`

### 3. Gin 上下文快捷函数

提供了直接从 `gin.Context` 记录日志的便捷函数：

- `DebugGin(c *gin.Context, msg map[string]any)`
- `InfoGin(c *gin.Context, msg map[string]any)`
- `WarnGin(c *gin.Context, msg map[string]any)`
- `ErrorGin(c *gin.Context, msg map[string]any)`
- `FatalGin(c *gin.Context, msg map[string]any)`
- `PanicGin(c *gin.Context, msg map[string]any)`

### 4. Context 增强工具

- `EnrichContext(ctx context.Context, fields map[string]any) context.Context`: 向 Context 中添加结构化字段
- `GinContextToContext(c *gin.Context) context.Context`: 将 Gin Context 转换为增强的 Context

## 使用示例

### 示例 1: 在 Service 层使用 Context

```go
package services

import (
    "context"
    "github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"
)

func (s *UserService) GetUser(ctx context.Context, userID uint) (*User, error) {
    // 记录信息日志，自动包含 request_id, user_id 等上下文信息
    logger.InfoCtx(ctx, map[string]any{
        "action": "get_user",
        "target_user_id": userID,
        "message": "fetching user from database",
    })
    
    user, err := s.db.FindUser(userID)
    if err != nil {
        // 记录错误日志
        logger.ErrorCtx(ctx, map[string]any{
            "action": "get_user",
            "target_user_id": userID,
            "error": err.Error(),
            "message": "failed to fetch user",
        })
        return nil, err
    }
    
    return user, nil
}
```

### 示例 2: 在 Handler 层使用 Gin Context

```go
package handlers

import (
    "github.com/gin-gonic/gin"
    "github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"
)

func (h *UserHandler) GetProfile(c *gin.Context) {
    // 直接从 gin.Context 记录日志，自动提取所有请求元数据
    logger.InfoGin(c, map[string]any{
        "action": "get_profile",
        "message": "user requested profile",
    })
    
    // 或者转换为 context.Context 后使用
    ctx := logger.GinContextToContext(c)
    
    user, err := h.service.GetUser(ctx, helper.GetUserID(c))
    if err != nil {
        logger.ErrorCtx(ctx, map[string]any{
            "action": "get_profile",
            "error": err.Error(),
            "message": "failed to get user profile",
        })
        helper.ErrorResponse(c, 500, "获取用户信息失败")
        return
    }
    
    logger.InfoGin(c, map[string]any{
        "action": "get_profile_success",
        "message": "profile retrieved successfully",
    })
    
    helper.SuccessResponse(c, user)
}
```

### 示例 3: 手动增强 Context

```go
package services

import (
    "context"
    "github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"
)

func (s *Service) ProcessTask(ctx context.Context, taskID string) error {
    // 手动向 context 中添加字段
    enrichedCtx := logger.EnrichContext(ctx, map[string]any{
        "task_id": taskID,
        "worker_id": "worker-001",
    })
    
    logger.InfoCtx(enrichedCtx, map[string]any{
        "action": "process_task",
        "message": "starting task processing",
    })
    
    // 继续使用增强后的 context
    return s.doWork(enrichedCtx)
}
```

### 示例 4: 在中间件中使用

```go
package middleware

import (
    "github.com/gin-gonic/gin"
    "github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"
    "time"
)

func RequestLogger() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        
        logger.DebugGin(c, map[string]any{
            "message": "request started",
        })
        
        c.Next()
        
        latency := time.Since(start)
        statusCode := c.Writer.Status()
        
        logger.InfoGin(c, map[string]any{
            "message": "request completed",
            "latency_ms": latency.Milliseconds(),
            "status_code": statusCode,
        })
    }
}
```

## 输出格式

所有日志都会：
1. 输出到控制台（通过 zap logger）
2. 发送到腾讯云 CLS（结构化存储）

### CLS 日志字段示例

```json
{
  "level": "info",
  "request_id": "123e4567-e89b-12d3-a456-426614174000",
  "timestamp": "2026-01-10T07:56:57Z",
  "user_id": "12345",
  "is_admin": "false",
  "user_roles": "student,member",
  "user_permissions": "read:courses,write:reviews",
  "client_ip": "192.168.1.100",
  "method": "GET",
  "path": "/api/v0/users/profile",
  "protocol": "HTTP/1.1",
  "user_agent": "Mozilla/5.0...",
  "action": "get_profile",
  "message": "user requested profile"
}
```

## 迁移指南

### 从旧的日志方式迁移

**旧方式:**
```go
logger.Infof("User %d fetched profile", userID)
```

**新方式:**
```go
logger.InfoCtx(ctx, map[string]any{
    "action": "get_profile",
    "user_id": userID,
    "message": "user fetched profile",
})
```

### 最佳实践

1. **在 Handler 层**: 使用 `logger.InfoGin()`, `logger.ErrorGin()` 等函数
2. **在 Service 层**: 使用 `logger.InfoCtx()`, `logger.ErrorCtx()` 等函数
3. **传递 Context**: 始终将 `context.Context` 从 Handler 传递到 Service
4. **结构化字段**: 使用有意义的字段名（如 `action`, `error`, `message`）
5. **避免敏感信息**: 不要记录密码、token 等敏感信息

## Context Keys 定义

以下 context keys 在 `pkg/logger/helper.go` 中定义：

```go
type contextKey string

const (
    ctxKeyUserID          contextKey = "user_id"
    ctxKeyUserRoles       contextKey = "user_roles"
    ctxKeyUserPermissions contextKey = "user_permissions"
    ctxKeyIsAdmin         contextKey = "is_admin"
    ctxKeyClientIP        contextKey = "client_ip"
    ctxKeyMethod          contextKey = "method"
    ctxKeyPath            contextKey = "path"
    ctxKeyProtocol        contextKey = "protocol"
    ctxKeyUserAgent       contextKey = "user_agent"
    ctxKeyTimestamp       contextKey = "timestamp"
    ctxKeyIdempotencyKey  contextKey = "idempotency_key"
)
```

## 注意事项

1. **RequestID**: 由 `RequestID()` 中间件自动设置
2. **UserID**: 由 `AuthMiddleware()` 中间件自动设置
3. **RBAC 信息**: 仅在使用 `RequirePermission()` 中间件时可用
4. **幂等性Key**: 仅在使用幂等性中间件且客户端提供时可用
5. **Context 传递**: Service 层需要接收 `context.Context` 参数才能获取这些信息

## 文件清单

- `pkg/logger/helper.go`: Context 解析和增强工具
- `pkg/logger/ctxLogger.go`: 基于 Context 的日志函数
- `pkg/logger/gin_helper.go`: Gin Context 日志快捷函数
- `pkg/constant/const.go`: 常量定义（RequestID 等）
