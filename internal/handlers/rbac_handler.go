package handlers

import (
	"net/http"
	"strconv"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"
	"github.com/gin-gonic/gin"
)

// RBACHandler RBAC 管理相关接口
type RBACHandler struct {
	svc *services.RBACService
}

// NewRBACHandler 构造函数
func NewRBACHandler(svc *services.RBACService) *RBACHandler {
	return &RBACHandler{svc: svc}
}

// ListRoles 获取角色列表
func (h *RBACHandler) ListRoles(c *gin.Context) {
	roles, err := h.svc.ListRoles(c.Request.Context())
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.SuccessResponse(c, roles)
}

// CreateRole 创建角色
func (h *RBACHandler) CreateRole(c *gin.Context) {
	var req request.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, err.Error())
		return
	}

	role := &models.Role{
		RoleTag:     req.RoleTag,
		Name:        req.Name,
		Description: req.Description,
	}
	if err := h.svc.CreateRole(c.Request.Context(), role); err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.SuccessResponse(c, role)
}

// UpdateRole 更新角色
func (h *RBACHandler) UpdateRole(c *gin.Context) {
	id, err := parseUintParam(c.Param("id"))
	if err != nil {
		helper.ValidateResponse(c, "invalid id")
		return
	}

	var req request.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, err.Error())
		return
	}

	updates := map[string]any{
		"name": req.Name,
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if err := h.svc.UpdateRole(c.Request.Context(), id, updates); err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.SuccessResponse(c, gin.H{"id": id})
}

// DeleteRole 删除角色
func (h *RBACHandler) DeleteRole(c *gin.Context) {
	id, err := parseUintParam(c.Param("id"))
	if err != nil {
		helper.ValidateResponse(c, "invalid id")
		return
	}
	if err := h.svc.DeleteRole(c.Request.Context(), id); err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.SuccessResponse(c, gin.H{"id": id})
}

// ListPermissions 获取权限列表
func (h *RBACHandler) ListPermissions(c *gin.Context) {
	perms, err := h.svc.ListPermissions(c.Request.Context())
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.SuccessResponse(c, perms)
}

// CreatePermission 创建权限
func (h *RBACHandler) CreatePermission(c *gin.Context) {
	var req request.CreatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, err.Error())
		return
	}
	perm := &models.Permission{
		PermissionTag: req.PermissionTag,
		Name:          req.Name,
		Description:   req.Description,
	}
	if err := h.svc.CreatePermission(c.Request.Context(), perm); err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.SuccessResponse(c, perm)
}

// UpdateRolePermissions 重置角色权限
func (h *RBACHandler) UpdateRolePermissions(c *gin.Context) {
	roleID, err := parseUintParam(c.Param("id"))
	if err != nil {
		helper.ValidateResponse(c, "invalid id")
		return
	}
	var req request.UpdateRolePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, err.Error())
		return
	}
	if err := h.svc.UpdateRolePermissions(c.Request.Context(), roleID, req.PermissionIDs); err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.SuccessResponse(c, gin.H{"role_id": roleID})
}

// UpdateUserRoles 更新用户角色
func (h *RBACHandler) UpdateUserRoles(c *gin.Context) {
	userID, err := parseUintParam(c.Param("id"))
	if err != nil {
		helper.ValidateResponse(c, "invalid id")
		return
	}
	var req request.UpdateUserRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, err.Error())
		return
	}
	if err := h.svc.UpdateUserRoles(c.Request.Context(), userID, req.RoleIDs); err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.SuccessResponse(c, gin.H{"user_id": userID})
}

// GetUserPermissions 获取用户权限列表
func (h *RBACHandler) GetUserPermissions(c *gin.Context) {
	userID, err := parseUintParam(c.Param("id"))
	if err != nil {
		helper.ValidateResponse(c, "invalid id")
		return
	}
	perms, err := h.svc.GetUserPermissions(c.Request.Context(), userID)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.SuccessResponse(c, gin.H{
		"user_id":     userID,
		"permissions": perms,
	})
}

// ListRolesWithPermissions 获取所有角色及其对应的权限列表
func (h *RBACHandler) ListRolesWithPermissions(c *gin.Context) {
	roles, rolePermMap, err := h.svc.GetRolesWithPermissions(c.Request.Context())
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	var result []response.RoleWithPermissionsResponse
	for _, role := range roles {
		permissions := rolePermMap[role.ID]
		if permissions == nil {
			permissions = []models.Permission{} // 确保返回空数组而不是nil
		}
		result = append(result, response.RoleWithPermissionsResponse{
			Role:        role,
			Permissions: permissions,
		})
	}

	helper.SuccessResponse(c, response.RolesWithPermissionsResponse{
		Roles: result,
	})
}

func parseUintParam(raw string) (uint, error) {
	val, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(val), nil
}
