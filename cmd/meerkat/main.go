// @title           Meerkat API
// @version         1.0
// @description     This is the Meerkat monitoring and metrics API server.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
// @description API Key authentication

// @host      localhost:8080
// @BasePath  /api/v1

package main

import (
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"time"

	_ "modernc.org/sqlite"

	_ "meerkat-v0/docs" // Swagger docs

	"meerkat-v0/db"
	apiserver "meerkat-v0/internal/api"
	configapp "meerkat-v0/internal/config/application"
	"meerkat-v0/internal/infrastructure/database"
	"meerkat-v0/internal/infrastructure/logger"
	metricsapp "meerkat-v0/internal/metrics/application"
	metricsinfra "meerkat-v0/internal/metrics/infrastructure"
	monitoringapp "meerkat-v0/internal/monitoring/application"
	monitoringinfra "meerkat-v0/internal/monitoring/infrastructure"
	"meerkat-v0/internal/schema"
	entityinfra "meerkat-v0/internal/shared/entity/infrastructure"
)

func help() {
	fmt.Fprintln(os.Stderr, "./meerkat [config]")
}

func run() error {
	if len(os.Args) < 2 {
		help()
		return fmt.Errorf("not enough arguments")
	}

	sigCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	configPath := os.Args[1]
	rawCfg, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Initialize database connections
	dbRead, err := database.ConnectSQLite("observations.db")
	if err != nil {
		return fmt.Errorf("failed to connect to read database: %w", err)
	}
	defer dbRead.Close()
	dbRead.SetMaxOpenConns(runtime.NumCPU())

	dbWrite, err := database.ConnectSQLite("observations.db")
	if err != nil {
		return fmt.Errorf("failed to connect to write database: %w", err)
	}
	defer dbWrite.Close()
	dbWrite.SetMaxOpenConns(1)

	// Initialize schema
	_, err = dbWrite.ExecContext(sigCtx, schema.DDL)
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	readDB := db.New(dbRead)
	writeDB := db.New(dbWrite)

	// Initialize shared entity repository
	entityRepo := entityinfra.NewRepository(readDB, writeDB)

	// Initialize monitoring services
	monitorRepo := monitoringinfra.NewRepository(readDB, writeDB, dbRead, entityRepo)
	monitorLogger := logger.DefaultLogger()
	monitorService := monitoringapp.NewService(monitorLogger, monitorRepo, entityRepo)

	// Initialize metrics services
	metricsRepo := metricsinfra.NewRepository(readDB, writeDB, dbRead, entityRepo)
	metricsLogger := logger.DefaultLogger()
	metricsService := metricsapp.NewService(metricsLogger, metricsRepo, entityRepo)

	// Initialize configuration loader
	configLoader := configapp.NewLoader(monitorService, metricsService)

	// Load configuration
	err = configLoader.LoadConfig(sigCtx, rawCfg)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize API server
	apiServer, err := apiserver.NewServer(configLoader, entityRepo, monitorRepo, metricsRepo)
	if err != nil {
		return fmt.Errorf("failed to create API server: %w", err)
	}

	// Start API server in a goroutine
	serverErrChan := make(chan error, 1)
	go func() {
		if err := apiServer.Start(); err != nil && err != http.ErrServerClosed {
			serverErrChan <- fmt.Errorf("API server error: %w", err)
		}
	}()

	// Wait for interrupt or server error
	select {
	case <-sigCtx.Done():
		// Graceful shutdown
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		var shutdownErr error
		if err := apiServer.Shutdown(shutdownCtx); err != nil {
			shutdownErr = fmt.Errorf("API server shutdown error: %w", err)
		}

		if err := configLoader.Stop(shutdownCtx); err != nil {
			if shutdownErr != nil {
				return fmt.Errorf("multiple shutdown errors: %v, %v", shutdownErr, err)
			}
			return fmt.Errorf("config loader shutdown error: %w", err)
		}

		return shutdownErr
	case err := <-serverErrChan:
		return err
	}
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

