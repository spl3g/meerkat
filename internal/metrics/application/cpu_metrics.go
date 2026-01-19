package application

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"meerkat-v0/pkg/utils"
	"meerkat-v0/internal/metrics/domain"
	"meerkat-v0/internal/shared/validation"
)

// CPUMetrics is an application service for CPU metrics collection
type CPUMetrics struct {
	ID           utils.EntityID
	sink         domain.Sink
	systemReader domain.SystemMetricsReader
}

// Run collects and emits CPU metrics
func (m *CPUMetrics) Run(ctx context.Context) error {
	loadAvg, err := m.systemReader.ReadLoadAvg(ctx)
	if err != nil {
		return err
	}

	sample := domain.NewSample(
		m.ID,
		time.Now(),
		domain.MetricGauge,
		"cpu_loadavg",
		loadAvg,
		map[string]string{
			"span": "1m",
		},
	)

	return m.sink.Emit(ctx, sample)
}

// Configure configures the CPU metrics collector with the given ID and raw config
func (m *CPUMetrics) Configure(id utils.EntityID, cfg []byte, systemReader domain.SystemMetricsReader) error {
	var config struct {
		Type string `json:"type"`
		Name string `json:"name"`
	}
	err := json.Unmarshal(cfg, &config)
	if err != nil {
		return validation.NewValidationError(map[string]string{
			"config": fmt.Sprintf("invalid config: %v", err),
		}, id.Labels["name"])
	}

	m.ID = id
	m.systemReader = systemReader
	return nil
}

// Eq checks if the new config equals the current config
func (m *CPUMetrics) Eq(newCfg []byte) (bool, error) {
	// CPU metrics config doesn't change behavior, so always return true
	return true, nil
}

