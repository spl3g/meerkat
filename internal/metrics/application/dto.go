package application

import (
	"context"

	"meerkat-v0/pkg/utils"
	metricsdomain "meerkat-v0/internal/metrics/domain"
)

// MetricInstance represents a running metric collection instance in the application layer
type MetricInstance struct {
	ID      utils.EntityID
	Metric  metricsdomain.Metric
	Config  metricsdomain.MetricConfig
	RawCfg  []byte

	ctx     context.Context
	cancel  context.CancelFunc
	running bool
}

// NewMetricInstance creates a new metric instance
func NewMetricInstance(id utils.EntityID, metric metricsdomain.Metric, cfg metricsdomain.MetricConfig, rawCfg []byte) *MetricInstance {
	return &MetricInstance{
		ID:     id,
		Metric: metric,
		Config: cfg,
		RawCfg: rawCfg,
	}
}

// ServiceInstance represents a service with its metrics
type ServiceInstance struct {
	ID        utils.EntityID
	MetricIDs []string
	RawCfg    []byte
}

// NewServiceInstance creates a new service instance
func NewServiceInstance(id utils.EntityID, rawCfg []byte, metricIDs []string) *ServiceInstance {
	return &ServiceInstance{
		ID:        id,
		MetricIDs: metricIDs,
		RawCfg:    rawCfg,
	}
}

