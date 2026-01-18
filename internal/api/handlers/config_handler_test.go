package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	configapp "meerkat-v0/internal/config/application"
	api "meerkat-v0/internal/api/application"
	monitoringapp "meerkat-v0/internal/monitoring/application"
	metricsapp "meerkat-v0/internal/metrics/application"
	"meerkat-v0/internal/infrastructure/logger"
	monitoringinfra "meerkat-v0/internal/monitoring/infrastructure"
	metricsinfra "meerkat-v0/internal/metrics/infrastructure"
	entityinfra "meerkat-v0/internal/shared/entity/infrastructure"
	"meerkat-v0/internal/infrastructure/database/queries"
	"meerkat-v0/internal/infrastructure/database"
)

func setupTestConfigHandler(t *testing.T) (*ConfigHandler, func()) {
	// Setup in-memory database
	testDB, err := database.ConnectSQLite(":memory:")
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	q := queries.New(testDB)
	entityRepo := entityinfra.NewRepository(q, q)
	monitorRepo := monitoringinfra.NewRepository(q, q, testDB, testDB, entityRepo)
	metricsRepo := metricsinfra.NewRepository(q, q, testDB, testDB, entityRepo)

	logger := logger.DefaultLogger()
	monitorService := monitoringapp.NewService(logger, monitorRepo, entityRepo)
	metricsService := metricsapp.NewService(logger, metricsRepo, entityRepo)
	configLoader := configapp.NewLoader(logger, monitorService, metricsService)

	handler := NewConfigHandler(configLoader)

	cleanup := func() {
		testDB.Close()
	}

	return handler, cleanup
}

func TestConfigHandler_LoadConfig(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:   "valid config",
			method: http.MethodPost,
			body: api.LoadConfigRequest{
				Config: json.RawMessage(`{
					"name": "test-instance",
					"services": [
						{
							"name": "test-service"
						}
					]
				}`),
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "invalid JSON",
			method: http.MethodPost,
			body:   "not json",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request body",
		},
		{
			name:   "missing config field",
			method: http.MethodPost,
			body: map[string]interface{}{
				"wrong": "field",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Failed to load config",
		},
		{
			name:   "invalid config validation",
			method: http.MethodPost,
			body: api.LoadConfigRequest{
				Config: json.RawMessage(`{
					"name": "",
					"services": []
				}`),
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Failed to load config",
		},
		{
			name:           "method not allowed - GET",
			method:         http.MethodGet,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "method not allowed - PUT",
			method:         http.MethodPut,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "method not allowed - DELETE",
			method:         http.MethodDelete,
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, cleanup := setupTestConfigHandler(t)
			defer cleanup()

			var bodyBytes []byte
			var err error
			if tt.body != nil {
				bodyBytes, err = json.Marshal(tt.body)
				if err != nil {
					bodyBytes = []byte(tt.body.(string))
				}
			}

			req := httptest.NewRequest(tt.method, "/api/v1/config", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.LoadConfig(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedError != "" {
				var errorResp api.ErrorResponse
				if err := json.NewDecoder(w.Body).Decode(&errorResp); err == nil {
					if !contains(errorResp.Error, tt.expectedError) {
						t.Errorf("expected error to contain %q, got %q", tt.expectedError, errorResp.Error)
					}
				}
			}

			if tt.expectedStatus == http.StatusOK {
				var resp map[string]string
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Errorf("failed to decode response: %v", err)
				}
				if resp["status"] != "ok" {
					t.Errorf("expected status 'ok', got %q", resp["status"])
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

