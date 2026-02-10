package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"meerkat-v0/pkg/utils"
	"meerkat-v0/internal/shared/validation"
)

// HTTPConfig represents HTTP monitor configuration
type HTTPConfig struct {
	URL            string `json:"url"`
	Method         string `json:"method"`
	Timeout        int    `json:"timeout"`
	ExpectedStatus int    `json:"expectedStatus"`
}

func (c *HTTPConfig) Valid(ctx context.Context) map[string]string {
	problems := make(map[string]string, 4)

	// Validate URL
	if len(c.URL) == 0 {
		problems["url"] = "url is required"
	} else {
		if !strings.HasPrefix(c.URL, "http://") && !strings.HasPrefix(c.URL, "https://") {
			problems["url"] = "url must start with http:// or https://"
		}
	}

	// Validate method (default to GET if empty)
	if len(c.Method) == 0 {
		c.Method = "GET"
	} else {
		validMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
		isValid := false
		for _, m := range validMethods {
			if strings.ToUpper(c.Method) == m {
				isValid = true
				break
			}
		}
		if !isValid {
			problems["method"] = fmt.Sprintf("invalid HTTP method: %s", c.Method)
		} else {
			// Normalize to uppercase
			c.Method = strings.ToUpper(c.Method)
		}
	}

	// Validate timeout
	if c.Timeout < 0 {
		problems["timeout"] = "cannot be less than zero"
	}

	// Validate expected status code (if provided)
	if c.ExpectedStatus != 0 {
		if c.ExpectedStatus < 100 || c.ExpectedStatus > 599 {
			problems["expectedStatus"] = "must be a valid HTTP status code (100-599)"
		}
	}

	return problems
}

// HTTPMonitor is a domain service for HTTP monitoring
type HTTPMonitor struct {
	ID  utils.EntityID
	cfg HTTPConfig
}

// Run executes an HTTP check
func (m *HTTPMonitor) Run(parentCtx context.Context) error {
	timeout := time.Duration(m.cfg.Timeout) * time.Millisecond
	return PingHTTP(parentCtx, m.cfg.URL, m.cfg.Method, timeout, m.cfg.ExpectedStatus)
}

// PingHTTP performs an HTTP request check
func PingHTTP(ctx context.Context, url, method string, timeout time.Duration, expectedStatus int) error {
	// Create context with timeout
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create HTTP request
	req, err := http.NewRequestWithContext(reqCtx, method, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: timeout,
	}

	// Perform request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Validate status code
	if expectedStatus != 0 {
		// Exact match required
		if resp.StatusCode != expectedStatus {
			return fmt.Errorf("expected status %d, got %d", expectedStatus, resp.StatusCode)
		}
	} else {
		// Default: check for 200-299 range
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return fmt.Errorf("expected status in 200-299 range, got %d", resp.StatusCode)
		}
	}

	return nil
}

// Configure configures the HTTP monitor with the given ID and raw config
func (m *HTTPMonitor) Configure(id utils.EntityID, rawCfg []byte) error {
	var cfg HTTPConfig
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
func (m *HTTPMonitor) Eq(newRawCfg []byte) (bool, error) {
	var newCfg HTTPConfig
	err := json.Unmarshal(newRawCfg, &newCfg)
	if err != nil {
		return false, err
	}

	// Normalize method for comparison
	if len(newCfg.Method) == 0 {
		newCfg.Method = "GET"
	} else {
		newCfg.Method = strings.ToUpper(newCfg.Method)
	}

	return m.cfg == newCfg, nil
}

