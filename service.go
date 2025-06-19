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

	for i := 0; i < len(args); i++ {
		if attr, ok := args[i].(slog.Attr); ok {
			attrs = append(attrs, attr)
		}
	}

	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "???"
		line = 0
	}

	record := slog.NewRecord(time.Now(), level, msg, 0)
	record.AddAttrs(attrs...)

	record.AddAttrs(slog.Group("source",
		slog.String("file", file),
		slog.Int("line", line),
	))

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
		if a.Key == slog.SourceKey || (len(groups) > 0 && groups[0] == "source") {
			if source, ok := a.Value.Any().(*slog.Source); ok {
				return slog.String("source", fmt.Sprintf("[%s:%d]", source.File, source.Line))
			}
			if len(groups) > 0 && groups[0] == "source" {
				switch a.Key {
				case "file":
					file := a.Value.String()
					line := 0
					// Find the line number from attrs
					for _, g := range groups {
						if g == "line" {
							line = int(a.Value.Int64())
							break
						}
					}
					return slog.String("source", fmt.Sprintf("[%s:%d]", file, line))
				}
			}
		}
		return a
	}

	opts := &slog.HandlerOptions{
		AddSource:   false,
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
