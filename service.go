package sloglog

import (
	"context"
	"fmt"
	"log/slog"
	"os"
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

	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			if k, ok := args[i].(string); ok {
				attrs = append(attrs, slog.Any(k, args[i+1]))
			}
		}
	}

	record := slog.NewRecord(time.Now(), level, msg, 2)
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
	replace := func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.SourceKey {
			source := a.Value.Any().(*slog.Source)
			source.File = fmt.Sprintf("[%s:%d]", source.File, source.Line)
		}
		return a
	}

	opts := &slog.HandlerOptions{
		AddSource:   true,
		Level:       level,
		ReplaceAttr: replace,
	}
	handler := slog.NewTextHandler(os.Stdout, opts)

	optsMin := &slog.HandlerOptions{
		AddSource: false,
		Level:     level,
	}
	handlerMin := slog.NewTextHandler(os.Stdout, optsMin)

	Logger = slog.New(handler)
	LoggerMin = slog.New(handlerMin)
}

func init() {
	InitLogger(slog.LevelInfo)
}
