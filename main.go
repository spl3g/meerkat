package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"meerkat-v0/utils"
)

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

	hr := WriterHeartbeat{os.Stdout}

	ms := NewMonitorService(&hr)

	meerkat := NewMeerkat(ms)
	err = meerkat.LoadConfig(rawCfg)
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

type ConfigDiff struct {
	Add    []string
	Update []string
	Delete []string
}

type InstanceConfig struct {
	Name     string          `json:"name"`
	Services []ServiceConfig `json:"services"`
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

	for i, service := range c.Services {
		serviceProblems := service.Valid(ctx)
		for field, problem := range serviceProblems {
			problemName := fmt.Sprintf("services[%d].%s", i, field)
			problems[problemName] = problem
		}
	}

	return problems
}

type ServiceConfig struct {
	Name     string            `json:"name"`
	Monitors []json.RawMessage `json:"monitors"`
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
	monitors *MonitorService

	rawCfg []byte
	cfg    InstanceConfig
	mu     sync.RWMutex
}

func NewMeerkat(monitorService *MonitorService) *Meerkat {
	return &Meerkat{
		monitors: monitorService,
	}
}

func (m *Meerkat) LoadConfig(newConfig []byte) error {
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

	for _, service := range cfg.Services {
		id := NewServiceID(cfg.Name, service.Name)
		err := m.monitors.LoadService(id, service.Monitors)
		var val *ValidationError
		if errors.As(err, &val) {
			val.PrependPath(cfg.Name)
			return err
		} else if err != nil {
			return err
		}
	}

	return nil
}

func (m *Meerkat) Stop(ctx context.Context) error {
	var wg sync.WaitGroup

	errChan := make(chan error, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := m.monitors.Stop(ctx)
		if err != nil {
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
