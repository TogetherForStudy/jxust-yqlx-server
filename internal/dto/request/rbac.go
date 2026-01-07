package request

// CreateRoleRequest 创建角色
type CreateRoleRequest struct {
	RoleTag     string `json:"role_tag" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// UpdateRoleRequest 更新角色
type UpdateRoleRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description"`
}

// CreatePermissionRequest 创建权限
type CreatePermissionRequest struct {
	PermissionTag string `json:"permission_tag" binding:"required"`
	Name          string `json:"name" binding:"required"`
	Description   string `json:"description"`
}

// UpdateRolePermissionsRequest 角色权限重置请求
type UpdateRolePermissionsRequest struct {
	PermissionIDs []uint `json:"permission_ids" binding:"required"`
}

// UpdateUserRolesRequest 更新用户角色请求
type UpdateUserRolesRequest struct {
	RoleIDs []uint `json:"role_ids" binding:"required"`
}
