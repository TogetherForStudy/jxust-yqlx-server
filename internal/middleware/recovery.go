package middleware

import (
	"fmt"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"

	"github.com/gin-gonic/gin"
)

func RecoveryJSON() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		logger.ErrorGin(c, map[string]any{
			"action":  "panic_recovered",
			"message": "request panicked",
			"panic":   fmt.Sprintf("%v", recovered),
		})
		helper.HandleErrCode(c, constant.CommonRequestPanicked)
		c.Abort()
	})
}
