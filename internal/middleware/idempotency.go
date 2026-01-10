package middleware

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/cache"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"

	json "github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
)

// IdempotencyResponse 缓存的响应结构
type IdempotencyResponse struct {
	StatusCode int         `json:"status_code"`
	Body       string      `json:"body"`
	Headers    http.Header `json:"headers"`
}

// responseWriter 包装gin的ResponseWriter以捕获响应
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// IdempotencyMiddleware 幂等性中间件
// 通过Redis存储请求的幂等性Key，防止重复提交
// 如果请求没有携带X-Idempotency-Key，在宽松模式下仅打印警告日志
func IdempotencyMiddleware(ca cache.Cache) gin.HandlerFunc {
	if ca == nil {
		ca = cache.GlobalCache
	}
	return func(c *gin.Context) {
		idempotencyMiddleware(c, ca, false)
	}
}

// IdempotencyMiddlewareStrict 严格模式的幂等性中间件
// 如果请求没有携带X-Idempotency-Key，将拒绝请求
func IdempotencyMiddlewareStrict(ca cache.Cache) gin.HandlerFunc {
	if ca == nil {
		ca = cache.GlobalCache
	}
	return func(c *gin.Context) {
		idempotencyMiddleware(c, ca, true)
	}
}

func idempotencyMiddleware(c *gin.Context, ca cache.Cache, strict bool) {
	idempotencyMiddlewareWithTTL(c, ca, strict, constant.IdempotencyExpiration)
}

// CreateIdempotencyMiddleware 创建幂等性中间件的工厂函数
// strict: true表示严格模式，false表示宽松模式
func CreateIdempotencyMiddleware(ca cache.Cache, strict bool) gin.HandlerFunc {
	if ca == nil {
		ca = cache.GlobalCache
	}
	return func(c *gin.Context) {
		idempotencyMiddleware(c, ca, strict)
	}
}

// IdempotencyRequired 返回严格模式的幂等性中间件（必须携带X-Idempotency-Key）
func IdempotencyRequired(ca cache.Cache) gin.HandlerFunc {
	if ca == nil {
		ca = cache.GlobalCache
	}
	return func(c *gin.Context) {
		idempotencyMiddleware(c, ca, true)
	}
}

// IdempotencyRecommended 返回宽松模式的幂等性中间件（推荐携带，但不强制）
func IdempotencyRecommended(ca cache.Cache) gin.HandlerFunc {
	if ca == nil {
		ca = cache.GlobalCache
	}
	return func(c *gin.Context) {
		idempotencyMiddleware(c, ca, false)
	}
}

// IdempotencyWithTTL 返回自定义过期时间的幂等性中间件
func IdempotencyWithTTL(ttl time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		idempotencyMiddlewareWithTTL(c, cache.GlobalCache, false, ttl)
	}
}

func idempotencyMiddlewareWithTTL(c *gin.Context, ca cache.Cache, strict bool, ttl time.Duration) {
	// 仅对写操作启用幂等性检查
	if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead || c.Request.Method == http.MethodOptions {
		c.Next()
		return
	}

	// 获取幂等性Key
	idempotencyKey := c.GetHeader(constant.IdempotencyKey)
	if idempotencyKey == "" {
		if strict {
			c.JSON(http.StatusBadRequest, dto.Response{
				RequestId:     helper.GetRequestID(c),
				StatusCode:    http.StatusBadRequest,
				StatusMessage: "缺少幂等性Key，请在Header中添加 X-Idempotency-Key",
			})
			c.Abort()
			return
		}
		// 宽松模式：仅打印警告日志，不阻止请求
		logger.WarnGin(c, map[string]any{
			"action":  "idempotency_check",
			"message": "请求缺少幂等性Key",
			"status":  "missing_key",
		})
		c.Next()
		return
	}

	// 检查缓存是否可用
	if ca == nil {
		logger.WarnGin(c, map[string]any{
			"action":          "idempotency_check",
			"message":         "缓存服务不可用，跳过幂等性检查",
			"idempotency_key": idempotencyKey,
		})
		c.Next()
		return
	}

	// 获取用户ID（如果已认证）
	userID := helper.GetUserID(c)
	var cacheKey string
	if userID != 0 {
		cacheKey = fmt.Sprintf("%s%d:%s", constant.IdempotencyCachePrefix, userID, idempotencyKey)
	} else {
		cacheKey = fmt.Sprintf("%s%s:%s", constant.IdempotencyCachePrefix, c.ClientIP(), idempotencyKey)
	}

	ctx := c.Request.Context()

	// 尝试获取分布式锁，防止并发请求
	lockKey := cacheKey + ":lock"
	locked, err := ca.Lock(ctx, lockKey, constant.IdempotencyLockTimeout)
	if err != nil {
		logger.ErrorGin(c, map[string]any{
			"action":          "idempotency_lock",
			"message":         "获取分布式锁失败",
			"error":           err.Error(),
			"cache_key":       cacheKey,
			"idempotency_key": idempotencyKey,
		})
		c.Next()
		return
	}

	if !locked {
		// 无法获取锁，说明有相同请求正在处理中
		c.JSON(http.StatusConflict, dto.Response{
			RequestId:     helper.GetRequestID(c),
			StatusCode:    http.StatusConflict,
			StatusMessage: "请求正在处理中，请稍后重试",
		})
		c.Abort()
		return
	}

	// 确保释放锁
	defer func() {
		if err := ca.Unlock(ctx, lockKey); err != nil {
			logger.ErrorGin(c, map[string]any{
				"action":          "idempotency_unlock",
				"message":         "释放分布式锁失败",
				"error":           err.Error(),
				"cache_key":       cacheKey,
				"idempotency_key": idempotencyKey,
			})
		}
	}()

	// 检查是否已有缓存的响应
	cachedResponse, err := ca.Get(ctx, cacheKey)
	if err == nil && cachedResponse != "" {
		// 已有缓存，直接返回
		var resp IdempotencyResponse
		if err := json.Unmarshal([]byte(cachedResponse), &resp); err != nil {
			logger.ErrorGin(c, map[string]any{
				"action":          "idempotency_unmarshal",
				"message":         "解析缓存响应失败",
				"error":           err.Error(),
				"cache_key":       cacheKey,
				"idempotency_key": idempotencyKey,
			})
			c.Next()
			return
		}

		logger.InfoGin(c, map[string]any{
			"action":          "idempotency_cache_hit",
			"message":         "命中缓存，返回已缓存的响应",
			"idempotency_key": idempotencyKey,
		})

		// 设置缓存的响应头
		for key, values := range resp.Headers {
			for _, value := range values {
				c.Header(key, value)
			}
		}
		c.Header("X-Idempotency-Replayed", "true")

		c.Data(resp.StatusCode, "application/json; charset=utf-8", []byte(resp.Body))
		c.Abort()
		return
	}

	// 标记请求正在处理
	if _, err := ca.SetNX(ctx, cacheKey, constant.IdempotencyStatusPending, ttl); err != nil {
		logger.ErrorGin(c, map[string]any{
			"action":          "idempotency_set_pending",
			"message":         "设置处理状态失败",
			"error":           err.Error(),
			"cache_key":       cacheKey,
			"idempotency_key": idempotencyKey,
		})
	}

	// 保存幂等性Key到上下文
	c.Set(constant.IdempotencyKeyCtx, idempotencyKey)

	// 包装ResponseWriter以捕获响应
	writer := &responseWriter{
		ResponseWriter: c.Writer,
		body:           bytes.NewBuffer(nil),
	}
	c.Writer = writer

	// 继续处理请求
	c.Next()

	// 请求处理完成后，缓存成功的响应
	if c.Writer.Status() < 400 {
		// 仅缓存成功的响应
		resp := IdempotencyResponse{
			StatusCode: c.Writer.Status(),
			Body:       writer.body.String(),
			Headers:    c.Writer.Header().Clone(),
		}

		respJSON, err := json.Marshal(resp)
		if err != nil {
			logger.ErrorGin(c, map[string]any{
				"action":          "idempotency_marshal",
				"message":         "序列化响应失败",
				"error":           err.Error(),
				"cache_key":       cacheKey,
				"idempotency_key": idempotencyKey,
			})
			return
		}

		if err := ca.Set(ctx, cacheKey, string(respJSON), &ttl); err != nil {
			logger.ErrorGin(c, map[string]any{
				"action":          "idempotency_cache_set",
				"message":         "缓存响应失败",
				"error":           err.Error(),
				"cache_key":       cacheKey,
				"idempotency_key": idempotencyKey,
			})
		} else {
			logger.InfoGin(c, map[string]any{
				"action":          "idempotency_cache_success",
				"message":         "响应已缓存",
				"idempotency_key": idempotencyKey,
				"expiration":      ttl.String(),
			})
		}
	} else {
		// 请求失败，删除pending状态，允许重试
		if err := ca.Delete(ctx, cacheKey); err != nil {
			logger.ErrorGin(c, map[string]any{
				"action":          "idempotency_delete_pending",
				"message":         "删除失败状态失败",
				"error":           err.Error(),
				"cache_key":       cacheKey,
				"idempotency_key": idempotencyKey,
			})
		}
	}
}
