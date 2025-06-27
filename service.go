package sloglog

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

// Logger wraps slog.Logger to provide additional functionality
type Logger struct {
	logger    *slog.Logger
	addSource bool
}

// Package-level loggers
var (
	defaultLogger *Logger
	Min           *Logger
)

// TraceIDKey is the key used to store trace IDs in context
const TraceIDKey = "trace_id"

// TraceIDToFHCtx adds a new trace ID to fasthttp context
func TraceIDToFHCtx(ctx *fasthttp.RequestCtx) {
	ctx.SetUserValue(TraceIDKey, uuid.New().String())
}

// CtxWithTraceID creates a new context with timeout and trace ID
func CtxWithTraceID(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	return context.WithValue(ctx, TraceIDKey, uuid.New().String()), cancel
}

// GetTraceID extracts trace ID from context
func GetTraceID(ctx any) string {
	if ctx == nil {
		return ""
	}

	// Try to get trace ID from fasthttp.RequestCtx
	if requestCtx, ok := ctx.(*fasthttp.RequestCtx); ok {
		if v := requestCtx.UserValue(TraceIDKey); v != nil {
			return v.(string)
		}
		return ""
	}

	// Try to get trace ID from context.Context
	if stdCtx, ok := ctx.(context.Context); ok {
		if v := stdCtx.Value(TraceIDKey); v != nil {
			if traceID, ok := v.(string); ok {
				return traceID
			}
		}
	}

	return ""
}

// log implements the core logging functionality
func (l *Logger) log(ctx context.Context, callerSkip int, level slog.Level, msg string, args ...any) {
	attrs := make([]slog.Attr, 0, len(args)+1)

	// Add trace ID if available
	if ctx != nil {
		traceID := GetTraceID(ctx)
		if traceID != "" {
			attrs = append(attrs, slog.String("trace_id", traceID))
		}
	}

	// Add source information if enabled
	if l.addSource {
		_, file, line, ok := runtime.Caller(callerSkip)
		if ok {
			attrs = append(attrs, slog.String("source", fmt.Sprintf("%s:%d", file, line)))
		}
	}

	// Add additional attributes
	for i := range args {
		if attr, ok := args[i].(slog.Attr); ok {
			attrs = append(attrs, attr)
		}
	}

	record := slog.NewRecord(time.Now(), level, msg, 0)
	record.AddAttrs(attrs...)
	l.logger.Handler().Handle(ctx, record)
}

// Debug logs at debug level without context
func (l *Logger) Debug(msg string, args ...any) {
	l.log(context.Background(), 3, slog.LevelDebug, msg, args...)
}

// Info logs at info level without context
func (l *Logger) Info(msg string, args ...any) {
	l.log(context.Background(), 3, slog.LevelInfo, msg, args...)
}

// Warn logs at warn level without context
func (l *Logger) Warn(msg string, args ...any) {
	l.log(context.Background(), 3, slog.LevelWarn, msg, args...)
}

// Error logs at error level without context
func (l *Logger) Error(msg string, args ...any) {
	l.log(context.Background(), 3, slog.LevelError, msg, args...)
}

// DebugCtx logs at debug level with context
func (l *Logger) DebugCtx(ctx context.Context, msg string, args ...any) {
	l.log(ctx, 3, slog.LevelDebug, msg, args...)
}

// InfoCtx logs at info level with context
func (l *Logger) InfoCtx(ctx context.Context, msg string, args ...any) {
	l.log(ctx, 3, slog.LevelInfo, msg, args...)
}

// WarnCtx logs at warn level with context
func (l *Logger) WarnCtx(ctx context.Context, msg string, args ...any) {
	l.log(ctx, 3, slog.LevelWarn, msg, args...)
}

// ErrorCtx logs at error level with context
func (l *Logger) ErrorCtx(ctx context.Context, msg string, args ...any) {
	l.log(ctx, 3, slog.LevelError, msg, args...)
}

// Package-level convenience functions that use the default logger

// Debug logs at debug level without context
func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

// Info logs at info level without context
func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

// Warn logs at warn level without context
func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

// Error logs at error level without context
func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

// DebugCtx logs at debug level with context
func DebugCtx(ctx context.Context, msg string, args ...any) {
	defaultLogger.DebugCtx(ctx, msg, args...)
}

// InfoCtx logs at info level with context
func InfoCtx(ctx context.Context, msg string, args ...any) {
	defaultLogger.InfoCtx(ctx, msg, args...)
}

// WarnCtx logs at warn level with context
func WarnCtx(ctx context.Context, msg string, args ...any) {
	defaultLogger.WarnCtx(ctx, msg, args...)
}

// ErrorCtx logs at error level with context
func ErrorCtx(ctx context.Context, msg string, args ...any) {
	defaultLogger.ErrorCtx(ctx, msg, args...)
}

// InitLogger initializes the loggers with the specified level
func InitLogger(level slog.Level) {
	opts := &slog.HandlerOptions{
		AddSource: false,
		Level:     level,
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	defaultLogger = &Logger{
		logger:    slog.New(handler),
		addSource: true,
	}

	Min = &Logger{
		logger:    slog.New(handler),
		addSource: false,
	}
}

func init() {
	InitLogger(slog.LevelInfo)
}
