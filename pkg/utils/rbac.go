package utils

import (
	"context"
	"slices"
)

type contextKey string

const (
	userRolesKey       contextKey = "user_roles"
	userPermissionsKey contextKey = "user_permissions"
	isAdminKey         contextKey = "is_admin"
)

// WithUserRoles 将用户角色信息存储到 context 中
func WithUserRoles(ctx context.Context, roles []string) context.Context {
	return context.WithValue(ctx, userRolesKey, roles)
}

// GetUserRoles 从 context 中获取用户角色信息
func GetUserRoles(ctx context.Context) []string {
	if roles, ok := ctx.Value(userRolesKey).([]string); ok {
		return roles
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
	if isAdmin, ok := ctx.Value(isAdminKey).(bool); ok {
		return isAdmin
	}
	// 如果没有显式设置，检查是否包含 admin 角色
	return HasUserRole(ctx, "admin")
}

// WithIsAdmin 将管理员标识存储到 context 中
func WithIsAdmin(ctx context.Context, isAdmin bool) context.Context {
	return context.WithValue(ctx, isAdminKey, isAdmin)
}

// WithUserPermissions 将用户权限信息存储到 context 中
func WithUserPermissions(ctx context.Context, permissions []string) context.Context {
	return context.WithValue(ctx, userPermissionsKey, permissions)
}

// GetUserPermissions 从 context 中获取用户权限信息
func GetUserPermissions(ctx context.Context) []string {
	if permissions, ok := ctx.Value(userPermissionsKey).([]string); ok {
		return permissions
	}
	return nil
}

// HasPermission 检查 context 中的用户是否拥有指定权限
func HasPermission(ctx context.Context, permission string) bool {
	permissions := GetUserPermissions(ctx)
	return slices.Contains(permissions, permission)
}
