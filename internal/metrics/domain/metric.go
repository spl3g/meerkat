package domain

import (
	"context"
	"time"

	"meerkat-v0/pkg/utils"
)

// Metric represents a metrics collection entity in the domain
type Metric interface {
	Run(ctx context.Context) error
	Configure(id utils.EntityID, cfg []byte, systemReader SystemMetricsReader) error
	Eq(newCfg []byte) (bool, error)
}

// MetricConfig represents the configuration for a metric
type MetricConfig struct {
	Type     string        `json:"type"`
	Name     string        `json:"name"`
	Interval time.Duration `json:"interval"`
}

func (c *MetricConfig) Valid(ctx context.Context) map[string]string {
	problems := make(map[string]string, 3)

	err := utils.CheckName(c.Name)
	if err != nil {
		problems["name"] = err.Error()
	}

	if len(c.Type) == 0 {
		problems["type"] = "'type' is required"
	}

	if c.Interval == 0 {
		problems["interval"] = "interval should be more than zero"
	}

	return problems
}

// NewMetricID creates a metric entity ID
func NewMetricID(instance, service, metricType, name string) utils.EntityID {
	return utils.EntityID{
		Kind: "metric",
		Labels: map[string]string{
			"instance": instance,
			"service":  service,
			"type":     metricType,
			"name":     name,
		},
	}
}

// NewMetricIDFromServiceID creates a metric ID from a service ID
func NewMetricIDFromServiceID(serviceID utils.EntityID, metricType, name string) utils.EntityID {
	return NewMetricID(
		serviceID.Labels["instance"],
		serviceID.Labels["name"],
		metricType,
		name,
	)
}


