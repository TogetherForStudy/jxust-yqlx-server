package helper

import (
	"net/http"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/apperr"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"

	"github.com/gin-gonic/gin"
)

// SuccessResponse 成功响应
func SuccessResponse(c *gin.Context, data any) {
	statusMessage := constant.DefaultErrorMessage(constant.SuccessCode)
	setResponseMetadata(c, http.StatusOK, constant.SuccessCode, statusMessage, false)
	c.JSON(http.StatusOK, dto.Response{
		RequestId:     GetRequestID(c),
		StatusCode:    int(constant.SuccessCode),
		StatusMessage: statusMessage,
		Result:        data,
	})
}

// HandleErrCode 将业务错误码统一映射为对外响应。
func HandleErrCode(c *gin.Context, code constant.ResCode) {
	HandleError(c, apperr.FromCode(code))
}

// HandleError 将内部错误统一映射为对外响应。
func HandleError(c *gin.Context, err error) {
	appErr := apperr.FromError(err)
	if appErr == nil {
		appErr = apperr.FromCode(constant.CommonInternal)
	}

	setResponseMetadata(c, appErr.HTTPStatus, appErr.Code, appErr.Message, true)

	logFields := map[string]any{
		"action":      "http_error_response",
		"message":     appErr.Message,
		"http_status": appErr.HTTPStatus,
		"biz_code":    appErr.Code,
	}
	if appErr.Cause != nil {
		logFields["error"] = appErr.Cause.Error()
	}

	switch {
	case appErr.HTTPStatus >= http.StatusInternalServerError:
		logger.ErrorGin(c, logFields)
	case appErr.HTTPStatus >= http.StatusBadRequest:
		logger.WarnGin(c, logFields)
	default:
		logger.InfoGin(c, logFields)
	}

	c.JSON(appErr.HTTPStatus, dto.Response{
		RequestId:     GetRequestID(c),
		StatusCode:    int(appErr.Code),
		StatusMessage: appErr.Message,
	})
}

// GetResponseBizCode 读取当前响应上下文中的业务码
func GetResponseBizCode(c *gin.Context) constant.ResCode {
	if c == nil {
		return constant.SuccessCode
	}
	if val, exists := c.Get("response_biz_code"); exists {
		switch code := val.(type) {
		case constant.ResCode:
			return code
		case int:
			return constant.ResCode(code)
		}
	}
	return constant.SuccessCode
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

func setResponseMetadata(c *gin.Context, httpStatus int, code constant.ResCode, message string, hasError bool) {
	c.Set("response_has_error", hasError)
	c.Set("response_http_status", httpStatus)
	c.Set("response_status_code", int(code))
	c.Set("response_biz_code", code)
	c.Set("body_message", message)
}
