package sloglog

import (
	"context"
	"testing"
	"time"
)

func TestTimestampFormat(t *testing.T) {
	// Test the new timestamp format
	Info("Testing new timestamp format with full date and timezone")
	Warn("Warning message with full timestamp")

	ctx, cancel := CtxWithTraceID(context.Background(), time.Second*5)
	defer cancel()

	InfoCtx(ctx, "Context message with trace ID and full timestamp")
}
