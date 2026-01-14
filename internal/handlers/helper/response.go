package helper

import (
	"fmt"
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
	// 设置标记，通知Logger中间件需要详细日志
	c.Set("response_has_error", true)
	c.Set("response_status_code", serviceCode)

	logger.Errorf("Error response: %v, message: %v", message, http.StatusText(serviceCode))

	response := dto.Response{
		RequestId:     GetRequestID(c),
		StatusCode:    serviceCode,
		StatusMessage: message,
	}

	// 保存响应体供Logger使用
	c.Set("body_message", message)

	c.JSON(http.StatusOK, response)
}

// ValidateResponse 验证失败响应(400 Bad Request)
func ValidateResponse(c *gin.Context, message string) {
	// 设置标记，通知Logger中间件需要详细日志
	c.Set("response_has_error", true)
	c.Set("response_status_code", http.StatusBadRequest)

	logger.Warnf("Validation error: %v", message)

	response := dto.Response{
		RequestId:     GetRequestID(c),
		StatusCode:    http.StatusBadRequest,
		StatusMessage: message,
	}

	// 保存响应体供Logger使用
	c.Set("body_message", message)

	c.JSON(http.StatusBadRequest, response)
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

// formatResponseBody 格式化响应体为字符串
func formatResponseBody(response dto.Response) string {
	return fmt.Sprintf(`{"StatusCode":%d,"StatusMessage":"%s","RequestId":"%s"}`,
		response.StatusCode, response.StatusMessage, response.RequestId)
}
