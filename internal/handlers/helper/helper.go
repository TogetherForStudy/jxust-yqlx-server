package helper

import (
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

// GetUserId 获取用户ID（与 GetOpenID 相同，保持兼容性）
func GetUserId(c *gin.Context) string {
	return GetOpenID(c)
}
