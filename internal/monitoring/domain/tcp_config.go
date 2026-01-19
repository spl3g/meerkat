package domain

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
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

// Valid validates the TCP configuration and returns any problems found
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

