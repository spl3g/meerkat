package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"meerkat-v0/db"
	"meerkat-v0/utils"
)

type Entity interface {
	Run(ctx context.Context) error
	Configure(id utils.EntityID, cfg []byte) error
	Eq(newCfg []byte) (bool, error)
}

type EntityConfig struct {
	Type     string        `json:"type"`
	Name     string        `json:"name"`
	Interval time.Duration `json:"interval"`
}

func (c *EntityConfig) Valid(ctx context.Context) map[string]string {
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

type EntityInstance struct {
	ID     utils.EntityID
	Ent    Entity
	Cfg    EntityConfig
	RawCfg []byte

	ctx     context.Context
	cancel  context.CancelFunc
	running bool
}

func NewEntityInstance(id utils.EntityID, ent Entity, cfg EntityConfig, rawCfg []byte) *EntityInstance {
	return &EntityInstance{
		ID:     id,
		Ent:    ent,
		Cfg:    cfg,
		RawCfg: rawCfg,
	}
}

type ServiceInstance struct {
	ID        utils.EntityID
	EntityIDs []string
	RawCfg    []byte
}

func NewServiceInstance(id utils.EntityID, rawCfg []byte, entityIDs []string) *ServiceInstance {
	return &ServiceInstance{
		ID:        id,
		EntityIDs: entityIDs,
		RawCfg:    rawCfg,
	}
}

type EntityBuilder func(
	serviceID utils.EntityID,
	raw []byte,
) (
	id utils.EntityID,
	inst *EntityInstance,
	err error,
)

type EntityRunner func(logger *utils.Logger, inst *EntityInstance)

type EntityService struct {
	Name string

	logger *utils.Logger

	buildEntity EntityBuilder
	runEntity   EntityRunner

	entityRepo EntityRepo

	mu sync.RWMutex
	// Monitor ID to entity instance
	entities map[string]*EntityInstance
	// Service ID to a list of entity ids
	services map[string]*ServiceInstance

	wg sync.WaitGroup

	ctx    context.Context
	cancel context.CancelFunc
}

func NewEntityService(name string, entityBuilder EntityBuilder, entityRunner EntityRunner, entityRepo EntityRepo) *EntityService {
	ctx, cancel := context.WithCancel(context.Background())
	return &EntityService{
		Name:        name,
		entityRepo:  entityRepo,
		buildEntity: entityBuilder,
		runEntity:   entityRunner,
		entities:    make(map[string]*EntityInstance),
		ctx:         ctx,
		cancel:      cancel,
	}
}

func (m *EntityService) DiffEntities(serviceID utils.EntityID, rawConfigs []json.RawMessage) (*ConfigDiff, error) {
	add := make([]string, 10)
	update := make([]string, 10)
	delete := make([]string, 10)

	newConfigs := make(map[string]EntityConfig)
	for _, rawCfg := range rawConfigs {
		var cfg EntityConfig
		err := json.Unmarshal(rawCfg, &cfg)
		if err != nil {
			return nil, err
		}

		problems := cfg.Valid(context.TODO())
		if len(problems) > 0 {
			return nil, NewValidationError(problems, serviceID.Labels["name"], cfg.Name)
		}

		id := NewMonitorIDFromServiceID(serviceID, cfg.Type, cfg.Name)

		oldCfg, ok := m.entities[id.Canonical()]
		if !ok {
			add = append(add, id.Canonical())
			continue
		}

		ok, err = oldCfg.Ent.Eq(rawCfg)
		if err != nil {
			return nil, err
		}
		if !ok {
			update = append(update, id.Canonical())
		}
		newConfigs[id.Canonical()] = cfg
	}

	for id := range m.entities {
		_, ok := newConfigs[id]
		if ok {
			continue
		}
		delete = append(delete, id)
	}

	return &ConfigDiff{
		Add:    add,
		Update: update,
		Delete: delete,
	}, nil
}

func (m *EntityService) LoadService(ctx context.Context, serviceID utils.EntityID, rawConfigs []json.RawMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	newMonitors, err := m.buildAll(serviceID, rawConfigs)
	if err != nil {
		return err
	}

	for id := range newMonitors {
		_, err := m.entityRepo.GetID(ctx, id)
		if errors.Is(err, ErrIDNotFound) {
			_, err = m.entityRepo.InsertEntity(ctx, id)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}

	service, exisits := m.services[serviceID.Canonical()]
	if exisits {
		for _, id := range service.EntityIDs {
			_, ok := newMonitors[id]
			if !ok {
				m.stopInstanceUnsynced(id)
				delete(m.entities, id)
			}
		}
	}

	for id, inst := range newMonitors {
		old, ok := m.entities[id]
		if ok && old.running {
			m.stopInstanceUnsynced(id)
		}

		ctx, cancel := context.WithCancel(m.ctx)

		inst.ctx = ctx
		inst.cancel = cancel

		m.entities[id] = inst

		m.startInstanceUnsynced(id)
	}

	return nil
}

func (m *EntityService) buildAll(serviceID utils.EntityID, rawConfigs []json.RawMessage) (map[string]*EntityInstance, error) {
	result := make(map[string]*EntityInstance)

	for i, rawMonitorCfg := range rawConfigs {
		id, inst, err := m.buildEntity(serviceID, rawMonitorCfg)
		var nnerr *NoNameError
		if errors.As(err, &nnerr) {
			nnerr.SetIndex(i)
			return nil, nnerr
		} else if err != nil {
			return nil, err
		}

		canon := id.Canonical()

		if _, exists := result[canon]; exists {
			return nil, NewDuplicateFoundError(serviceID.Labels["name"], fmt.Sprint(i))
		}

		result[canon] = inst
	}

	return result, nil
}

func (m *EntityService) startInstanceUnsynced(monitorID string) {
	m.wg.Add(1)
	go func() {
		m.runEntity(m.logger, m.entities[monitorID])
		m.wg.Done()
	}()
}

func (m *EntityService) stopInstanceUnsynced(monitorID string) {
	monitor := m.entities[monitorID]
	if !monitor.running {
		return
	}
	monitor.cancel()
	monitor.running = false
}

var ErrIDNotFound = errors.New("could not find entity with this id")

type EntityRepo interface {
	GetID(ctx context.Context, canonID string) (int64, error)
	InsertEntity(ctx context.Context, canonID string) (int64, error)
	GetCanonicalID(ctx context.Context, id int64) (string, error)
}

type SqliteEntityRepo struct {
	readDB  *db.Queries
	writeDB *db.Queries
}

func NewSqliteEntityRepo(readDB *db.Queries, writeDB *db.Queries) *SqliteEntityRepo {
	return &SqliteEntityRepo{
		readDB:  readDB,
		writeDB: writeDB,
	}
}

func (r *SqliteEntityRepo) GetID(ctx context.Context, canonID string) (int64, error) {
	id, err := r.readDB.GetEntityID(ctx, canonID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrIDNotFound
	}
	return id, nil
}

func (r *SqliteEntityRepo) GetCanonicalID(ctx context.Context, id int64) (string, error) {
	canon, err := r.readDB.GetCanonicalID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrIDNotFound
	}
	return canon, nil
}

func (r *SqliteEntityRepo) InsertEntity(ctx context.Context, canonID string) (int64, error) {
	id, err := r.writeDB.InsertEntity(ctx, canonID)
	if err != nil {
		return 0, err
	}

	return id, nil
}
