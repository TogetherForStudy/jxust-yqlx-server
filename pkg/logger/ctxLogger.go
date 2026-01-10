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
	zlog.Debugln(l.String())
	safeSendLog(l)
}

// InfoCtx logs an info level message with context
func InfoCtx(ctx context.Context, msg map[string]any) {
	merged := mergeContextAndMessage(ctx, msg)
	l := NewStructuredClsLogging(constant.InfoLevel, merged)
	zlog.Infoln(l.String())
	safeSendLog(l)
}

// WarnCtx logs a warning level message with context
func WarnCtx(ctx context.Context, msg map[string]any) {
	merged := mergeContextAndMessage(ctx, msg)
	l := NewStructuredClsLogging(constant.WarnLevel, merged)
	zlog.Warnln(l.String())
	safeSendLog(l)
}

// ErrorCtx logs an error level message with context
func ErrorCtx(ctx context.Context, msg map[string]any) {
	merged := mergeContextAndMessage(ctx, msg)
	l := NewStructuredClsLogging(constant.ErrorLevel, merged)
	zlog.Errorln(l.String())
	safeSendLog(l)
}

// FatalCtx logs a fatal level message with context
func FatalCtx(ctx context.Context, msg map[string]any) {
	merged := mergeContextAndMessage(ctx, msg)
	l := NewStructuredClsLogging(constant.FatalLevel, merged)
	safeSendLog(l)
	// Wait for log to be sent before exiting
	ShutdownLogger(5 * time.Second)
	zlog.Fatalln(l.String())
}

// PanicCtx logs a panic level message with context
func PanicCtx(ctx context.Context, msg map[string]any) {
	merged := mergeContextAndMessage(ctx, msg)
	l := NewStructuredClsLogging(constant.PanicLevel, merged)
	safeSendLog(l)
	// Wait for log to be sent before panicking
	ShutdownLogger(5 * time.Second)
	zlog.Errorln(l.String()) // Log to zap before panicking
	panic(l.String())
}

// safeSendLog safely sends log to channel without blocking
func safeSendLog(log *tencentcloudclssdkgo.Log) {
	if !clsEnabled {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			// Channel is closed, ignore
		}
	}()

	select {
	case logChannel <- log:
		// Log sent successfully
	default:
		// Channel is full or closed, log dropped
		zlog.Warnf("Log channel full, log dropped")
	}
}
