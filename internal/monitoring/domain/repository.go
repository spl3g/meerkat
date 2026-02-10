package domain

import "context"

// Repository defines the interface for monitor persistence
type Repository interface {
	InsertHeartbeat(ctx context.Context, heartbeat Heartbeat) error
}

