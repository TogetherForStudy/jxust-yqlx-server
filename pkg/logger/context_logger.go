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

// logWithLevel writes structured fields to zap and optionally sends to CLS
func logWithLevel(level constant.LogLevel, merged map[string]string, l *tencentcloudclssdkgo.Log) {
	fields := make([]any, 0, len(merged)*2)
	for k, v := range merged {
		fields = append(fields, k, v)
	}
	switch level {
	case constant.DebugLevel:
		zlog.Debugw("", fields...)
	case constant.WarnLevel:
		zlog.Warnw("", fields...)
	case constant.ErrorLevel:
		zlog.Errorw("", fields...)
	case constant.FatalLevel:
		mergeAndFatal(merged)
	case constant.PanicLevel:
		mergeAndPanic(merged)
	default:
		zlog.Infow("", fields...)
	}
	safeSendLog(l)
}

func mergeAndFatal(merged map[string]string) {
	fields := make([]any, 0, len(merged)*2)
	for k, v := range merged {
		fields = append(fields, k, v)
	}
	ShutdownLogger(5 * time.Second)
	zlog.Fatalw("", fields...)
}

func mergeAndPanic(merged map[string]string) {
	fields := make([]any, 0, len(merged)*2)
	for k, v := range merged {
		fields = append(fields, k, v)
	}
	ShutdownLogger(5 * time.Second)
	zlog.Panicw("", fields...)
}

// DebugCtx logs a debug level message with context
func DebugCtx(ctx context.Context, msg map[string]any) {
	merged := mergeContextAndMessage(ctx, msg)
	l := NewStructuredClsLogging(constant.DebugLevel, merged)
	logWithLevel(constant.DebugLevel, merged, l)
}

// InfoCtx logs an info level message with context
func InfoCtx(ctx context.Context, msg map[string]any) {
	merged := mergeContextAndMessage(ctx, msg)
	l := NewStructuredClsLogging(constant.InfoLevel, merged)
	logWithLevel(constant.InfoLevel, merged, l)
}

// WarnCtx logs a warning level message with context
func WarnCtx(ctx context.Context, msg map[string]any) {
	merged := mergeContextAndMessage(ctx, msg)
	l := NewStructuredClsLogging(constant.WarnLevel, merged)
	logWithLevel(constant.WarnLevel, merged, l)
}

// ErrorCtx logs an error level message with context
func ErrorCtx(ctx context.Context, msg map[string]any) {
	merged := mergeContextAndMessage(ctx, msg)
	l := NewStructuredClsLogging(constant.ErrorLevel, merged)
	logWithLevel(constant.ErrorLevel, merged, l)
}

// FatalCtx logs a fatal level message with context
func FatalCtx(ctx context.Context, msg map[string]any) {
	merged := mergeContextAndMessage(ctx, msg)
	l := NewStructuredClsLogging(constant.FatalLevel, merged)
	safeSendLog(l)
	mergeAndFatal(merged)
}

// PanicCtx logs a panic level message with context
func PanicCtx(ctx context.Context, msg map[string]any) {
	merged := mergeContextAndMessage(ctx, msg)
	l := NewStructuredClsLogging(constant.PanicLevel, merged)
	safeSendLog(l)
	mergeAndPanic(merged)
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
