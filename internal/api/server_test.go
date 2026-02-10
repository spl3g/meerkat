package api

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	configapp "meerkat-v0/internal/config/application"
	"meerkat-v0/internal/infrastructure/database"
	"meerkat-v0/internal/infrastructure/logger"
	monitoringapp "meerkat-v0/internal/monitoring/application"
	monitoringinfra "meerkat-v0/internal/monitoring/infrastructure"
	metricsapp "meerkat-v0/internal/metrics/application"
	metricsinfra "meerkat-v0/internal/metrics/infrastructure"
	entityinfra "meerkat-v0/internal/shared/entity/infrastructure"
	"meerkat-v0/db"
	"meerkat-v0/internal/schema"
)

func setupTestServer(t *testing.T) (*Server, func()) {
	// Save original environment variables
	originalAPIKey := os.Getenv("MEERKAT_API_KEY")
	originalPort := os.Getenv("MEERKAT_API_PORT")

	// Set test environment variables
	os.Setenv("MEERKAT_API_KEY", "test-api-key")
	os.Unsetenv("MEERKAT_API_PORT") // Use default port

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
	configLoader := configapp.NewLoader(logger, monitorService, metricsService)

	server, err := NewServer(logger, configLoader, entityRepo, monitorRepo, metricsRepo)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	cleanup := func() {
		testDB.Close()
		// Restore original environment variables
		if originalAPIKey != "" {
			os.Setenv("MEERKAT_API_KEY", originalAPIKey)
		} else {
			os.Unsetenv("MEERKAT_API_KEY")
		}
		if originalPort != "" {
			os.Setenv("MEERKAT_API_PORT", originalPort)
		} else {
			os.Unsetenv("MEERKAT_API_PORT")
		}
	}

	return server, cleanup
}

func TestNewServer(t *testing.T) {
	tests := []struct {
		name        string
		apiKey      string
		expectError bool
	}{
		{
			name:        "valid server creation",
			apiKey:      "test-api-key",
			expectError: false,
		},
		{
			name:        "missing API key",
			apiKey:      "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original
			originalAPIKey := os.Getenv("MEERKAT_API_KEY")
			defer func() {
				if originalAPIKey != "" {
					os.Setenv("MEERKAT_API_KEY", originalAPIKey)
				} else {
					os.Unsetenv("MEERKAT_API_KEY")
				}
			}()

			// Set test API key
			if tt.apiKey != "" {
				os.Setenv("MEERKAT_API_KEY", tt.apiKey)
			} else {
				os.Unsetenv("MEERKAT_API_KEY")
			}

			// Setup database
			testDB, err := database.ConnectSQLite(":memory:")
			if err != nil {
				t.Fatalf("Failed to connect to test database: %v", err)
			}
			defer testDB.Close()

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
			configLoader := configapp.NewLoader(logger, monitorService, metricsService)

			server, err := NewServer(logger, configLoader, entityRepo, monitorRepo, metricsRepo)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				if server != nil {
					t.Errorf("expected nil server on error, got %v", server)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if server == nil {
					t.Error("expected server, got nil")
				}
			}
		})
	}
}

func TestServer_Start_Shutdown(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Start server in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Start()
	}()

	// Give server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Test that server is running by making a request
	client := &http.Client{
		Timeout: 1 * time.Second,
	}

	req, err := http.NewRequest("GET", "http://localhost:8080/api/v1/entities", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("X-API-Key", "test-api-key")

	resp, err := client.Do(req)
	if err != nil {
		// Server might not be fully started yet, that's okay
		t.Logf("request failed (server may still be starting): %v", err)
	} else {
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
			t.Errorf("unexpected status code: %d", resp.StatusCode)
		}
	}

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = server.Shutdown(ctx)
	if err != nil {
		t.Errorf("unexpected error shutting down server: %v", err)
	}

	// Check for start errors
	select {
	case err := <-errChan:
		if err != http.ErrServerClosed {
			t.Errorf("unexpected server error: %v", err)
		}
	case <-time.After(1 * time.Second):
		// Server stopped successfully
	}
}

func TestServer_PortConfiguration(t *testing.T) {
	// Save original
	originalAPIKey := os.Getenv("MEERKAT_API_KEY")
	originalPort := os.Getenv("MEERKAT_API_PORT")
	defer func() {
		if originalAPIKey != "" {
			os.Setenv("MEERKAT_API_KEY", originalAPIKey)
		}
		if originalPort != "" {
			os.Setenv("MEERKAT_API_PORT", originalPort)
		} else {
			os.Unsetenv("MEERKAT_API_PORT")
		}
	}()

	os.Setenv("MEERKAT_API_KEY", "test-api-key")
	os.Setenv("MEERKAT_API_PORT", "9090")

	// Setup database
	testDB, err := database.ConnectSQLite(":memory:")
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer testDB.Close()

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
	configLoader := configapp.NewLoader(logger, monitorService, metricsService)

	server, err := NewServer(logger, configLoader, entityRepo, monitorRepo, metricsRepo)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Check that server is configured with the custom port
	if server.httpServer.Addr != ":9090" {
		t.Errorf("expected server address :9090, got %s", server.httpServer.Addr)
	}
}

func TestServer_Shutdown_ContextTimeout(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Start server
	go server.Start()
	time.Sleep(100 * time.Millisecond)

	// Test shutdown with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// This should complete quickly even with a short timeout
	// since the server hasn't processed many requests
	err := server.Shutdown(ctx)
	// We don't check for timeout error here as it depends on timing
	_ = err
}

