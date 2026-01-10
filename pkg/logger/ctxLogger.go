package logger

import (
	"context"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	tencentcloudclssdkgo "github.com/tencentcloud/tencentcloud-cls-sdk-go"
)

// NewStructuredClsLogging creates a new CLS log with structured fields
func NewStructuredClsLogging(level constant.LogLevel, msg map[string]string) *tencentcloudclssdkgo.Log {
	msg["level"] = string(level)
	return tencentcloudclssdkgo.NewCLSLog(time.Now().Unix(), msg)
}

// mergeContextAndMessage merges context fields and custom message fields
func mergeContextAndMessage(ctx context.Context, msg map[string]any) map[string]string {
	m1 := parserCtxToMapStringString(ctx)
	m2 := parserMapStringAnyToMapStringString(msg)
	for k, v := range m2 {
		m1[k] = v
	}
	return m1
}

// DebugCtx logs a debug level message with context
func DebugCtx(ctx context.Context, msg map[string]any) {
	merged := mergeContextAndMessage(ctx, msg)
	l := NewStructuredClsLogging(constant.DebugLevel, merged)
	Debugln(l.String())
	logChannel <- l
}

// InfoCtx logs an info level message with context
func InfoCtx(ctx context.Context, msg map[string]any) {
	merged := mergeContextAndMessage(ctx, msg)
	l := NewStructuredClsLogging(constant.InfoLevel, merged)
	logChannel <- l
	Infoln(l.String())
}

// WarnCtx logs a warning level message with context
func WarnCtx(ctx context.Context, msg map[string]any) {
	merged := mergeContextAndMessage(ctx, msg)
	l := NewStructuredClsLogging(constant.WarnLevel, merged)
	logChannel <- l
	Warnln(l.String())
}

// ErrorCtx logs an error level message with context
func ErrorCtx(ctx context.Context, msg map[string]any) {
	merged := mergeContextAndMessage(ctx, msg)
	l := NewStructuredClsLogging(constant.ErrorLevel, merged)
	logChannel <- l
	Errorln(l.String())
}

// FatalCtx logs a fatal level message with context
func FatalCtx(ctx context.Context, msg map[string]any) {
	merged := mergeContextAndMessage(ctx, msg)
	l := NewStructuredClsLogging(constant.FatalLevel, merged)
	logChannel <- l
	time.Sleep(time.Second) // Ensure the log is sent before exiting
	Fatalln(l.String())
}

// PanicCtx logs a panic level message with context
func PanicCtx(ctx context.Context, msg map[string]any) {
	merged := mergeContextAndMessage(ctx, msg)
	l := NewStructuredClsLogging(constant.PanicLevel, merged)
	logChannel <- l
	time.Sleep(time.Second) // Ensure the log is sent before exiting
	panic(l.String())
}
