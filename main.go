package main

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"meerkat-v0/db"
	"meerkat-v0/utils"
)

//go:embed schema.sql
var ddl string

func help() {
	fmt.Fprintln(os.Stderr, "./meerkat [config]")
}

func run() error {
	if len(os.Args) < 2 {
		help()
		return fmt.Errorf("not enough arguments")
	}

	sigCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	configPath := os.Args[1]
	rawCfg, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	dbRead, err := connectSqliteDb("observations.db")
	if err != nil {
		return err
	}
	defer dbRead.Close()
	dbRead.SetMaxOpenConns(runtime.NumCPU())

	dbWrite, err := connectSqliteDb("observations.db")
	if err != nil {
		return err
	}
	defer dbWrite.Close()
	dbWrite.SetMaxOpenConns(1)

	_, err = dbWrite.ExecContext(sigCtx, ddl)
	if err != nil {
		return err
	}

	readDB := db.New(dbRead)
	writeDB := db.New(dbWrite)

	entityRepo := NewSqliteEntityRepo(readDB, writeDB)
	heartbeatRepo := NewSqliteHeartbeatRepo(readDB, writeDB, entityRepo)
	metricsRepo := NewSqliteMetricsRepo(readDB, writeDB, entityRepo)

	monitorRunner := func(logger *utils.Logger, inst *EntityInstance) {
		RunMonitor(heartbeatRepo, logger, inst)
	}

	metricsBuilder := func(serviceID utils.EntityID, rawCfg []byte) (utils.EntityID, *EntityInstance, error) {
		return BuildMetrics(metricsRepo, serviceID, rawCfg)
	}

	monitorService := NewEntityService("monitor", BuildMonitor, monitorRunner, entityRepo)
	metricsSerivce := NewEntityService("metrics", metricsBuilder, RunMetrics, entityRepo)

	meerkat := NewMeerkat([]*EntityService{monitorService, metricsSerivce})
	err = meerkat.LoadConfig(sigCtx, rawCfg)
	if err != nil {
		return err
	}

	<-sigCtx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 350*time.Millisecond)
	defer cancel()
	return meerkat.Stop(ctx)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func connectSqliteDb(dbName string) (*sql.DB, error) {
	return sql.Open("sqlite", dbName)
}

type ConfigDiff struct {
	Add    []string
	Update []string
	Delete []string
}

type InstanceConfig struct {
	Name     string            `json:"name"`
	Services []json.RawMessage `json:"services"`
}

func (c *InstanceConfig) Valid(ctx context.Context) map[string]string {
	problems := make(map[string]string, 2)

	err := utils.CheckName(c.Name)
	if err != nil {
		problems["name"] = err.Error()
	}

	if len(c.Services) == 0 {
		problems["services"] = "services cannot be empty"
	}

	return problems
}

type ServiceConfig struct {
	Name string `json:"name"`
}

func (c *ServiceConfig) Valid(ctx context.Context) map[string]string {
	err := utils.CheckName(c.Name)
	if err != nil {
		return map[string]string{
			"name": err.Error(),
		}
	}

	return nil
}

func NewServiceID(instance string, name string) utils.EntityID {
	return utils.EntityID{
		Kind: "service",
		Labels: map[string]string{
			"instance": instance,
			"name":     name,
		},
	}
}

type Meerkat struct {
	services map[string]*EntityService

	rawCfg []byte
	cfg    InstanceConfig
	mu     sync.RWMutex
}

func NewMeerkat(services []*EntityService) *Meerkat {
	serviceMap := make(map[string]*EntityService, len(services))
	for _, service := range services {
		serviceMap[service.Name] = service
	}
	return &Meerkat{
		services: serviceMap,
	}
}

func (m *Meerkat) LoadConfig(ctx context.Context, newConfig []byte) error {
	var cfg InstanceConfig
	err := json.Unmarshal(newConfig, &cfg)
	if err != nil {
		return err
	}

	problems := cfg.Valid(context.TODO())
	if len(problems) > 0 {
		return NewValidationError(problems, cfg.Name)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for i, service := range cfg.Services {
		var servCfg map[string]json.RawMessage
		err := json.Unmarshal(service, &servCfg)
		if err != nil {
			return err
		}

		anyName, exists := servCfg["name"]
		if !exists {
			err := NewNoNameError(cfg.Name)
			err.SetIndex(i)
			return err
		}
		var name string
		err = json.Unmarshal(anyName, &name)
		if !exists || err != nil {
			err := NewNoNameError(cfg.Name)
			err.SetIndex(i)
			return err
		}

		id := NewServiceID(cfg.Name, name)
		for name, entService := range m.services {
			var configs []json.RawMessage
			rawConfigs, exists := servCfg[name]
			if !exists {
				continue
			}

			err = json.Unmarshal(rawConfigs, &configs)
			if err != nil {
				return err
			}
			err = entService.LoadService(ctx, id, configs)
			var val ConfigError
			if errors.As(err, &val) {
				val.PrependPath(cfg.Name)
				return err
			} else if err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *Meerkat) Stop(ctx context.Context) error {
	var wg sync.WaitGroup

	errChan := make(chan error, 1)
	for _, service := range m.services {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := service.Stop(ctx)
			if err != nil {
				select {
				case errChan <- err:
				default:
				}
			}
		}()
	}

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
