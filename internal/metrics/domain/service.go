package domain

import (
	"context"
	"encoding/json"

	"meerkat-v0/pkg/utils"
)

// Service defines the interface for metrics service operations
// This interface allows bounded contexts to depend on abstractions rather than concrete implementations
type Service interface {
	// LoadService loads metrics for a service
	LoadService(ctx context.Context, serviceID utils.EntityID, rawConfigs []json.RawMessage) error
	// Stop stops all metrics collection
	Stop(ctx context.Context) error
}


