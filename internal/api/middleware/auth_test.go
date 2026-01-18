package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestAPIKeyAuth(t *testing.T) {
	tests := []struct {
		name           string
		envKey         string
		headerKey      string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "valid API key",
			envKey:         "test-api-key",
			headerKey:      "test-api-key",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing API key header",
			envKey:         "test-api-key",
			headerKey:      "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Invalid or missing API key",
		},
		{
			name:           "invalid API key",
			envKey:         "test-api-key",
			headerKey:      "wrong-key",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Invalid or missing API key",
		},
		{
			name:           "missing environment variable",
			envKey:         "",
			headerKey:      "any-key",
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "API key not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original value
			originalKey := os.Getenv("MEERKAT_API_KEY")
			defer os.Setenv("MEERKAT_API_KEY", originalKey)

			// Set test environment variable
			if tt.envKey != "" {
				os.Setenv("MEERKAT_API_KEY", tt.envKey)
			} else {
				os.Unsetenv("MEERKAT_API_KEY")
			}

			// Create a test handler that returns 200 OK
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})

			// Create middleware
			handler := APIKeyAuth(nextHandler)

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
			if tt.headerKey != "" {
				req.Header.Set("X-API-Key", tt.headerKey)
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Execute middleware
			handler.ServeHTTP(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check response body if expected
			if tt.expectedBody != "" {
				body := w.Body.String()
				if !contains(body, tt.expectedBody) {
					t.Errorf("expected body to contain %q, got %q", tt.expectedBody, body)
				}
			}

			// If status is OK, verify the next handler was called
			if tt.expectedStatus == http.StatusOK {
				if w.Body.String() != "OK" {
					t.Errorf("expected next handler to be called, got body: %q", w.Body.String())
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

