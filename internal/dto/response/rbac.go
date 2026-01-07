package response

import "github.com/TogetherForStudy/jxust-yqlx-server/internal/models"

// RoleWithPermissionsResponse 角色及其权限列表响应
type RoleWithPermissionsResponse struct {
	Role        models.Role         `json:"role"`
	Permissions []models.Permission `json:"permissions"`
}

// RolesWithPermissionsResponse 所有角色及其权限列表响应
type RolesWithPermissionsResponse struct {
	Roles []RoleWithPermissionsResponse `json:"roles"`
}

// RoleWithUsersResponse 角色及其用户信息响应
type RoleWithUsersResponse struct {
	Role      models.Role `json:"role"`
	UserCount int         `json:"user_count"` // 拥有该角色的用户数量
	UserIDs   []uint      `json:"user_ids"`   // 拥有该角色的用户ID列表（basic_user除外）
}
