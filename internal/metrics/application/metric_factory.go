package application

import (
	"context"
	"encoding/json"

	"meerkat-v0/pkg/utils"
	"meerkat-v0/internal/metrics/domain"
	"meerkat-v0/internal/shared/validation"
)

// BuildMetric creates a metric from raw configuration
// sink and systemReader are injected into metrics that need them
// Returns a domain.Metric interface, but creates application layer implementations
func BuildMetric(serviceID utils.EntityID, rawCfg []byte, sink domain.Sink, systemReader domain.SystemMetricsReader) (utils.EntityID, domain.Metric, error) {
	var id utils.EntityID
	var cfg domain.MetricConfig
	err := json.Unmarshal(rawCfg, &cfg)
	if err != nil {
		return id, nil, err
	}

	problems := cfg.Valid(context.TODO())
	if len(problems) > 0 {
		return id, nil, validation.NewValidationError(problems, serviceID.Labels["name"], cfg.Name)
	}

	id = domain.NewMetricIDFromServiceID(serviceID, cfg.Type, cfg.Name)

	// Create application layer implementations
	var metric domain.Metric
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

	err = metric.Configure(id, rawCfg, systemReader)
	if err != nil {
		return id, nil, err
	}

	return id, metric, nil
}

