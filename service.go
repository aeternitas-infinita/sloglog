package sloglog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

// Logger wraps slog.Logger to provide additional functionality
type Logger struct {
	logger    *slog.Logger
	addSource bool
}

// FileLogger manages file logging with daily rotation
type FileLogger struct {
	mu      sync.RWMutex
	file    *os.File
	dir     string
	date    string
	enabled bool
}

// Global file logger instance
var fileLogger *FileLogger

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
			attrs = append(attrs, slog.String("source", fmt.Sprintf("[%s:%d]", file, line)))
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

	// Write to stdout/stderr
	l.logger.Handler().Handle(ctx, record)

	// Write to file if enabled
	if fileLogger != nil && fileLogger.enabled {
		logEntry := l.formatLogEntry(record)
		fileLogger.writeToFile(logEntry)
	}
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
		AddSource: true, // Changed to true to include source information
		Level:     level,
	}

	// Use custom handler for better formatting
	handler := NewCustomHandler(os.Stdout, opts, true)
	defaultLogger = &Logger{
		logger:    slog.New(handler),
		addSource: true,
	}

	minHandler := NewCustomHandler(os.Stdout, opts, false)
	Min = &Logger{
		logger:    slog.New(minHandler),
		addSource: false,
	}
}

func init() {
	InitLogger(slog.LevelInfo)
	initFileLogger()
}

// initFileLogger initializes the file logger
func initFileLogger() {
	logDir := os.Getenv("LOG_DIR_PATH")
	if logDir == "" {
		// Use current working directory of the importing project
		cwd, err := os.Getwd()
		if err != nil {
			logDir = "external/logs"
		} else {
			logDir = filepath.Join(cwd, "external/logs")
		}
	}

	fileLogger = &FileLogger{
		dir:     logDir,
		enabled: false,
	}
}

// EnableFileLogging enables file logging
func EnableFileLogging() {
	if fileLogger == nil {
		initFileLogger()
	}
	fileLogger.enabled = true
}

// DisableFileLogging disables file logging
func DisableFileLogging() {
	if fileLogger != nil {
		fileLogger.mu.Lock()
		defer fileLogger.mu.Unlock()
		if fileLogger.file != nil {
			fileLogger.file.Close()
			fileLogger.file = nil
		}
		fileLogger.enabled = false
	}
}

// getLogFile returns the current log file, creating a new one if needed
func (fl *FileLogger) getLogFile() (*os.File, error) {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	if !fl.enabled {
		return nil, nil
	}

	today := time.Now().Format("2006-01-02")

	if fl.file == nil || fl.date != today {
		if fl.file != nil {
			fl.file.Close()
		}

		if err := os.MkdirAll(fl.dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		filename := filepath.Join(fl.dir, fmt.Sprintf("%s.log", today))
		file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}

		fl.file = file
		fl.date = today
	}

	return fl.file, nil
}

// writeToFile writes log entry to file
func (fl *FileLogger) writeToFile(entry string) {
	if !fl.enabled {
		return
	}

	file, err := fl.getLogFile()
	if err != nil || file == nil {
		return
	}

	fl.mu.RLock()
	defer fl.mu.RUnlock()

	file.WriteString(entry + "\n")
}

// formatLogEntry formats a log record for file output
func (l *Logger) formatLogEntry(record slog.Record) string {
	var parts []string

	// Format timestamp in a more readable format
	timestamp := record.Time.Format("2006-01-02 15:04:05.000")

	// Format level with fixed width and color-like indicators
	level := formatLevel(record.Level)

	// Build the main log line
	mainLine := fmt.Sprintf("[%s] %s | %s", timestamp, level, record.Message)
	parts = append(parts, mainLine)

	// Add attributes on separate indented lines if present
	var attrs []string
	record.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, fmt.Sprintf("  ├─ %s: %s", a.Key, a.Value.String()))
		return true
	})

	if len(attrs) > 0 {
		// Change the last attribute prefix to indicate end
		if len(attrs) > 1 {
			attrs[len(attrs)-1] = strings.Replace(attrs[len(attrs)-1], "├─", "└─", 1)
		}
		parts = append(parts, attrs...)
	}

	return strings.Join(parts, "\n")
}

// formatLevel formats the log level with consistent width
func formatLevel(level slog.Level) string {
	switch level {
	case slog.LevelDebug:
		return "DEBUG"
	case slog.LevelInfo:
		return "INFO"
	case slog.LevelWarn:
		return "WARN"
	case slog.LevelError:
		return "ERROR"
	default:
		return level.String()
	}
}

// CustomHandler implements slog.Handler for better formatting
type CustomHandler struct {
	opts      slog.HandlerOptions
	writer    io.Writer
	addSource bool
}

// NewCustomHandler creates a new custom handler
func NewCustomHandler(w io.Writer, opts *slog.HandlerOptions, addSource bool) *CustomHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	return &CustomHandler{
		opts:      *opts,
		writer:    w,
		addSource: addSource,
	}
}

// Enabled reports whether the handler handles records at the given level
func (h *CustomHandler) Enabled(ctx context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.opts.Level != nil {
		minLevel = h.opts.Level.Level()
	}
	return level >= minLevel
}

// Handle handles the Record
func (h *CustomHandler) Handle(ctx context.Context, r slog.Record) error {
	// Check if we should handle this level
	if !h.Enabled(ctx, r.Level) {
		return nil
	}

	// Format timestamp with full date and timezone
	timestamp := r.Time.Format("2006-01-02 15:04:05 MST")

	// Format level with colors for console
	level := formatLevelWithColor(r.Level)

	// Build the main log line
	var parts []string
	mainLine := fmt.Sprintf("%s %s %s", timestamp, level, r.Message)

	// Add source information if enabled
	if h.addSource {
		if h.opts.AddSource {
			// Source info is already in attributes
			r.Attrs(func(a slog.Attr) bool {
				if a.Key == "source" {
					mainLine += fmt.Sprintf(" %s", a.Value.String())
					return false // Don't process this attribute again
				}
				return true
			})
		}
	}

	parts = append(parts, mainLine)

	// Add other attributes on the same line for console (more compact)
	var attrs []string
	r.Attrs(func(a slog.Attr) bool {
		if a.Key != "source" { // Skip source as it's already handled
			attrs = append(attrs, fmt.Sprintf("%s=%s", a.Key, a.Value.String()))
		}
		return true
	})

	if len(attrs) > 0 {
		parts[0] += " " + strings.Join(attrs, " ")
	}

	// Write to output
	fmt.Fprintln(h.writer, strings.Join(parts, "\n"))
	return nil
}

// WithAttrs returns a new Handler whose attributes consist of h's attributes followed by attrs
func (h *CustomHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h // Simplified implementation
}

// WithGroup returns a new Handler with the given group appended to the receiver's existing groups
func (h *CustomHandler) WithGroup(name string) slog.Handler {
	return h // Simplified implementation
}

// formatLevelWithColor formats the log level with ANSI colors for console
func formatLevelWithColor(level slog.Level) string {
	const (
		colorReset  = "\033[0m"
		colorRed    = "\033[31m"
		colorYellow = "\033[33m"
		colorBlue   = "\033[34m"
		colorGray   = "\033[37m"
	)

	switch level {
	case slog.LevelDebug:
		return colorGray + "[DEBUG]" + colorReset
	case slog.LevelInfo:
		return colorBlue + "[INFO]" + colorReset
	case slog.LevelWarn:
		return colorYellow + "[WARN]" + colorReset
	case slog.LevelError:
		return colorRed + "[ERROR]" + colorReset
	default:
		return fmt.Sprintf("[%s]", level.String())
	}
}
