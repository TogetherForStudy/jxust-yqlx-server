package middleware

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/config"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/cache"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/utils"

	"github.com/gin-gonic/gin"
	rediscache "github.com/redis/go-redis/v9"
)

// AuthMiddleware JWT认证中间件
func AuthMiddleware(cfg *config.Config, ca cache.Cache) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ca == nil {
			helper.HandleErrCode(c, constant.AuthCacheUnavailable)
			c.Abort()
			return
		}

		tokenString := helper.GetAuthorizationToken(c)
		if tokenString == "" {
			logAuthRejected(c, "missing_authorization", 0, "")
			helper.HandleErrCode(c, constant.AuthInvalidAuthorizationHeader)
			c.Abort()
			return
		}

		claims, err := utils.ParseToken(tokenString, cfg.JWTSecret)
		if err != nil {
			logAuthRejected(c, "invalid_token", 0, "")
			helper.HandleErrCode(c, constant.AuthInvalidToken)
			c.Abort()
			return
		}
		if claims.TokenType != constant.AuthTokenTypeAccess {
			logAuthRejected(c, "invalid_token_type", claims.UserID, claims.SID)
			helper.HandleErrCode(c, constant.AuthInvalidTokenType)
			c.Abort()
			return
		}
		if claims.UserID == 0 || claims.SID == "" || claims.JTI == "" || claims.IssuedAt == nil {
			logAuthRejected(c, "invalid_claims", claims.UserID, claims.SID)
			helper.HandleErrCode(c, constant.AuthInvalidTokenClaims)
			c.Abort()
			return
		}

		blocked, err := ca.Exists(c.Request.Context(), fmt.Sprintf(constant.AuthBlockedKeyFormat, claims.UserID))
		if err != nil {
			helper.HandleErrCode(c, constant.AuthStateReadFailed)
			c.Abort()
			return
		}
		if blocked {
			logAuthRejected(c, "blocked_user", claims.UserID, claims.SID)
			helper.HandleErrCode(c, constant.AuthAccountBlocked)
			c.Abort()
			return
		}

		revokedSession, err := ca.Exists(c.Request.Context(), fmt.Sprintf(constant.AuthRevokedSessionKeyFormat, claims.SID))
		if err != nil {
			helper.HandleErrCode(c, constant.AuthStateReadFailed)
			c.Abort()
			return
		}
		if revokedSession {
			logAuthRejected(c, "revoked_session", claims.UserID, claims.SID)
			helper.HandleErrCode(c, constant.AuthSessionInvalid)
			c.Abort()
			return
		}

		revokedBeforeStr, err := ca.Get(c.Request.Context(), fmt.Sprintf(constant.AuthRevokedBeforeKeyFormat, claims.UserID))
		if err != nil && !errors.Is(err, rediscache.Nil) {
			helper.HandleErrCode(c, constant.AuthStateReadFailed)
			c.Abort()
			return
		}
		if err == nil && revokedBeforeStr != "" {
			revokedBefore, parseErr := strconv.ParseInt(revokedBeforeStr, 10, 64)
			if parseErr != nil {
				helper.HandleErrCode(c, constant.AuthStateParseFailed)
				c.Abort()
				return
			}
			if claims.IssuedAt.Unix() <= revokedBefore {
				logAuthRejected(c, "revoked_before", claims.UserID, claims.SID)
				helper.HandleErrCode(c, constant.AuthSessionInvalid)
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
