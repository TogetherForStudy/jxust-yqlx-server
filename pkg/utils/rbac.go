package utils

import (
	"context"
	"slices"

	"github.com/gin-gonic/gin"
)

// getGinContext 从 context.Context 中提取 *gin.Context
func getGinContext(ctx context.Context) *gin.Context {
	if c, ok := ctx.(*gin.Context); ok {
		return c
	}
	return nil
}

// GetUserRoles 从 context 中获取用户角色信息
func GetUserRoles(ctx context.Context) []string {
	c := getGinContext(ctx)
	if c == nil {
		return nil
	}
	if roles, ok := c.Get("user_roles"); ok {
		if roleList, ok := roles.([]string); ok {
			return roleList
		}
	}
	return nil
}

// HasUserRole 检查 context 中的用户是否拥有指定角色
func HasUserRole(ctx context.Context, role string) bool {
	roles := GetUserRoles(ctx)
	return slices.Contains(roles, role)
}

// IsAdmin 检查 context 中的用户是否是管理员
func IsAdmin(ctx context.Context) bool {
	c := getGinContext(ctx)
	if c == nil {
		return false
	}
	if isAdmin, ok := c.Get("is_admin"); ok {
		if admin, ok := isAdmin.(bool); ok {
			return admin
		}
	}
	// 如果没有显式设置，检查是否包含 admin 角色
	return HasUserRole(ctx, "admin")
}

// GetUserPermissions 从 context 中获取用户权限信息
func GetUserPermissions(ctx context.Context) []string {
	c := getGinContext(ctx)
	if c == nil {
		return nil
	}
	if permissions, ok := c.Get("user_permissions"); ok {
		if permList, ok := permissions.([]string); ok {
			return permList
		}
	}
	return nil
}

// HasPermission 检查 context 中的用户是否拥有指定权限
func HasPermission(ctx context.Context, permission string) bool {
	permissions := GetUserPermissions(ctx)
	return slices.Contains(permissions, permission)
}
