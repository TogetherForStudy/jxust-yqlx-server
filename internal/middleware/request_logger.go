package middleware

import (
	"net/http"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"

	"github.com/gin-gonic/gin"
)

// Logger 结构化日志中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		hasError, _ := c.Get("response_has_error")
		bodyStatusCode := 0
		if val, exists := c.Get("response_biz_code"); exists {
			if code, ok := val.(int); ok {
				bodyStatusCode = code
			}
			if code, ok := val.(constant.ResCode); ok {
				bodyStatusCode = int(code)
			}
		}

		logFields := map[string]any{
			"action":        "http_request",
			"message":       "HTTP request processed",
			"http_status":   statusCode,
			"latency_ms":    latency.Milliseconds(),
			"latency":       latency.String(),
			"response_size": c.Writer.Size(),
		}
		if query != "" {
			logFields["query"] = query
		}
		if len(c.Errors) > 0 {
			logFields["errors"] = c.Errors.String()
		}
		if bodyStatusCode != 0 {
			logFields["biz_code"] = bodyStatusCode
		}

		shouldLogDetails := statusCode != http.StatusOK || hasError == true
		if shouldLogDetails {
			logFields["biz_message"], _ = c.Get("body_message")
		}

		switch {
		case statusCode >= 500:
			logger.ErrorGin(c, logFields)
		case statusCode >= 400:
			logger.WarnGin(c, logFields)
		case statusCode >= 300:
			logger.InfoGin(c, logFields)
		case hasError == true:
			logger.WarnGin(c, logFields)
		default:
			logger.InfoGin(c, logFields)
		}
	}
}
