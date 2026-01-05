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
