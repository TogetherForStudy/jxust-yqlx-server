package helper

import (
	"net/http"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"

	"github.com/gin-gonic/gin"
)

// SuccessResponse 成功响应
func SuccessResponse(c *gin.Context, data any) {
	c.JSON(http.StatusOK, dto.Response{
		RequestId:     GetRequestID(c),
		StatusMessage: "Success",
		Result:        data,
	})
}

// ErrorResponse 错误响应(服务失败)
func ErrorResponse(c *gin.Context, serviceCode int, message string) {
	logger.Errorf("Error response: %v, message: %v", message, http.StatusText(serviceCode))
	c.JSON(http.StatusOK, dto.Response{
		RequestId:     GetRequestID(c),
		StatusCode:    serviceCode,
		StatusMessage: message,
	})
}

// ValidateResponse 验证失败响应(400 Bad Request)
func ValidateResponse(c *gin.Context, message string) {
	logger.Warnf("Validation error: %v", message)
	c.JSON(http.StatusBadRequest, dto.Response{
		RequestId:     GetRequestID(c),
		StatusCode:    http.StatusBadRequest,
		StatusMessage: message,
	})
}

// PageSuccessResponse 分页成功响应
func PageSuccessResponse(c *gin.Context, data any, total int64, page, size int) {
	SuccessResponse(c, response.PageResponse{
		Data:  data,
		Total: total,
		Page:  page,
		Size:  size,
	})
}
