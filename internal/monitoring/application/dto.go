package application

import (
	"context"

	"meerkat-v0/pkg/utils"
	monitordomain "meerkat-v0/internal/monitoring/domain"
)

// MonitorInstance represents a running monitor instance in the application layer
type MonitorInstance struct {
	ID      utils.EntityID
	Monitor monitordomain.Monitor
	Config  monitordomain.MonitorConfig
	RawCfg  []byte

	ctx     context.Context
	cancel  context.CancelFunc
	running bool
}

// NewMonitorInstance creates a new monitor instance
func NewMonitorInstance(id utils.EntityID, monitor monitordomain.Monitor, cfg monitordomain.MonitorConfig, rawCfg []byte) *MonitorInstance {
	return &MonitorInstance{
		ID:      id,
		Monitor: monitor,
		Config:  cfg,
		RawCfg:  rawCfg,
	}
}

// ServiceInstance represents a service with its monitors
type ServiceInstance struct {
	ID        utils.EntityID
	MonitorIDs []string
	RawCfg    []byte
}

// NewServiceInstance creates a new service instance
func NewServiceInstance(id utils.EntityID, rawCfg []byte, monitorIDs []string) *ServiceInstance {
	return &ServiceInstance{
		ID:         id,
		MonitorIDs: monitorIDs,
		RawCfg:     rawCfg,
	}
}

// ConfigDiff represents the difference between old and new configurations
type ConfigDiff struct {
	Add    []string
	Update []string
	Delete []string
}

