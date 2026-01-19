package domain

import (
	"context"
	"time"
)

// HTTPClient defines the interface for HTTP client operations
// This interface abstracts HTTP concerns from the domain layer
type HTTPClient interface {
	// Do performs an HTTP request with the given method, URL, timeout, and expected status
	// Returns an error if the request fails or the status code doesn't match expectations
	Do(ctx context.Context, method, url string, timeout time.Duration, expectedStatus int) error
}


