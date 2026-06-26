package middleware

import (
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"

	"github.com/gin-gonic/gin"
)

// RequireFeature 功能权限中间件
// 检查用户是否有指定功能的访问权限
func RequireFeature(featureService *services.FeatureService, featureKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户ID
		userID := helper.GetUserID(c)
		if userID == 0 {
			helper.HandleErrCode(c, constant.AuthMissingUserContext)
			c.Abort()
			return
		}

		// 检查用户是否有该功能权限
		hasFeature, err := featureService.CheckUserFeature(c.Request.Context(), userID, featureKey)
		if err != nil {
			helper.HandleError(c, err)
			c.Abort()
			return
		}

		if !hasFeature {
			helper.HandleErrCode(c, constant.CommonForbidden)
			c.Abort()
			return
		}

		c.Next()
	}
}
