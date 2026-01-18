package domain

import (
	"time"

	"meerkat-v0/pkg/utils"
)

// MetricType represents the type of metric
type MetricType string

const (
	MetricGauge     MetricType = "gauge"
	MetricCounter   MetricType = "counter"
	MetricHistogram MetricType = "histogram"
)

// Sample represents a metrics sample value object
type Sample struct {
	ID        utils.EntityID
	Timestamp time.Time
	Type      MetricType
	Name      string
	Value     float64
	Labels    map[string]string
}

// NewSample creates a new metrics sample
func NewSample(id utils.EntityID, timestamp time.Time, metricType MetricType, name string, value float64, labels map[string]string) Sample {
	return Sample{
		ID:        id,
		Timestamp: timestamp,
		Type:      metricType,
		Name:      name,
		Value:     value,
		Labels:    labels,
	}
}

