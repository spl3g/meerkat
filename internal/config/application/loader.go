package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"meerkat-v0/internal/config/domain"
	sharedlogger "meerkat-v0/internal/shared/logger"
	monitoringdomain "meerkat-v0/internal/monitoring/domain"
	metricsdomain "meerkat-v0/internal/metrics/domain"
	"meerkat-v0/internal/shared/validation"
)

// Loader handles configuration loading and translation to domain operations
type Loader struct {
	logger         sharedlogger.Logger
	monitorService monitoringdomain.Service
	metricsService metricsdomain.Service
	mu            sync.RWMutex
	currentConfig []byte // Stores the current configuration as raw JSON
}

// NewLoader creates a new configuration loader
func NewLoader(logger sharedlogger.Logger, monitorService monitoringdomain.Service, metricsService metricsdomain.Service) *Loader {
	return &Loader{
		logger:         logger,
		monitorService: monitorService,
		metricsService: metricsService,
	}
}

// LoadConfig loads and applies configuration from raw JSON bytes
func (l *Loader) LoadConfig(ctx context.Context, rawConfig []byte) error {
	l.logger.Debug("Parsing configuration")
	var cfg domain.InstanceConfig
	err := json.Unmarshal(rawConfig, &cfg)
	if err != nil {
		l.logger.Error("Failed to parse config", "err", err)
		return fmt.Errorf("failed to parse config: %w", err)
	}

	l.logger.Debug("Validating configuration", "instance_name", cfg.Name, "service_count", len(cfg.Services))
	problems := cfg.Valid(context.TODO())
	if len(problems) > 0 {
		l.logger.Warn("Configuration validation failed", "instance_name", cfg.Name, "problem_count", len(problems))
		return validation.NewValidationError(problems, cfg.Name)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Store the current config
	l.currentConfig = rawConfig

	for i, service := range cfg.Services {
		var servCfg map[string]json.RawMessage
		err := json.Unmarshal(service, &servCfg)
		if err != nil {
			l.logger.Error("Failed to parse service config", "index", i, "err", err)
			return fmt.Errorf("failed to parse service config at index %d: %w", i, err)
		}

		anyName, exists := servCfg["name"]
		if !exists {
			err := validation.NewNoNameError(cfg.Name)
			err.SetIndex(i)
			l.logger.Error("Service missing name", "index", i)
			return err
		}
		var name string
		err = json.Unmarshal(anyName, &name)
		if !exists || err != nil {
			err := validation.NewNoNameError(cfg.Name)
			err.SetIndex(i)
			l.logger.Error("Service name invalid", "index", i, "err", err)
			return err
		}

		serviceID := domain.NewServiceID(cfg.Name, name)
		l.logger.Debug("Processing service", "service_id", serviceID.Canonical())

		// Load monitors
		if rawMonitors, exists := servCfg["monitors"]; exists {
			var monitorConfigs []json.RawMessage
			err = json.Unmarshal(rawMonitors, &monitorConfigs)
			if err != nil {
				l.logger.Error("Failed to parse monitors", "service", name, "err", err)
				return fmt.Errorf("failed to parse monitors for service %s: %w", name, err)
			}
			l.logger.Debug("Loading monitors", "service", name, "count", len(monitorConfigs))
			err = l.monitorService.LoadService(ctx, serviceID, monitorConfigs)
			var valErr validation.ConfigError
			if errors.As(err, &valErr) {
				valErr.PrependPath(cfg.Name)
				return err
			} else if err != nil {
				l.logger.Error("Failed to load monitors", "service", name, "err", err)
				return fmt.Errorf("failed to load monitors for service %s: %w", name, err)
			}
			l.logger.Debug("Monitors loaded", "service", name)
		}

		// Load metrics
		if rawMetrics, exists := servCfg["metrics"]; exists {
			var metricConfigs []json.RawMessage
			err = json.Unmarshal(rawMetrics, &metricConfigs)
			if err != nil {
				l.logger.Error("Failed to parse metrics", "service", name, "err", err)
				return fmt.Errorf("failed to parse metrics for service %s: %w", name, err)
			}
			l.logger.Debug("Loading metrics", "service", name, "count", len(metricConfigs))
			err = l.metricsService.LoadService(ctx, serviceID, metricConfigs)
			var valErr validation.ConfigError
			if errors.As(err, &valErr) {
				valErr.PrependPath(cfg.Name)
				return err
			} else if err != nil {
				l.logger.Error("Failed to load metrics", "service", name, "err", err)
				return fmt.Errorf("failed to load metrics for service %s: %w", name, err)
			}
			l.logger.Debug("Metrics loaded", "service", name)
		}
	}

	l.logger.Info("Configuration loaded successfully", "instance_name", cfg.Name, "service_count", len(cfg.Services))
	return nil
}

// GetConfig returns the current configuration
func (l *Loader) GetConfig() []byte {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.currentConfig
}

// Stop stops all services
func (l *Loader) Stop(ctx context.Context) error {
	l.logger.Debug("Stopping configuration loader")
	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := l.monitorService.Stop(ctx); err != nil {
			l.logger.Error("Monitor service stop error", "err", err)
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
			l.logger.Error("Metrics service stop error", "err", err)
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
		l.logger.Warn("Stop timeout exceeded", "err", ctx.Err())
		return ctx.Err()
	case err := <-errChan:
		return err
	case <-done:
		l.logger.Debug("Configuration loader stopped")
		return nil
	}
}

