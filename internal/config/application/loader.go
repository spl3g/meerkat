package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"meerkat-v0/internal/config/domain"
	monitoringapp "meerkat-v0/internal/monitoring/application"
	metricsapp "meerkat-v0/internal/metrics/application"
	"meerkat-v0/internal/shared/validation"
)

// Loader handles configuration loading and translation to domain operations
type Loader struct {
	monitorService *monitoringapp.Service
	metricsService *metricsapp.Service
	mu            sync.RWMutex
}

// NewLoader creates a new configuration loader
func NewLoader(monitorService *monitoringapp.Service, metricsService *metricsapp.Service) *Loader {
	return &Loader{
		monitorService: monitorService,
		metricsService: metricsService,
	}
}

// LoadConfig loads and applies configuration from raw JSON bytes
func (l *Loader) LoadConfig(ctx context.Context, rawConfig []byte) error {
	var cfg domain.InstanceConfig
	err := json.Unmarshal(rawConfig, &cfg)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	problems := cfg.Valid(context.TODO())
	if len(problems) > 0 {
		return validation.NewValidationError(problems, cfg.Name)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	for i, service := range cfg.Services {
		var servCfg map[string]json.RawMessage
		err := json.Unmarshal(service, &servCfg)
		if err != nil {
			return fmt.Errorf("failed to parse service config at index %d: %w", i, err)
		}

		anyName, exists := servCfg["name"]
		if !exists {
			err := validation.NewNoNameError(cfg.Name)
			err.SetIndex(i)
			return err
		}
		var name string
		err = json.Unmarshal(anyName, &name)
		if !exists || err != nil {
			err := validation.NewNoNameError(cfg.Name)
			err.SetIndex(i)
			return err
		}

		serviceID := domain.NewServiceID(cfg.Name, name)

		// Load monitors
		if rawMonitors, exists := servCfg["monitors"]; exists {
			var monitorConfigs []json.RawMessage
			err = json.Unmarshal(rawMonitors, &monitorConfigs)
			if err != nil {
				return fmt.Errorf("failed to parse monitors for service %s: %w", name, err)
			}
			err = l.monitorService.LoadService(ctx, serviceID, monitorConfigs)
			var valErr validation.ConfigError
			if errors.As(err, &valErr) {
				valErr.PrependPath(cfg.Name)
				return err
			} else if err != nil {
				return fmt.Errorf("failed to load monitors for service %s: %w", name, err)
			}
		}

		// Load metrics
		if rawMetrics, exists := servCfg["metrics"]; exists {
			var metricConfigs []json.RawMessage
			err = json.Unmarshal(rawMetrics, &metricConfigs)
			if err != nil {
				return fmt.Errorf("failed to parse metrics for service %s: %w", name, err)
			}
			err = l.metricsService.LoadService(ctx, serviceID, metricConfigs)
			var valErr validation.ConfigError
			if errors.As(err, &valErr) {
				valErr.PrependPath(cfg.Name)
				return err
			} else if err != nil {
				return fmt.Errorf("failed to load metrics for service %s: %w", name, err)
			}
		}
	}

	return nil
}

// Stop stops all services
func (l *Loader) Stop(ctx context.Context) error {
	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := l.monitorService.Stop(ctx); err != nil {
			select {
			case errChan <- err:
			default:
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := l.metricsService.Stop(ctx); err != nil {
			select {
			case errChan <- err:
			default:
			}
		}
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	case <-done:
		return nil
	}
}

