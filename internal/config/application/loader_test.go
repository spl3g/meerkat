package application

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"meerkat-v0/internal/infrastructure/database"
	"meerkat-v0/internal/infrastructure/logger"
	monitoringapp "meerkat-v0/internal/monitoring/application"
	monitoringinfra "meerkat-v0/internal/monitoring/infrastructure"
	metricsapp "meerkat-v0/internal/metrics/application"
	metricsinfra "meerkat-v0/internal/metrics/infrastructure"
	entityinfra "meerkat-v0/internal/shared/entity/infrastructure"
	"meerkat-v0/db"
	"meerkat-v0/internal/schema"
	"meerkat-v0/internal/shared/validation"
)

func setupTestLoader(t *testing.T) (*Loader, func()) {
	// Setup in-memory database
	testDB, err := database.ConnectSQLite(":memory:")
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Initialize schema
	_, err = testDB.Exec(schema.DDL)
	if err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	queries := db.New(testDB)
	entityRepo := entityinfra.NewRepository(queries, queries)
	monitorRepo := monitoringinfra.NewRepository(queries, queries, testDB, entityRepo)
	metricsRepo := metricsinfra.NewRepository(queries, queries, testDB, entityRepo)

	logger := logger.DefaultLogger()
	monitorService := monitoringapp.NewService(logger, monitorRepo, entityRepo)
	metricsService := metricsapp.NewService(logger, metricsRepo, entityRepo)
	loader := NewLoader(logger, monitorService, metricsService)

	cleanup := func() {
		testDB.Close()
	}

	return loader, cleanup
}

func TestLoader_LoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		expectError bool
		errorType   string
	}{
		{
			name: "valid config with service",
			config: `{
				"name": "test-instance",
				"services": [
					{
						"name": "test-service"
					}
				]
			}`,
			expectError: false,
		},
		{
			name: "invalid JSON",
			config: `{
				"name": "test-instance",
				"services": [
					{
						"name": "test-service"
					}
				]
			`,
			expectError: true,
			errorType:   "parse",
		},
		{
			name: "config validation error - empty name",
			config: `{
				"name": "",
				"services": [
					{
						"name": "test-service"
					}
				]
			}`,
			expectError: true,
			errorType:   "validation",
		},
		{
			name: "config validation error - empty services",
			config: `{
				"name": "test-instance",
				"services": []
			}`,
			expectError: true,
			errorType:   "validation",
		},
		{
			name: "missing service name",
			config: `{
				"name": "test-instance",
				"services": [
					{}
				]
			}`,
			expectError: true,
			errorType:   "validation",
		},
		{
			name: "valid config with monitors",
			config: `{
				"name": "test-instance",
				"services": [
					{
						"name": "test-service",
						"monitors": [
							{
								"type": "tcp",
								"name": "tcp-check",
								"interval": 60,
								"hostname": "localhost",
								"port": "8080",
								"timeout": 5000
							}
						]
					}
				]
			}`,
			expectError: false,
		},
		{
			name: "valid config with metrics",
			config: `{
				"name": "test-instance",
				"services": [
					{
						"name": "test-service",
						"metrics": [
							{
								"type": "cpu",
								"name": "cpu-usage",
								"interval": 30
							}
						]
					}
				]
			}`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader, cleanup := setupTestLoader(t)
			defer cleanup()

			rawConfig := []byte(tt.config)
			err := loader.LoadConfig(context.Background(), rawConfig)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}

				if tt.errorType == "parse" {
					if err.Error()[:len("failed to parse config")] != "failed to parse config" {
						t.Errorf("expected parse error, got: %v", err)
					}
				} else if tt.errorType == "validation" {
					var valErr validation.ConfigError
					if !errors.As(err, &valErr) {
						t.Errorf("expected validation error, got: %v", err)
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestLoader_LoadConfig_ConcurrentAccess(t *testing.T) {
	loader, cleanup := setupTestLoader(t)
	defer cleanup()

	config := `{
		"name": "test-instance",
		"services": [
			{
				"name": "test-service"
			}
		]
	}`

	// Test concurrent access
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := loader.LoadConfig(context.Background(), []byte(config))
			if err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("unexpected error during concurrent access: %v", err)
	}
}

func TestLoader_Stop(t *testing.T) {
	loader, cleanup := setupTestLoader(t)
	defer cleanup()

	// Load a config with monitors/metrics that will run
	config := `{
		"name": "test-instance",
		"services": [
			{
				"name": "test-service",
				"monitors": [
					{
						"type": "tcp",
						"name": "tcp-check",
						"interval": 1,
						"hostname": "127.0.0.1",
						"port": "9999",
						"timeout": 1000
					}
				]
			}
		]
	}`

	err := loader.LoadConfig(context.Background(), []byte(config))
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Test stop with context
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = loader.Stop(ctx)
	if err != nil {
		t.Errorf("unexpected error stopping loader: %v", err)
	}
}

func TestLoader_Stop_ContextCancellation(t *testing.T) {
	loader, cleanup := setupTestLoader(t)
	defer cleanup()

	// Load a config
	config := `{
		"name": "test-instance",
		"services": [
			{
				"name": "test-service"
			}
		]
	}`

	err := loader.LoadConfig(context.Background(), []byte(config))
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Test stop with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err = loader.Stop(ctx)
	if err == nil {
		t.Error("expected error when context is cancelled, got nil")
	}
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

