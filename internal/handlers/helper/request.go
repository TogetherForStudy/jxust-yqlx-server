package helper

import (
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/utils"

	"github.com/gin-gonic/gin"
)

func GetRequestID(c *gin.Context) string {
	return utils.GetRequestID(c)
}
