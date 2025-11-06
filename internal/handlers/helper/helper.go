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

// GetUserRole 获取用户角色
func GetUserRole(c *gin.Context) uint8 {
	role, ok := c.Get("role")
	if !ok {
		return 0
	}
	if roleVal, ok := role.(uint8); ok {
		return roleVal
	}
	return 0
}
