package application

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"meerkat-v0/pkg/utils"
	"meerkat-v0/internal/monitoring/domain"
	"meerkat-v0/internal/shared/validation"
)

// HTTPMonitor is an application service for HTTP monitoring
type HTTPMonitor struct {
	ID         utils.EntityID
	cfg        domain.HTTPConfig
	httpClient domain.HTTPClient
}

// Run executes an HTTP check
func (m *HTTPMonitor) Run(parentCtx context.Context) error {
	timeout := time.Duration(m.cfg.Timeout) * time.Millisecond
	return m.httpClient.Do(parentCtx, m.cfg.Method, m.cfg.URL, timeout, m.cfg.ExpectedStatus)
}

// Configure configures the HTTP monitor with the given ID and raw config
func (m *HTTPMonitor) Configure(id utils.EntityID, rawCfg []byte, httpClient domain.HTTPClient, tcpClient domain.TCPClient) error {
	var cfg domain.HTTPConfig
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
	m.httpClient = httpClient
	return nil
}

// Eq checks if the new config equals the current config
func (m *HTTPMonitor) Eq(newRawCfg []byte) (bool, error) {
	var newCfg domain.HTTPConfig
	err := json.Unmarshal(newRawCfg, &newCfg)
	if err != nil {
		return false, err
	}

	// Normalize method for comparison
	if len(newCfg.Method) == 0 {
		newCfg.Method = "GET"
	} else {
		newCfg.Method = strings.ToUpper(newCfg.Method)
	}

	return m.cfg == newCfg, nil
}

