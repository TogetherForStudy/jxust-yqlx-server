package middleware

import (
	"net/http"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/utils"
	"github.com/gin-gonic/gin"
)

// RequirePermission 校验用户是否拥有指定权限（任一满足即可）
// 同时将权限信息注入到 context 中，供服务层使用
func RequirePermission(rbac *services.RBACService, permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDVal, exists := c.Get("user_id")
		if !exists {
			helper.ErrorResponse(c, http.StatusUnauthorized, "未获取到用户信息")
			c.Abort()
			return
		}
		userID, ok := userIDVal.(uint)
		if !ok {
			helper.ErrorResponse(c, http.StatusUnauthorized, "用户信息异常")
			c.Abort()
			return
		}

		// 获取用户权限快照
		snap, err := rbac.GetUserPermissionSnapshot(c.Request.Context(), userID)
		if err != nil {
			helper.ErrorResponse(c, http.StatusInternalServerError, "权限校验失败")
			c.Abort()
			return
		}

		// 将权限信息注入到 context
		ctx := utils.WithUserRoles(c.Request.Context(), snap.RoleTags)
		ctx = utils.WithUserPermissions(ctx, snap.PermissionTags)
		ctx = utils.WithIsAdmin(ctx, snap.IsAdmin)
		c.Request = c.Request.WithContext(ctx)

		// 检查权限
		if snap.IsAdmin {
			c.Next()
			return
		}

		for _, p := range permissions {
			for _, userPerm := range snap.PermissionTags {
				if userPerm == p {
					c.Next()
					return
				}
			}
		}

		helper.ErrorResponse(c, http.StatusForbidden, "权限不足")
		c.Abort()
	}
}
