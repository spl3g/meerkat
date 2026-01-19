package application

import (
	"context"
	"encoding/json"
	"time"

	"meerkat-v0/pkg/utils"
	"meerkat-v0/internal/monitoring/domain"
	"meerkat-v0/internal/shared/validation"
)

// TCPMonitor is an application service for TCP monitoring
type TCPMonitor struct {
	ID        utils.EntityID
	cfg       domain.TCPConfig
	tcpClient domain.TCPClient
}

// Run executes a TCP check
func (m *TCPMonitor) Run(parentCtx context.Context) error {
	ctx, cancel := context.WithTimeout(parentCtx, time.Duration(m.cfg.Timeout)*time.Millisecond)
	defer cancel()
	return m.tcpClient.Dial(ctx, m.cfg.Hostname, m.cfg.Port)
}

// Configure configures the TCP monitor with the given ID and raw config
func (m *TCPMonitor) Configure(id utils.EntityID, rawCfg []byte, httpClient domain.HTTPClient, tcpClient domain.TCPClient) error {
	var cfg domain.TCPConfig
	err := json.Unmarshal(rawCfg, &cfg)
	if err != nil {
		return err
	}

	problems := cfg.Valid(context.TODO())
	if len(problems) > 0 {
		return validation.NewValidationError(problems, id.Labels["name"])
	}

	m.ID = id
	m.cfg = cfg
	m.tcpClient = tcpClient
	return nil
}

// Eq checks if the new config equals the current config
func (m *TCPMonitor) Eq(newRawCfg []byte) (bool, error) {
	var newCfg domain.TCPConfig
	err := json.Unmarshal(newRawCfg, &newCfg)
	if err != nil {
		return false, err
	}

	return m.cfg == newCfg, nil
}

