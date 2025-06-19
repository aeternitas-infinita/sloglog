package sloglog

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/valyala/fasthttp"
)

var Logger *slog.Logger
var LoggerMin *slog.Logger

func GetTraceID(ctx any) string {
	if ctx == nil {
		return ""
	}
	if requestCtx, ok := ctx.(*fasthttp.RequestCtx); ok {
		if v := requestCtx.UserValue("trace_id"); v != nil {
			return v.(string)
		}
	}
	return ""
}

func LogWithContext(ctx context.Context, level slog.Level, msg string, args ...any) {
	traceID := GetTraceID(ctx)
	attrs := make([]slog.Attr, 0, len(args)+1)

	if traceID != "" {
		attrs = append(attrs, slog.String("trace_id", traceID))
	}

	for i := range args {
		if attr, ok := args[i].(slog.Attr); ok {
			attrs = append(attrs, attr)
		}
	}

	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "unknown"
		line = 0
	}

	attrs = append(attrs, slog.String("source", fmt.Sprintf("[%s:%d]", file, line)))

	record := slog.NewRecord(time.Now(), level, msg, 0)
	record.AddAttrs(attrs...)
	Logger.Handler().Handle(ctx, record)
}

func InfoContext(ctx context.Context, msg string, args ...any) {
	LogWithContext(ctx, slog.LevelInfo, msg, args...)
}

func DebugContext(ctx context.Context, msg string, args ...any) {
	LogWithContext(ctx, slog.LevelDebug, msg, args...)
}

func WarnContext(ctx context.Context, msg string, args ...any) {
	LogWithContext(ctx, slog.LevelWarn, msg, args...)
}

func ErrorContext(ctx context.Context, msg string, args ...any) {
	LogWithContext(ctx, slog.LevelError, msg, args...)
}

func InitLogger(level slog.Level) {
	opts := &slog.HandlerOptions{
		AddSource: false,
		Level:     level,
	}
	handler := slog.NewTextHandler(os.Stdout, opts)
	handlerMin := slog.NewTextHandler(os.Stdout, opts)

	Logger = slog.New(handler)
	LoggerMin = slog.New(handlerMin)
}

func init() {
	InitLogger(slog.LevelInfo)
}
