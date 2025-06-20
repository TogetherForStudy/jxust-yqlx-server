package utils

import (
	"context"

	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
)

func GetRequestID(c context.Context) string {
	if reqID, ok := c.Value(constant.RequestID).(string); ok {
		return reqID
	}
	return ""
}
