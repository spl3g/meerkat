package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"meerkat-v0/pkg/utils"
	"meerkat-v0/internal/metrics/domain"
	entitydomain "meerkat-v0/internal/shared/entity/domain"
	"meerkat-v0/internal/shared/validation"
	"meerkat-v0/internal/infrastructure/logger"
)

// Service handles metrics collection lifecycle management
type Service struct {
	logger     *logger.Logger
	metricsRepo domain.Repository
	entityRepo  entitydomain.Repository

	mu sync.RWMutex
	// Metric ID to metric instance
	metrics map[string]*MetricInstance
	// Service ID to service instance
	services map[string]*ServiceInstance

	wg sync.WaitGroup

	ctx    context.Context
	cancel context.CancelFunc
}

// NewService creates a new metrics service
func NewService(logger *logger.Logger, metricsRepo domain.Repository, entityRepo entitydomain.Repository) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		logger:     logger,
		metricsRepo: metricsRepo,
		entityRepo:  entityRepo,
		metrics:     make(map[string]*MetricInstance),
		services:    make(map[string]*ServiceInstance),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// LoadService loads metrics for a service
func (s *Service) LoadService(ctx context.Context, serviceID utils.EntityID, rawConfigs []json.RawMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sink := NewDBSink(s.metricsRepo)
	newMetrics, err := s.buildAll(serviceID, rawConfigs, sink)
	if err != nil {
		return err
	}

	// Ensure entities exist in database
	for id := range newMetrics {
		_, err := s.entityRepo.GetID(ctx, id)
		if errors.Is(err, entitydomain.ErrIDNotFound) {
			_, err = s.entityRepo.InsertEntity(ctx, id)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}

	// Remove metrics that are no longer in config
	service, exists := s.services[serviceID.Canonical()]
	if exists {
		for _, id := range service.MetricIDs {
			_, ok := newMetrics[id]
			if !ok {
				s.stopInstanceUnsynced(id)
				delete(s.metrics, id)
			}
		}
	}

	// Update or add metrics
	for id, inst := range newMetrics {
		old, ok := s.metrics[id]
		if ok && old.running {
			s.stopInstanceUnsynced(id)
		}

		ctx, cancel := context.WithCancel(s.ctx)
		inst.ctx = ctx
		inst.cancel = cancel

		s.metrics[id] = inst
		s.startInstanceUnsynced(id)
	}

	// Update service instance
	metricIDs := make([]string, 0, len(newMetrics))
	for id := range newMetrics {
		metricIDs = append(metricIDs, id)
	}
	s.services[serviceID.Canonical()] = NewServiceInstance(serviceID, nil, metricIDs)

	return nil
}

// Stop stops all metrics collection
func (s *Service) Stop(ctx context.Context) error {
	s.cancel()

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (s *Service) buildAll(serviceID utils.EntityID, rawConfigs []json.RawMessage, sink domain.Sink) (map[string]*MetricInstance, error) {
	result := make(map[string]*MetricInstance)

	for i, rawMetricCfg := range rawConfigs {
		id, metric, err := domain.BuildMetric(serviceID, rawMetricCfg, sink)
		var nnerr *validation.NoNameError
		if errors.As(err, &nnerr) {
			nnerr.SetIndex(i)
			return nil, nnerr
		} else if err != nil {
			return nil, err
		}

		var cfg domain.MetricConfig
		if err := json.Unmarshal(rawMetricCfg, &cfg); err != nil {
			return nil, err
		}

		canon := id.Canonical()

		if _, exists := result[canon]; exists {
			return nil, validation.NewDuplicateFoundError(serviceID.Labels["name"], fmt.Sprint(i))
		}

		result[canon] = NewMetricInstance(id, metric, cfg, rawMetricCfg)
	}

	return result, nil
}

func (s *Service) startInstanceUnsynced(metricID string) {
	inst := s.metrics[metricID]
	inst.running = true
	s.wg.Add(1)
	go func() {
		s.runMetric(inst)
		s.wg.Done()
	}()
}

func (s *Service) stopInstanceUnsynced(metricID string) {
	inst := s.metrics[metricID]
	if !inst.running {
		return
	}
	inst.cancel()
	inst.running = false
}

func (s *Service) runMetric(inst *MetricInstance) {
	interval := inst.Config.Interval
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := inst.Metric.Run(inst.ctx)
			if errors.Is(err, context.Canceled) {
				s.logger.Warn("Metrics tick error", "id", inst.ID.Canonical(), "err", err)
				continue
			}
		case <-inst.ctx.Done():
			return
		}
	}
}

// DBSink implements the domain Sink interface using the repository
type DBSink struct {
	repo domain.Repository
}

// NewDBSink creates a new database sink
func NewDBSink(repo domain.Repository) *DBSink {
	return &DBSink{repo: repo}
}

// Emit emits a metrics sample to the database
func (s *DBSink) Emit(ctx context.Context, sample domain.Sample) error {
	return s.repo.InsertSample(ctx, sample)
}

