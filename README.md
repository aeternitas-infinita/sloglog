# sloglog

A structured logging library for Go that extends the standard `log/slog` package with additional functionality including file logging with daily rotation.

## Features

- **Structured logging** with `log/slog` 
- **Trace ID support** for request tracing
- **File logging** with daily rotation
- **Context-aware logging** 
- **Source code location** tracking
- **FastHTTP integration**

## Installation

```bash
go get github.com/aeternitas-infinita/sloglog
```

## Basic Usage

### Console Logging

```go
package main

import (
    "log/slog"
    "github.com/aeternitas-infinita/sloglog"
)

func main() {
    // Initialize logger with desired level
    sloglog.InitLogger(slog.LevelDebug)
    
    // Simple logging
    sloglog.Info("Application started")
    sloglog.Warn("This is a warning")
    sloglog.Error("This is an error")
    sloglog.Debug("Debug message")
}
```

### Context Logging with Trace ID

```go
package main

import (
    "context"
    "time"
    "github.com/aeternitas-infinita/sloglog"
)

func main() {
    // Create context with trace ID and timeout
    ctx, cancel := sloglog.CtxWithTraceID(context.Background(), 30*time.Second)
    defer cancel()
    
    // Log with context (includes trace ID)
    sloglog.InfoCtx(ctx, "Processing request")
    sloglog.DebugCtx(ctx, "Processing step completed")
}
```

### FastHTTP Integration

```go
package main

import (
    "github.com/valyala/fasthttp"
    "github.com/aeternitas-infinita/sloglog"
)

func handler(ctx *fasthttp.RequestCtx) {
    // Add trace ID to fasthttp context
    sloglog.TraceIDToFHCtx(ctx)
    
    // Log with trace ID
    sloglog.InfoCtx(ctx, "Handling request")
}
```

## File Logging

The library supports file logging with daily rotation. Log files are created with the format `YYYY-MM-DD.log` and automatically rotated every 24 hours.

### Enable File Logging

```go
package main

import (
    "github.com/aeternitas-infinita/sloglog"
)

func main() {
    // Enable file logging
    sloglog.EnableFileLogging()
    
    // Log messages (will go to both console and file)
    sloglog.Info("Application started")
    sloglog.Warn("This is a warning")
    
    // Disable file logging when done
    defer sloglog.DisableFileLogging()
}
```

### Configuration

File logging can be configured using environment variables:

- `LOG_DIR_PATH`: Directory where log files will be stored (default: `{PROJECT_DIR}/external/logs`)

```bash
# Set custom log directory
export LOG_DIR_PATH="/var/log/myapp"
```

### File Logging Behavior

- **Daily Rotation**: New log files are created each day with the format `YYYY-MM-DD.log`
- **Automatic Directory Creation**: The log directory is created automatically if it doesn't exist
- **Project-Relative Path**: By default, logs are stored in `external/logs` directory relative to the importing project's working directory
- **Same Format**: File logs use the same format as console logs
- **Thread-Safe**: File operations are protected with mutex for concurrent access
- **Disabled by Default**: File logging is disabled by default and must be explicitly enabled

### Example Log File Output

```
time=2025-07-07T10:30:45.123Z level=INFO msg="Application started" source=[/path/to/main.go:15]
time=2025-07-07T10:30:45.124Z level=WARN msg="This is a warning" source=[/path/to/main.go:16]
time=2025-07-07T10:30:45.125Z level=INFO msg="Processing request" trace_id=550e8400-e29b-41d4-a716-446655440000 source=[/path/to/handler.go:25]
```

## Log Levels

The library supports standard slog levels:
- `slog.LevelDebug`
- `slog.LevelInfo`
- `slog.LevelWarn`
- `slog.LevelError`

## API Reference

### Package Functions

- `InitLogger(level slog.Level)` - Initialize the logger with specified level
- `EnableFileLogging()` - Enable file logging
- `DisableFileLogging()` - Disable file logging
- `Debug(msg string, args ...any)` - Log debug message
- `Info(msg string, args ...any)` - Log info message
- `Warn(msg string, args ...any)` - Log warning message
- `Error(msg string, args ...any)` - Log error message
- `DebugCtx(ctx context.Context, msg string, args ...any)` - Log debug with context
- `InfoCtx(ctx context.Context, msg string, args ...any)` - Log info with context
- `WarnCtx(ctx context.Context, msg string, args ...any)` - Log warning with context
- `ErrorCtx(ctx context.Context, msg string, args ...any)` - Log error with context

### Context Functions

- `CtxWithTraceID(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc)` - Create context with trace ID
- `TraceIDToFHCtx(ctx *fasthttp.RequestCtx)` - Add trace ID to fasthttp context
- `GetTraceID(ctx any) string` - Extract trace ID from context
