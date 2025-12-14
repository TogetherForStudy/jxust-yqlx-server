package helper

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// GetOpenID 获取用户ID
func GetOpenID(c *gin.Context) string {
	userId, ok := c.Get("open_id")
	if !ok {
		return ""
	}
	if userIdStr, ok := userId.(string); ok {
		return userIdStr
	}
	return ""
}

// GetUserID 获取用户ID (uint类型)
func GetUserID(c *gin.Context) uint {
	userID, ok := c.Get("user_id")
	if !ok {
		return 0
	}
	if id, ok := userID.(uint); ok {
		return id
	}
	return 0
}

// GetAuthorizationToken 从请求头中提取Bearer Token
func GetAuthorizationToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}

	// 检查Bearer前缀
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		return ""
	}

	return tokenString
}
