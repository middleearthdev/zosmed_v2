// Package log provides a thin slog wrapper for Zosmed services.
// Default output is JSON to stderr at Info level.
package log

import (
	"context"
	"log/slog"
	"os"
)

// Logger is the process-wide structured logger.
// Services that need a different logger should use WithContext/FromContext.
var Logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
	Level: slog.LevelInfo,
}))

type loggerKey struct{}

// WithContext stores l in ctx and returns the new context.
func WithContext(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, l)
}

// FromContext returns the logger stored in ctx.
// Falls back to the package-level Logger if none was stored.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok && l != nil {
		return l
	}
	return Logger
}
