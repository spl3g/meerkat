package infrastructure

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"meerkat-v0/internal/monitoring/domain"
)

// HTTPClientImpl implements the domain HTTPClient interface
type HTTPClientImpl struct{}

// NewHTTPClient creates a new HTTP client implementation
func NewHTTPClient() domain.HTTPClient {
	return &HTTPClientImpl{}
}

// Do performs an HTTP request with the given method, URL, timeout, and expected status
func (c *HTTPClientImpl) Do(ctx context.Context, method, url string, timeout time.Duration, expectedStatus int) error {
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


