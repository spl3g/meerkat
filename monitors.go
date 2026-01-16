package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"sync"
	"time"

	"meerkat-v0/utils"
)

type Monitor interface {
	Check(ctx context.Context) error
	Configure(id utils.EntityID, cfg []byte) error
	Eq(newCfg []byte) (bool, error)
}

type MonitorConfig struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Interval uint16 `json:"interval"`
}

func NewMonitorID(instance, service, monType, name string) utils.EntityID {
	return utils.EntityID{
		Kind: "monitor",
		Labels: map[string]string{
			"instance": instance,
			"service":  service,
			"type":     monType,
			"name":     name,
		},
	}
}

func NewMonitorIDFromServiceID(serviceID utils.EntityID, monType, name string) utils.EntityID {
	return NewMonitorID(
		serviceID.Labels["instance"],
		serviceID.Labels["name"],
		monType,
		name,
	)
}

func (c *MonitorConfig) Valid(ctx context.Context) map[string]string {
	problems := make(map[string]string, 3)

	err := utils.CheckName(c.Name)
	if err != nil {
		problems["name"] = err.Error()
	}

	if len(c.Type) == 0 {
		problems["type"] = "'type' is required"
	} else {
		// TODO: Replace with modules
		monitorTypes := []string{"tcp"}
		exists := false
		for _, mtype := range monitorTypes {
			if mtype == c.Type {
				exists = true
			}
		}

		if !exists {
			problems["type"] = fmt.Sprintf("'%s' type does not exist", c.Type)
		}
	}

	if c.Interval == 0 {
		problems["interval"] = "interval should be more than zero"
	}

	return problems
}

type monitorInstance struct {
	mon Monitor
	cfg MonitorConfig

	ctx     context.Context
	cancel  context.CancelFunc
	running bool
}

type MonitorService struct {
	heartbeatRepo HeartbeatRepo

	mu sync.RWMutex
	// Monitor ID to monitor instance
	monitors map[string]*monitorInstance
	// Service ID to a list of monitor ids
	services map[string][]string

	wg sync.WaitGroup

	ctx    context.Context
	cancel context.CancelFunc
}

func NewMonitorService(hRepo HeartbeatRepo) *MonitorService {
	ctx, cancel := context.WithCancel(context.Background())
	return &MonitorService{
		heartbeatRepo: hRepo,
		monitors:      make(map[string]*monitorInstance),
		ctx:           ctx,
		cancel:        cancel,
	}
}

func (m *MonitorService) DiffMonitors(serviceID utils.EntityID, rawConfigs []json.RawMessage) (*ConfigDiff, error) {
	add := make([]string, 10)
	update := make([]string, 10)
	delete := make([]string, 10)

	newConfigs := make(map[string]MonitorConfig)
	for _, rawCfg := range rawConfigs {
		var cfg MonitorConfig
		err := json.Unmarshal(rawCfg, &cfg)
		if err != nil {
			return nil, err
		}

		problems := cfg.Valid(context.TODO())
		if len(problems) > 0 {
			return nil, NewValidationError(problems, serviceID.Labels["name"], cfg.Name)
		}

		id := NewMonitorIDFromServiceID(serviceID, cfg.Type, cfg.Name)

		oldCfg, ok := m.monitors[id.Canonical()]
		if !ok {
			add = append(add, id.Canonical())
			continue
		}

		ok, err = oldCfg.mon.Eq(rawCfg)
		if err != nil {
			return nil, err
		}
		if !ok {
			update = append(update, id.Canonical())
		}
		newConfigs[id.Canonical()] = cfg
	}

	for id := range m.monitors {
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

func (m *MonitorService) LoadService(serviceID utils.EntityID, rawConfigs []json.RawMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	newMonitors, err := m.buildMonitors(serviceID, rawConfigs)
	if err != nil {
		return err
	}

	for _, id := range m.services[serviceID.Canonical()] {
		_, ok := newMonitors[id]
		if !ok {
			m.stopMonitorUnsynced(id)
			delete(m.monitors, id)
		}
	}

	for id, newMonitor := range newMonitors {
		oldMonitor, ok := m.monitors[id]
		if ok && oldMonitor.running {
			m.stopMonitorUnsynced(id)
		}

		ctx, cancel := context.WithCancel(m.ctx)

		newMonitor.ctx = ctx
		newMonitor.cancel = cancel

		m.monitors[id] = newMonitor

		m.startMonitorUnsynced(id)
	}

	return nil
}

func (m *MonitorService) buildMonitors(serviceID utils.EntityID, rawConfigs []json.RawMessage) (map[string]*monitorInstance, error) {
	result := make(map[string]*monitorInstance)

	for i, rawMonitorCfg := range rawConfigs {
		var monitorCfg MonitorConfig
		err := json.Unmarshal(rawMonitorCfg, &monitorCfg)
		if err != nil {
			return nil, err
		}

		// TODO: rewrite to return path
		problems := monitorCfg.Valid(context.TODO())
		if len(problems) > 0 {
			path := make([]string, 0, 2)
			if len(monitorCfg.Name) == 0 {
				path = append(path, fmt.Sprintf("%s[%d]", serviceID.Labels["name"], i))
			} else {
				path = append(path, serviceID.Labels["name"])
				path = append(path, monitorCfg.Name)
			}
			return nil, NewValidationError(problems, path...)
		}

		monitorID := NewMonitorIDFromServiceID(serviceID, monitorCfg.Type, monitorCfg.Name)

		// TODO: Replace with modules
		var monitor Monitor
		switch monitorCfg.Type {
		case "tcp":
			monitor = &TCPMonitor{}
		default:
			return nil, fmt.Errorf("unknown monitor type: %s", monitorCfg.Type)
		}

		err = monitor.Configure(monitorID, rawMonitorCfg)
		if err != nil {
			return nil, err
		}

		result[monitorID.Canonical()] = &monitorInstance{
			mon: monitor,
			cfg: monitorCfg,
		}
	}

	return result, nil
}

func (m *MonitorService) startMonitorUnsynced(monitorID string) {
	m.wg.Add(1)
	go func() {
		m.runMonitor(monitorID)
		m.wg.Done()
	}()
}

func (m *MonitorService) stopMonitorUnsynced(monitorID string) {
	monitor := m.monitors[monitorID]
	if !monitor.running {
		return
	}
	monitor.cancel()
	monitor.running = false
}

func (m *MonitorService) runMonitor(monitorID string) {
	m.mu.RLock()
	interval := m.monitors[monitorID].cfg.Interval
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	monitor := m.monitors[monitorID]
	m.mu.RUnlock()

	for {
		select {
		case <-ticker.C:
			err := monitor.mon.Check(monitor.ctx)
			if errors.Is(err, context.Canceled) {
				continue
			}
			m.heartbeatRepo.InsertHeartbeat(monitor.ctx, Heartbeat{
				MonitorID: monitorID,
				Timestamp: time.Now(),
				Error:     err,
			})
		case <-monitor.ctx.Done():
			return
		}
	}
}

func (m *MonitorService) Stop(ctx context.Context) error {
	m.cancel()

	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

type TCPConfig struct {
	Hostname string `json:"hostname"`
	Port     string `json:"port"`
	Timeout  int    `json:"timeout"`
}

var HostnameRegex = regexp.MustCompile(`^(([a-zA-Z]|[a-zA-Z][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z]|[A-Za-z][A-Za-z0-9\-]*[A-Za-z0-9])$`)
var IPRegex = regexp.MustCompile(`^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$`)

func (c *TCPConfig) Valid(ctx context.Context) map[string]string {
	problems := make(map[string]string, 3)
	if !HostnameRegex.MatchString(c.Hostname) || IPRegex.MatchString(c.Hostname) {
		problems["hostname"] = "invalid hostname or ip addres"
	}

	numPort, err := strconv.Atoi(c.Port)
	if err != nil {
		problems["port"] = fmt.Sprint("port should be a valid number: ", err)
	}

	if numPort < 0 {
		problems["port"] = "cannot be less than zero"
	}

	if numPort > 65535 {
		problems["port"] = "cannot be greater than 65,535"
	}

	if c.Timeout < 0 {
		problems["timeout"] = "cannot be less than zero"
	}

	return problems
}

type TCPMonitor struct {
	ID  utils.EntityID
	cfg TCPConfig
}

func (m *TCPMonitor) Check(parentCtx context.Context) error {
	ctx, cancel := context.WithTimeout(parentCtx, time.Duration(m.cfg.Timeout)*time.Second)
	defer cancel()
	return PingTCP(ctx, m.cfg.Hostname, m.cfg.Port)
}

func PingTCP(ctx context.Context, hostname string, port string) error {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(hostname, port))
	if err != nil {
		return err
	}
	defer conn.Close()

	return nil
}

func (m *TCPMonitor) Configure(id utils.EntityID, rawCfg []byte) error {
	var cfg TCPConfig
	err := json.Unmarshal(rawCfg, &cfg)
	if err != nil {
		return err
	}

	problems := cfg.Valid(context.TODO())
	if len(problems) > 0 {
		return NewValidationError(problems, id.Labels["name"])
	}

	m.ID = id
	m.cfg = cfg
	return nil
}

func (m *TCPMonitor) Eq(newRawCfg []byte) (bool, error) {
	var newCfg TCPConfig
	err := json.Unmarshal(newRawCfg, &newCfg)
	if err != nil {
		return false, err
	}

	return m.cfg == newCfg, nil
}
