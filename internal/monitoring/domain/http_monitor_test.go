package domain

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"meerkat-v0/pkg/utils"
	"meerkat-v0/internal/shared/validation"
)

func TestHTTPConfig_Valid(t *testing.T) {
	tests := []struct {
		name      string
		config    HTTPConfig
		wantError bool
		wantField string
	}{
		{
			name: "valid config with all fields",
			config: HTTPConfig{
				URL:            "https://example.com",
				Method:         "GET",
				Timeout:        5000,
				ExpectedStatus: 200,
			},
			wantError: false,
		},
		{
			name: "valid config with default method",
			config: HTTPConfig{
				URL:     "http://example.com",
				Timeout: 1000,
			},
			wantError: false,
		},
		{
			name: "valid config with status range (no ExpectedStatus)",
			config: HTTPConfig{
				URL:     "https://example.com",
				Method:  "POST",
				Timeout: 3000,
			},
			wantError: false,
		},
		{
			name: "empty URL",
			config: HTTPConfig{
				URL:     "",
				Timeout: 1000,
			},
			wantError: true,
			wantField: "url",
		},
		{
			name: "invalid URL - no protocol",
			config: HTTPConfig{
				URL:     "example.com",
				Timeout: 1000,
			},
			wantError: true,
			wantField: "url",
		},
		{
			name: "invalid URL - wrong protocol",
			config: HTTPConfig{
				URL:     "ftp://example.com",
				Timeout: 1000,
			},
			wantError: true,
			wantField: "url",
		},
		{
			name: "invalid method",
			config: HTTPConfig{
				URL:     "https://example.com",
				Method:  "INVALID",
				Timeout: 1000,
			},
			wantError: true,
			wantField: "method",
		},
		{
			name: "negative timeout",
			config: HTTPConfig{
				URL:     "https://example.com",
				Timeout: -1,
			},
			wantError: true,
			wantField: "timeout",
		},
		{
			name: "invalid expected status - too low",
			config: HTTPConfig{
				URL:            "https://example.com",
				Timeout:        1000,
				ExpectedStatus: 99,
			},
			wantError: true,
			wantField: "expectedStatus",
		},
		{
			name: "invalid expected status - too high",
			config: HTTPConfig{
				URL:            "https://example.com",
				Timeout:        1000,
				ExpectedStatus: 600,
			},
			wantError: true,
			wantField: "expectedStatus",
		},
		{
			name: "valid expected status - 200",
			config: HTTPConfig{
				URL:            "https://example.com",
				Timeout:        1000,
				ExpectedStatus: 200,
			},
			wantError: false,
		},
		{
			name: "valid expected status - 404",
			config: HTTPConfig{
				URL:            "https://example.com",
				Timeout:        1000,
				ExpectedStatus: 404,
			},
			wantError: false,
		},
		{
			name: "valid expected status - 500",
			config: HTTPConfig{
				URL:            "https://example.com",
				Timeout:        1000,
				ExpectedStatus: 500,
			},
			wantError: false,
		},
		{
			name: "method normalization - lowercase",
			config: HTTPConfig{
				URL:     "https://example.com",
				Method:  "get",
				Timeout: 1000,
			},
			wantError: false,
		},
		{
			name: "method normalization - mixed case",
			config: HTTPConfig{
				URL:     "https://example.com",
				Method:  "Post",
				Timeout: 1000,
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			problems := tt.config.Valid(context.Background())

			if tt.wantError {
				if len(problems) == 0 {
					t.Errorf("expected validation error, got none")
					return
				}
				if tt.wantField != "" {
					if _, ok := problems[tt.wantField]; !ok {
						t.Errorf("expected error for field %q, got problems: %v", tt.wantField, problems)
					}
				}
			} else {
				if len(problems) > 0 {
					t.Errorf("unexpected validation errors: %v", problems)
				}
				// Check method normalization - should be uppercase if set
				if tt.config.Method != "" {
					// Method should be normalized to uppercase by Valid()
					upperMethod := tt.config.Method
					for _, r := range upperMethod {
						if r >= 'a' && r <= 'z' {
							t.Errorf("expected method to be normalized to uppercase, got %q", tt.config.Method)
							break
						}
					}
				}
			}
		})
	}
}

func TestHTTPMonitor_Configure(t *testing.T) {
	tests := []struct {
		name      string
		rawCfg    string
		wantError bool
		checkID   bool
	}{
		{
			name: "valid configuration",
			rawCfg: `{
				"url": "https://example.com",
				"method": "GET",
				"timeout": 5000,
				"expectedStatus": 200
			}`,
			wantError: false,
			checkID:   true,
		},
		{
			name: "valid configuration with default method",
			rawCfg: `{
				"url": "http://example.com",
				"timeout": 1000
			}`,
			wantError: false,
			checkID:   true,
		},
		{
			name: "invalid JSON",
			rawCfg: `{
				"url": "https://example.com",
				"timeout": 1000
			`,
			wantError: true,
			checkID:   false,
		},
		{
			name: "invalid configuration - empty URL",
			rawCfg: `{
				"url": "",
				"timeout": 1000
			}`,
			wantError: true,
		},
		{
			name: "invalid configuration - negative timeout",
			rawCfg: `{
				"url": "https://example.com",
				"timeout": -1
			}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor := &HTTPMonitor{}
			id := utils.EntityID{
				Kind: "monitor",
				Labels: map[string]string{
					"name": "test-monitor",
				},
			}

			err := monitor.Configure(id, []byte(tt.rawCfg))

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}
				// For invalid JSON, we expect a JSON error, not a validation error
				if tt.name == "invalid JSON" {
					// JSON syntax errors are acceptable
					return
				}
				var validationErr *validation.ValidationError
				if !errors.As(err, &validationErr) {
					t.Errorf("expected ValidationError, got %T: %v", err, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if tt.checkID {
					if monitor.ID.Kind != id.Kind {
						t.Errorf("expected ID.Kind %q, got %q", id.Kind, monitor.ID.Kind)
					}
					if monitor.cfg.URL == "" {
						t.Errorf("expected URL to be set")
					}
				}
			}
		})
	}
}

func TestHTTPMonitor_Run(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		expectedStatus int
		wantError      bool
		serverDelay    time.Duration
		timeout        int
	}{
		{
			name:           "successful request - 200 with default range",
			statusCode:     200,
			expectedStatus: 0, // Use default range
			wantError:      false,
			timeout:        5000,
		},
		{
			name:           "successful request - 201 with default range",
			statusCode:     201,
			expectedStatus: 0,
			wantError:      false,
			timeout:        5000,
		},
		{
			name:           "successful request - 299 with default range",
			statusCode:     299,
			expectedStatus: 0,
			wantError:      false,
			timeout:        5000,
		},
		{
			name:           "successful request - exact match 200",
			statusCode:     200,
			expectedStatus: 200,
			wantError:      false,
			timeout:        5000,
		},
		{
			name:           "successful request - exact match 404",
			statusCode:     404,
			expectedStatus: 404,
			wantError:      false,
			timeout:        5000,
		},
		{
			name:           "error - 404 with default range",
			statusCode:     404,
			expectedStatus: 0,
			wantError:      true,
			timeout:        5000,
		},
		{
			name:           "error - 500 with default range",
			statusCode:     500,
			expectedStatus: 0,
			wantError:      true,
			timeout:        5000,
		},
		{
			name:           "error - wrong status code",
			statusCode:     200,
			expectedStatus: 201,
			wantError:      true,
			timeout:        5000,
		},
		{
			name:           "error - timeout",
			statusCode:     200,
			expectedStatus: 0,
			wantError:      true,
			serverDelay:    100 * time.Millisecond,
			timeout:        50, // 50ms timeout
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.serverDelay > 0 {
					time.Sleep(tt.serverDelay)
				}
				w.WriteHeader(tt.statusCode)
				w.Write([]byte("OK"))
			}))
			defer server.Close()

			monitor := &HTTPMonitor{
				cfg: HTTPConfig{
					URL:            server.URL,
					Method:         "GET",
					Timeout:        tt.timeout,
					ExpectedStatus: tt.expectedStatus,
				},
			}

			ctx := context.Background()
			err := monitor.Run(ctx)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestPingHTTP(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		statusCode     int
		expectedStatus int
		wantError      bool
	}{
		{
			name:           "GET request - success",
			method:         "GET",
			statusCode:     200,
			expectedStatus: 0,
			wantError:      false,
		},
		{
			name:           "POST request - success",
			method:         "POST",
			statusCode:     201,
			expectedStatus: 0,
			wantError:      false,
		},
		{
			name:           "HEAD request - success",
			method:         "HEAD",
			statusCode:     200,
			expectedStatus: 0,
			wantError:      false,
		},
		{
			name:           "exact status match - 404",
			method:         "GET",
			statusCode:     404,
			expectedStatus: 404,
			wantError:      false,
		},
		{
			name:           "status out of range",
			method:         "GET",
			statusCode:     404,
			expectedStatus: 0,
			wantError:      true,
		},
		{
			name:           "wrong exact status",
			method:         "GET",
			statusCode:     200,
			expectedStatus: 201,
			wantError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tt.method {
					t.Errorf("expected method %q, got %q", tt.method, r.Method)
				}
				w.WriteHeader(tt.statusCode)
				w.Write([]byte("OK"))
			}))
			defer server.Close()

			ctx := context.Background()
			err := PingHTTP(ctx, server.URL, tt.method, 5*time.Second, tt.expectedStatus)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestHTTPMonitor_Eq(t *testing.T) {
	tests := []struct {
		name     string
		cfg1     HTTPConfig
		cfg2     string
		wantEq   bool
		wantErr  bool
	}{
		{
			name: "identical configs",
			cfg1: HTTPConfig{
				URL:            "https://example.com",
				Method:         "GET",
				Timeout:        5000,
				ExpectedStatus: 200,
			},
			cfg2: `{
				"url": "https://example.com",
				"method": "GET",
				"timeout": 5000,
				"expectedStatus": 200
			}`,
			wantEq:  true,
			wantErr: false,
		},
		{
			name: "different URLs",
			cfg1: HTTPConfig{
				URL:     "https://example.com",
				Method:  "GET",
				Timeout: 5000,
			},
			cfg2: `{
				"url": "https://other.com",
				"method": "GET",
				"timeout": 5000
			}`,
			wantEq:  false,
			wantErr: false,
		},
		{
			name: "different methods",
			cfg1: HTTPConfig{
				URL:     "https://example.com",
				Method:  "GET",
				Timeout: 5000,
			},
			cfg2: `{
				"url": "https://example.com",
				"method": "POST",
				"timeout": 5000
			}`,
			wantEq:  false,
			wantErr: false,
		},
		{
			name: "different timeouts",
			cfg1: HTTPConfig{
				URL:     "https://example.com",
				Method:  "GET",
				Timeout: 5000,
			},
			cfg2: `{
				"url": "https://example.com",
				"method": "GET",
				"timeout": 1000
			}`,
			wantEq:  false,
			wantErr: false,
		},
		{
			name: "different expected status",
			cfg1: HTTPConfig{
				URL:            "https://example.com",
				Method:         "GET",
				Timeout:        5000,
				ExpectedStatus: 200,
			},
			cfg2: `{
				"url": "https://example.com",
				"method": "GET",
				"timeout": 5000,
				"expectedStatus": 201
			}`,
			wantEq:  false,
			wantErr: false,
		},
		{
			name: "method normalization - empty vs GET",
			cfg1: HTTPConfig{
				URL:     "https://example.com",
				Method:  "GET",
				Timeout: 5000,
			},
			cfg2: `{
				"url": "https://example.com",
				"timeout": 5000
			}`,
			wantEq:  true,
			wantErr: false,
		},
		{
			name: "method normalization - lowercase",
			cfg1: HTTPConfig{
				URL:     "https://example.com",
				Method:  "GET",
				Timeout: 5000,
			},
			cfg2: `{
				"url": "https://example.com",
				"method": "get",
				"timeout": 5000
			}`,
			wantEq:  true,
			wantErr: false,
		},
		{
			name: "invalid JSON",
			cfg1: HTTPConfig{
				URL:     "https://example.com",
				Method:  "GET",
				Timeout: 5000,
			},
			cfg2: `{
				"url": "https://example.com",
				"timeout": 5000
			`,
			wantEq:  false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor := &HTTPMonitor{
				cfg: tt.cfg1,
			}

			eq, err := monitor.Eq([]byte(tt.cfg2))

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if eq != tt.wantEq {
					t.Errorf("expected Eq to return %v, got %v", tt.wantEq, eq)
				}
			}
		})
	}
}

func TestPingHTTP_InvalidURL(t *testing.T) {
	ctx := context.Background()
	err := PingHTTP(ctx, "not-a-valid-url", "GET", 5*time.Second, 0)
	if err == nil {
		t.Error("expected error for invalid URL, got nil")
	}
}

func TestPingHTTP_ContextCancellation(t *testing.T) {
	// Create a server that delays
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(200)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := PingHTTP(ctx, server.URL, "GET", 5*time.Second, 0)
	if err == nil {
		t.Error("expected error for cancelled context, got nil")
	}
}

