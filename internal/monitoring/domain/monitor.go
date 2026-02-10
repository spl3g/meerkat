package domain

import (
	"context"
	"encoding/json"
	"time"

	"meerkat-v0/pkg/utils"
	"meerkat-v0/internal/shared/validation"
)

// Monitor represents a monitoring entity in the domain
type Monitor interface {
	Run(ctx context.Context) error
	Configure(id utils.EntityID, cfg []byte) error
	Eq(newCfg []byte) (bool, error)
}

// MonitorConfig represents the configuration for a monitor
type MonitorConfig struct {
	Type     string        `json:"type"`
	Name     string        `json:"name"`
	Interval time.Duration `json:"interval"`
}

func (c *MonitorConfig) Valid(ctx context.Context) map[string]string {
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

// NewMonitorID creates a monitor entity ID
func NewMonitorID(instance, service, monType, name string) utils.EntityID {
	return utils.EntityID{
		Kind: "monitor",
		Labels: map[string]string{
			"instance": instance,
			"service":  service,
			"type":     monType,
			"name":     name,
		},
	}
}

// NewMonitorIDFromServiceID creates a monitor ID from a service ID
func NewMonitorIDFromServiceID(serviceID utils.EntityID, monType, name string) utils.EntityID {
	return NewMonitorID(
		serviceID.Labels["instance"],
		serviceID.Labels["name"],
		monType,
		name,
	)
}

// BuildMonitor creates a monitor from raw configuration
func BuildMonitor(serviceID utils.EntityID, rawCfg []byte) (utils.EntityID, Monitor, error) {
	var id utils.EntityID
	var cfg MonitorConfig
	err := json.Unmarshal(rawCfg, &cfg)
	if err != nil {
		return id, nil, err
	}

	problems := cfg.Valid(context.TODO())
	if len(problems) > 0 {
		return id, nil, validation.NewValidationError(problems, serviceID.Labels["name"], cfg.Name)
	}

	id = NewMonitorIDFromServiceID(serviceID, cfg.Type, cfg.Name)

	// TODO: Replace with modules/factory pattern
	var monitor Monitor
	switch cfg.Type {
	case "tcp":
		monitor = &TCPMonitor{}
	case "http":
		monitor = &HTTPMonitor{}
	default:
		return id, nil, validation.NewValidationError(map[string]string{
			"type": "unknown monitor type: " + cfg.Type,
		}, serviceID.Labels["name"], cfg.Name)
	}

	err = monitor.Configure(id, rawCfg)
	if err != nil {
		return id, nil, err
	}

	return id, monitor, nil
}

