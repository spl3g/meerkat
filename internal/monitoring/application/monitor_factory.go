package application

import (
	"context"
	"encoding/json"

	"meerkat-v0/pkg/utils"
	"meerkat-v0/internal/monitoring/domain"
	"meerkat-v0/internal/shared/validation"
)

// BuildMonitor creates a monitor from raw configuration
// httpClient and tcpClient are injected into monitors that need them
// Returns a domain.Monitor interface, but creates application layer implementations
func BuildMonitor(serviceID utils.EntityID, rawCfg []byte, httpClient domain.HTTPClient, tcpClient domain.TCPClient) (utils.EntityID, domain.Monitor, error) {
	var id utils.EntityID
	var cfg domain.MonitorConfig
	err := json.Unmarshal(rawCfg, &cfg)
	if err != nil {
		return id, nil, err
	}

	problems := cfg.Valid(context.TODO())
	if len(problems) > 0 {
		return id, nil, validation.NewValidationError(problems, serviceID.Labels["name"], cfg.Name)
	}

	id = domain.NewMonitorIDFromServiceID(serviceID, cfg.Type, cfg.Name)

	// Create application layer implementations
	var monitor domain.Monitor
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

	err = monitor.Configure(id, rawCfg, httpClient, tcpClient)
	if err != nil {
		return id, nil, err
	}

	return id, monitor, nil
}

