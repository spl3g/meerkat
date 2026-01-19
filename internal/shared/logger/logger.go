package logger

// Logger defines the interface for logging operations
// This interface abstracts logging concerns from the application layer
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}


