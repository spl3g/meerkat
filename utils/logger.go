package utils

import (
	"log/slog"
)

type Logger struct {
	*slog.Logger
}

type loggerKeyType struct{}

var loggerKey loggerKeyType = struct{}{}

func DefaultLogger() *Logger {
	return &Logger{
		Logger: slog.Default(),
	}
}

func SetDefaultLogger(l *Logger) {
	slog.SetDefault(l.Logger)
}
