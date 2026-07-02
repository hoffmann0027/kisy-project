// Package logger configures the process-wide structured logger.
package logger

import (
	"log/slog"
	"os"
)

// New returns a JSON structured logger writing to stdout, suitable for
// collection by a log aggregator in production.
func New(level string) *slog.Logger {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		lvl = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: lvl,
	})

	return slog.New(handler)
}
