package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"meerkat-v0/pkg/utils"
	"meerkat-v0/internal/shared/validation"
)

// CPUMetrics is a domain service for CPU metrics collection
type CPUMetrics struct {
	ID   utils.EntityID
	sink Sink
}

// Run collects and emits CPU metrics
func (m *CPUMetrics) Run(ctx context.Context) error {
	contents, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return err
	}
	values := strings.Split(string(contents), " ")
	f64, err := strconv.ParseFloat(values[0], 32)
	if err != nil {
		return err
	}
	
	sample := NewSample(
		m.ID,
		time.Now(),
		MetricGauge,
		"cpu_loadavg",
		f64,
		map[string]string{
			"span": "1m",
		},
	)
	
	return m.sink.Emit(ctx, sample)
}

// Configure configures the CPU metrics collector with the given ID and raw config
func (m *CPUMetrics) Configure(id utils.EntityID, cfg []byte) error {
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
	return nil
}

// Eq checks if the new config equals the current config
func (m *CPUMetrics) Eq(newCfg []byte) (bool, error) {
	// CPU metrics config doesn't change behavior, so always return true
	return true, nil
}

