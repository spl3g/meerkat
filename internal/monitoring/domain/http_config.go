package domain

import (
	"context"
	"fmt"
	"strings"
)

// HTTPConfig represents HTTP monitor configuration
type HTTPConfig struct {
	URL            string `json:"url"`
	Method         string `json:"method"`
	Timeout        int    `json:"timeout"`
	ExpectedStatus int    `json:"expectedStatus"`
}

// Valid validates the HTTP configuration and returns any problems found
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

