package middleware

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/config"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// CORS 跨域中间件
func CORS() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization, Cache-Control, X-File-Name, X-Idempotency-Key, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type, X-Idempotency-Replayed, X-Request-ID")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})
}

// bodyLogWriter 包装gin.ResponseWriter以捕获响应体
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// Logger 结构化日志中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 记录请求开始时间
		start := time.Now()
		query := c.Request.URL.RawQuery

		// 处理请求
		c.Next()

		// 计算请求处理时长
		latency := time.Since(start)
		statusCode := c.Writer.Status()

		// 检查是否有错误标记（由ErrorResponse/ValidateResponse设置）
		hasError, _ := c.Get("response_has_error")
		bodyStatusCode := 0
		if val, exists := c.Get("response_status_code"); exists {
			if code, ok := val.(int); ok {
				bodyStatusCode = code
			}
		}

		// 构建结构化日志字段
		logFields := map[string]any{
			"action":      "http_request",
			"message":     "HTTP request processed",
			"status_code": statusCode,
			"latency_ms":  latency.Milliseconds(),
			"latency":     latency.String(),
			"body_size":   c.Writer.Size(),
		}

		// 添加查询参数（如果存在）
		if query != "" {
			logFields["query"] = query
		}

		// 添加错误信息（如果存在）
		if len(c.Errors) > 0 {
			logFields["errors"] = c.Errors.String()
		}

		// 添加body中的StatusCode（如果存在且不为0）
		if bodyStatusCode != 0 {
			logFields["body_status_code"] = bodyStatusCode
		}

		// 如果HTTP状态码不是200，或者有错误标记，记录详细信息
		shouldLogDetails := statusCode != http.StatusOK || hasError == true
		if shouldLogDetails {
			logFields["body_message"], _ = c.Get("body_message")
		}

		// 根据HTTP状态码和body中的StatusCode选择日志级别
		// 优先检查bodyStatusCode，如果它表示错误则按它的级别记录
		effectiveStatusCode := statusCode
		if bodyStatusCode >= 400 {
			effectiveStatusCode = bodyStatusCode
		}

		switch {
		case effectiveStatusCode >= 500:
			logger.ErrorGin(c, logFields)
		case effectiveStatusCode >= 400:
			logger.WarnGin(c, logFields)
		case effectiveStatusCode >= 300:
			logger.InfoGin(c, logFields)
		default:
			logger.InfoGin(c, logFields)
		}
	}
}

// AuthMiddleware JWT认证中间件
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			helper.ErrorResponse(c, http.StatusUnauthorized, "未授权访问")
			c.Abort()
			return
		}

		// 检查Bearer前缀
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			helper.ErrorResponse(c, http.StatusUnauthorized, "无效的 Authorization 头")
			c.Abort()
			return
		}

		// 解析JWT token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(cfg.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			helper.ErrorResponse(c, http.StatusUnauthorized, "无效的 Token")
			c.Abort()
			return
		}

		// 获取用户信息
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			userID := uint(claims["user_id"].(float64))

			c.Set("user_id", userID)
		} else {
			helper.ErrorResponse(c, http.StatusUnauthorized, "无效的 Token Claims")
			c.Abort()
			return
		}

		c.Next()
	}
}

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.Request.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
			c.Request.Header.Set("X-Request-ID", requestID)
		}
		c.Set(constant.RequestID, requestID)
		c.Next()
	}
}
