package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/config"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/cache"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	rediscache "github.com/redis/go-redis/v9"
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

// Logger 结构化日志中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		hasError, _ := c.Get("response_has_error")
		bodyStatusCode := 0
		if val, exists := c.Get("response_status_code"); exists {
			if code, ok := val.(int); ok {
				bodyStatusCode = code
			}
		}

		logFields := map[string]any{
			"action":      "http_request",
			"message":     "HTTP request processed",
			"status_code": statusCode,
			"latency_ms":  latency.Milliseconds(),
			"latency":     latency.String(),
			"body_size":   c.Writer.Size(),
		}
		if query != "" {
			logFields["query"] = query
		}
		if len(c.Errors) > 0 {
			logFields["errors"] = c.Errors.String()
		}
		if bodyStatusCode != 0 {
			logFields["body_status_code"] = bodyStatusCode
		}

		shouldLogDetails := statusCode != http.StatusOK || hasError == true
		if shouldLogDetails {
			logFields["body_message"], _ = c.Get("body_message")
		}

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
func AuthMiddleware(cfg *config.Config, ca cache.Cache) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ca == nil {
			helper.ErrorResponse(c, http.StatusServiceUnavailable, "鉴权缓存未初始化")
			c.Abort()
			return
		}

		tokenString := helper.GetAuthorizationToken(c)
		if tokenString == "" {
			logAuthRejected(c, "missing_authorization", 0, "")
			helper.ErrorResponse(c, http.StatusUnauthorized, "无效的 Authorization 头")
			c.Abort()
			return
		}

		claims, err := utils.ParseToken(tokenString, cfg.JWTSecret)
		if err != nil {
			logAuthRejected(c, "invalid_token", 0, "")
			helper.ErrorResponse(c, http.StatusUnauthorized, "无效的 Token")
			c.Abort()
			return
		}
		if claims.TokenType != constant.AuthTokenTypeAccess {
			logAuthRejected(c, "invalid_token_type", claims.UserID, claims.SID)
			helper.ErrorResponse(c, http.StatusUnauthorized, "Token 类型无效")
			c.Abort()
			return
		}
		if claims.UserID == 0 || claims.SID == "" || claims.JTI == "" || claims.IssuedAt == nil {
			logAuthRejected(c, "invalid_claims", claims.UserID, claims.SID)
			helper.ErrorResponse(c, http.StatusUnauthorized, "无效的 Token Claims")
			c.Abort()
			return
		}

		blocked, err := ca.Exists(c.Request.Context(), fmt.Sprintf(constant.AuthBlockedKeyFormat, claims.UserID))
		if err != nil {
			helper.ErrorResponse(c, http.StatusServiceUnavailable, "鉴权状态读取失败")
			c.Abort()
			return
		}
		if blocked {
			logAuthRejected(c, "blocked_user", claims.UserID, claims.SID)
			helper.ErrorResponse(c, http.StatusUnauthorized, "用户账号已被封禁")
			c.Abort()
			return
		}

		revokedSession, err := ca.Exists(c.Request.Context(), fmt.Sprintf(constant.AuthRevokedSessionKeyFormat, claims.SID))
		if err != nil {
			helper.ErrorResponse(c, http.StatusServiceUnavailable, "鉴权状态读取失败")
			c.Abort()
			return
		}
		if revokedSession {
			logAuthRejected(c, "revoked_session", claims.UserID, claims.SID)
			helper.ErrorResponse(c, http.StatusUnauthorized, "当前会话已失效")
			c.Abort()
			return
		}

		revokedBeforeStr, err := ca.Get(c.Request.Context(), fmt.Sprintf(constant.AuthRevokedBeforeKeyFormat, claims.UserID))
		if err != nil && !errors.Is(err, rediscache.Nil) {
			helper.ErrorResponse(c, http.StatusServiceUnavailable, "鉴权状态读取失败")
			c.Abort()
			return
		}
		if err == nil && revokedBeforeStr != "" {
			revokedBefore, parseErr := strconv.ParseInt(revokedBeforeStr, 10, 64)
			if parseErr != nil {
				helper.ErrorResponse(c, http.StatusServiceUnavailable, "鉴权状态解析失败")
				c.Abort()
				return
			}
			if claims.IssuedAt.Unix() <= revokedBefore {
				logAuthRejected(c, "revoked_before", claims.UserID, claims.SID)
				helper.ErrorResponse(c, http.StatusUnauthorized, "当前会话已失效")
				c.Abort()
				return
			}
		}

		c.Set("user_id", claims.UserID)
		c.Set(constant.AuthContextSessionID, claims.SID)
		c.Set(constant.AuthContextTokenJTI, claims.JTI)
		c.Set(constant.AuthContextTokenIAT, claims.IssuedAt.Unix())

		ctx := logger.EnrichContext(c.Request.Context(), map[string]any{
			"request_id": helper.GetRequestID(c),
			"user_id":    claims.UserID,
		})
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

func logAuthRejected(c *gin.Context, reasonCode string, userID uint, sid string) {
	fields := map[string]any{
		"action":      "auth_request_rejected",
		"message":     "request rejected by auth middleware",
		"reason_code": reasonCode,
	}
	if userID > 0 {
		fields["user_id"] = userID
	}
	if sid != "" {
		fields["sid"] = sid
	}
	logger.WarnGin(c, fields)
}

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.Request.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
			c.Request.Header.Set("X-Request-ID", requestID)
		}
		c.Header("X-Request-ID", requestID)
		c.Set(constant.RequestID, requestID)

		ctx := logger.EnrichContext(c.Request.Context(), map[string]any{
			"request_id": requestID,
		})
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
