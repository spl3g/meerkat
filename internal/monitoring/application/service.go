package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"meerkat-v0/pkg/utils"
	"meerkat-v0/internal/monitoring/domain"
	entitydomain "meerkat-v0/internal/shared/entity/domain"
	"meerkat-v0/internal/shared/validation"
	"meerkat-v0/internal/infrastructure/logger"
)

// Service handles monitor lifecycle management
type Service struct {
	logger      *logger.Logger
	monitorRepo domain.Repository
	entityRepo  entitydomain.Repository

	mu sync.RWMutex
	// Monitor ID to monitor instance
	monitors map[string]*MonitorInstance
	// Service ID to service instance
	services map[string]*ServiceInstance

	wg sync.WaitGroup

	ctx    context.Context
	cancel context.CancelFunc
}

// NewService creates a new monitor service
func NewService(logger *logger.Logger, monitorRepo domain.Repository, entityRepo entitydomain.Repository) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		logger:     logger,
		monitorRepo: monitorRepo,
		entityRepo:  entityRepo,
		monitors:    make(map[string]*MonitorInstance),
		services:    make(map[string]*ServiceInstance),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// LoadService loads monitors for a service
func (s *Service) LoadService(ctx context.Context, serviceID utils.EntityID, rawConfigs []json.RawMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	newMonitors, err := s.buildAll(serviceID, rawConfigs)
	if err != nil {
		return err
	}

	// Ensure entities exist in database
	for id := range newMonitors {
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

	// Remove monitors that are no longer in config
	service, exists := s.services[serviceID.Canonical()]
	if exists {
		for _, id := range service.MonitorIDs {
			_, ok := newMonitors[id]
			if !ok {
				s.stopInstanceUnsynced(id)
				delete(s.monitors, id)
			}
		}
	}

	// Update or add monitors
	for id, inst := range newMonitors {
		old, ok := s.monitors[id]
		if ok && old.running {
			s.stopInstanceUnsynced(id)
		}

		ctx, cancel := context.WithCancel(s.ctx)
		inst.ctx = ctx
		inst.cancel = cancel

		s.monitors[id] = inst
		s.startInstanceUnsynced(id)
	}

	// Update service instance
	monitorIDs := make([]string, 0, len(newMonitors))
	for id := range newMonitors {
		monitorIDs = append(monitorIDs, id)
	}
	s.services[serviceID.Canonical()] = NewServiceInstance(serviceID, nil, monitorIDs)

	return nil
}

// Stop stops all monitors
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

func (s *Service) buildAll(serviceID utils.EntityID, rawConfigs []json.RawMessage) (map[string]*MonitorInstance, error) {
	result := make(map[string]*MonitorInstance)

	for i, rawMonitorCfg := range rawConfigs {
		id, monitor, err := domain.BuildMonitor(serviceID, rawMonitorCfg)
		var nnerr *validation.NoNameError
		if errors.As(err, &nnerr) {
			nnerr.SetIndex(i)
			return nil, nnerr
		} else if err != nil {
			return nil, err
		}

		var cfg domain.MonitorConfig
		if err := json.Unmarshal(rawMonitorCfg, &cfg); err != nil {
			return nil, err
		}

		canon := id.Canonical()

		if _, exists := result[canon]; exists {
			return nil, validation.NewDuplicateFoundError(serviceID.Labels["name"], fmt.Sprint(i))
		}

		result[canon] = NewMonitorInstance(id, monitor, cfg, rawMonitorCfg)
	}

	return result, nil
}

func (s *Service) startInstanceUnsynced(monitorID string) {
	inst := s.monitors[monitorID]
	inst.running = true
	s.wg.Add(1)
	go func() {
		s.runMonitor(inst)
		s.wg.Done()
	}()
}

func (s *Service) stopInstanceUnsynced(monitorID string) {
	inst := s.monitors[monitorID]
	if !inst.running {
		return
	}
	inst.cancel()
	inst.running = false
}

func (s *Service) runMonitor(inst *MonitorInstance) {
	interval := inst.Config.Interval
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := inst.Monitor.Run(inst.ctx)
			if errors.Is(err, context.Canceled) {
				continue
			}
			heartbeat := domain.NewHeartbeat(inst.ID.Canonical(), time.Now(), err)
			_ = s.monitorRepo.InsertHeartbeat(inst.ctx, heartbeat)
		case <-inst.ctx.Done():
			return
		}
	}
}

