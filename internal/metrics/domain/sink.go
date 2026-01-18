package domain

import "context"

// Sink defines the interface for emitting metrics samples
type Sink interface {
	Emit(ctx context.Context, sample Sample) error
}

