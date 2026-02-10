package domain

import (
	"context"
	"encoding/json"
	"time"

	"meerkat-v0/pkg/utils"
	"meerkat-v0/internal/shared/validation"
)

// Metric represents a metrics collection entity in the domain
type Metric interface {
	Run(ctx context.Context) error
	Configure(id utils.EntityID, cfg []byte) error
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

// BuildMetric creates a metric from raw configuration
func BuildMetric(serviceID utils.EntityID, rawCfg []byte, sink Sink) (utils.EntityID, Metric, error) {
	var id utils.EntityID
	var cfg MetricConfig
	err := json.Unmarshal(rawCfg, &cfg)
	if err != nil {
		return id, nil, err
	}

	problems := cfg.Valid(context.TODO())
	if len(problems) > 0 {
		return id, nil, validation.NewValidationError(problems, serviceID.Labels["name"], cfg.Name)
	}

	id = NewMetricIDFromServiceID(serviceID, cfg.Type, cfg.Name)

	// TODO: Replace with modules/factory pattern
	var metric Metric
	switch cfg.Type {
	case "cpu":
		metric = &CPUMetrics{
			sink: sink,
		}
	default:
		return id, nil, validation.NewValidationError(map[string]string{
			"type": "unknown metrics type: " + cfg.Type,
		}, serviceID.Labels["name"], cfg.Name)
	}

	err = metric.Configure(id, rawCfg)
	if err != nil {
		return id, nil, err
	}

	return id, metric, nil
}

