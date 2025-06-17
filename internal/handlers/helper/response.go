package helper

import (
	"net/http"

	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"
	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// SuccessResponse 成功响应
func SuccessResponse(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    data,
	})
}

// ErrorResponse 错误响应
func ErrorResponse(c *gin.Context, httpCode int, message string) {
	logger.Errorf("Error response: %v, message: %v", message, http.StatusText(httpCode))
	c.JSON(httpCode, Response{
		Code:    httpCode,
		Message: message,
	})
}

// ValidateResponse 验证失败响应
func ValidateResponse(c *gin.Context, message string) {
	logger.Warnf("Validation error: %v", message)
	c.JSON(http.StatusBadRequest, Response{
		Code:    http.StatusBadRequest,
		Message: message,
	})
}

// PageResponse 分页响应
type PageResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
	Total   int64  `json:"total"`
	Page    int    `json:"page"`
	Size    int    `json:"size"`
}

// PageSuccessResponse 分页成功响应
func PageSuccessResponse(c *gin.Context, data any, total int64, page, size int) {
	c.JSON(http.StatusOK, PageResponse{
		Code:    200,
		Message: "success",
		Data:    data,
		Total:   total,
		Page:    page,
		Size:    size,
	})
}
