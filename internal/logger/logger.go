package logger

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
)

const (
	formatJson = "json"
	formatText = "text"

	levelDebug = "debug"
	levelInfo  = "info"
	levelWarn  = "warn"
	levelError = "error"
)

func NewLogger(output io.Writer, level string, format string) (*slog.Logger, error) {
	slogLevel, err := parseLevel(level)
	if err != nil {
		return nil, err
	}

	options := &slog.HandlerOptions{
		Level: slogLevel,
	}

	switch normalize(format) {
	case formatJson:
		return slog.New(slog.NewJSONHandler(output, options)), nil
	case formatText:
		return slog.New(slog.NewTextHandler(output, options)), nil
	default:
		return nil, fmt.Errorf("unsupported log format: %s", format)
	}
}

func parseLevel(level string) (slog.Level, error) {
	switch normalize(level) {
	case levelDebug:
		return slog.LevelDebug, nil
	case levelInfo:
		return slog.LevelInfo, nil
	case levelWarn:
		return slog.LevelWarn, nil
	case levelError:
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unsupported log level: %s", level)
	}
}

func normalize(level string) string {
	return strings.ToLower(strings.TrimSpace(level))
}
