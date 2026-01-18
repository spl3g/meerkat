package domain

import (
	"context"
	"time"
)

// HeartbeatFilters contains optional filters for querying heartbeats
type HeartbeatFilters struct {
	EntityID   *string
	From       *time.Time
	To         *time.Time
	Successful *bool
	Limit      int
	Offset     int
}

// Repository defines the interface for monitor persistence
type Repository interface {
	InsertHeartbeat(ctx context.Context, heartbeat Heartbeat) error
	ListHeartbeats(ctx context.Context, filters HeartbeatFilters) ([]Heartbeat, error)
}

