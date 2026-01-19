package domain

import "context"

// SystemMetricsReader defines the interface for reading system metrics
// This interface abstracts file I/O and system-level operations from the domain layer
type SystemMetricsReader interface {
	// ReadLoadAvg reads the 1-minute load average from the system
	// Returns the load average value as a float64
	ReadLoadAvg(ctx context.Context) (float64, error)
}

