package domain

import "context"

// Repository defines the interface for metrics persistence
type Repository interface {
	InsertSample(ctx context.Context, sample Sample) error
}

