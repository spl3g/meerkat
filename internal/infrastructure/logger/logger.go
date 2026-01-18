package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

type Logger struct {
	*slog.Logger
}

type loggerKeyType struct{}

var loggerKey loggerKeyType = struct{}{}

// DefaultLogger creates a logger using slog.Default()
func DefaultLogger() *Logger {
	return &Logger{
		Logger: slog.Default(),
	}
}

// NewLogger creates a configured logger based on environment variables:
// - MEERKAT_LOG_LEVEL: DEBUG, INFO, WARN, ERROR (default: INFO)
// - MEERKAT_LOG_FORMAT: json or text (default: text)
// - MEERKAT_LOG_OUTPUT: stdout, stderr, or file path (default: stdout)
func NewLogger() *Logger {
	level := parseLogLevel(os.Getenv("MEERKAT_LOG_LEVEL"))
	format := strings.ToLower(os.Getenv("MEERKAT_LOG_FORMAT"))
	output := os.Getenv("MEERKAT_LOG_OUTPUT")

	// Default to text format if not specified
	if format == "" {
		format = "text"
	}

	// Default to stdout if not specified
	if output == "" {
		output = "stdout"
	}

	// Get output writer
	var writer io.Writer
	switch output {
	case "stdout":
		writer = os.Stdout
	case "stderr":
		writer = os.Stderr
	default:
		// File path
		file, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			// Fallback to stdout if file can't be opened
			writer = os.Stdout
		} else {
			writer = file
		}
	}

	// Create handler based on format
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: level,
	}

	if format == "json" {
		handler = slog.NewJSONHandler(writer, opts)
	} else {
		handler = slog.NewTextHandler(writer, opts)
	}

	return &Logger{
		Logger: slog.New(handler),
	}
}

// parseLogLevel parses log level from string
func parseLogLevel(levelStr string) slog.Level {
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// SetDefaultLogger sets the logger as the default slog logger
func SetDefaultLogger(l *Logger) {
	slog.SetDefault(l.Logger)
}

