package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"time"

	"meerkat-v0/utils"
)

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

func BuildMonitor(serviceID utils.EntityID, rawCfg []byte) (utils.EntityID, *EntityInstance, error) {
	var id utils.EntityID
	var cfg EntityConfig
	err := json.Unmarshal(rawCfg, &cfg)
	if err != nil {
		return id, nil, err
	}

	problems := cfg.Valid(context.TODO())
	if len(problems) > 0 {
		return id, nil, NewValidationError(problems, serviceID.Labels["name"], cfg.Name)
	}

	id = NewMonitorIDFromServiceID(serviceID, cfg.Type, cfg.Name)

	// TODO: Replace with modules
	var entity Entity
	switch cfg.Type {
	case "cpu":
		entity = &TCPMonitor{}
	default:
		return id, nil, fmt.Errorf("unknown monitor type: %s", cfg.Type)
	}

	err = entity.Configure(id, rawCfg)
	if err != nil {
		return id, nil, err
	}

	return id, NewEntityInstance(id, entity, cfg, rawCfg), nil
}

func RunMonitor(heartbeatRepo HeartbeatRepo, logger *utils.Logger, inst *EntityInstance) {
	interval := inst.Cfg.Interval
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := inst.Ent.Run(inst.ctx)
			if errors.Is(err, context.Canceled) {
				continue
			}
			heartbeatRepo.InsertHeartbeat(inst.ctx, Heartbeat{
				MonitorID: inst.ID.Canonical(),
				Timestamp: time.Now(),
				Error:     err,
			})
		case <-inst.ctx.Done():
			return
		}
	}
}

func (m *EntityService) Stop(ctx context.Context) error {
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

func (m *TCPMonitor) Run(parentCtx context.Context) error {
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
