package logger

import (
	"log/slog"
	"os"
)

type Config struct {
	Level  string // debug, info, warn, error
	Format string // json, text
}

// New creates a new structured logger
func New(cfg Config) *slog.Logger {
	level := parseLevel(cfg.Level)

	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level: level,
	}

	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// WithFields adds fields to the logger
func WithFields(logger *slog.Logger, fields map[string]interface{}) *slog.Logger {
	args := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return logger.With(args...)
}
