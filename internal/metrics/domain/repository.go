package domain

import (
	"context"
	"time"
)

// SampleFilters contains optional filters for querying metrics samples
type SampleFilters struct {
	EntityID *string
	From     *time.Time
	To       *time.Time
	Name     *string
	Type     *MetricType
	Limit    int
	Offset   int
}

// Repository defines the interface for metrics persistence
type Repository interface {
	InsertSample(ctx context.Context, sample Sample) error
	ListSamples(ctx context.Context, filters SampleFilters) ([]Sample, error)
}

