package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"time"

	"meerkat-v0/pkg/utils"
	"meerkat-v0/internal/shared/validation"
)

var (
	HostnameRegex = regexp.MustCompile(`^(([a-zA-Z]|[a-zA-Z][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z]|[A-Za-z][A-Za-z0-9\-]*[A-Za-z0-9])$`)
	IPRegex       = regexp.MustCompile(`^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$`)
)

// TCPConfig represents TCP monitor configuration
type TCPConfig struct {
	Hostname string `json:"hostname"`
	Port     string `json:"port"`
	Timeout  int    `json:"timeout"`
}

func (c *TCPConfig) Valid(ctx context.Context) map[string]string {
	problems := make(map[string]string, 3)
	if !HostnameRegex.MatchString(c.Hostname) && !IPRegex.MatchString(c.Hostname) {
		problems["hostname"] = "invalid hostname or ip address"
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

// TCPMonitor is a domain service for TCP monitoring
type TCPMonitor struct {
	ID  utils.EntityID
	cfg TCPConfig
}

// Run executes a TCP check
func (m *TCPMonitor) Run(parentCtx context.Context) error {
	ctx, cancel := context.WithTimeout(parentCtx, time.Duration(m.cfg.Timeout)*time.Millisecond)
	defer cancel()
	return PingTCP(ctx, m.cfg.Hostname, m.cfg.Port)
}

// PingTCP performs a TCP connection check
func PingTCP(ctx context.Context, hostname string, port string) error {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(hostname, port))
	if err != nil {
		return err
	}
	defer conn.Close()
	return nil
}

// Configure configures the TCP monitor with the given ID and raw config
func (m *TCPMonitor) Configure(id utils.EntityID, rawCfg []byte) error {
	var cfg TCPConfig
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
	return nil
}

// Eq checks if the new config equals the current config
func (m *TCPMonitor) Eq(newRawCfg []byte) (bool, error) {
	var newCfg TCPConfig
	err := json.Unmarshal(newRawCfg, &newCfg)
	if err != nil {
		return false, err
	}

	return m.cfg == newCfg, nil
}

