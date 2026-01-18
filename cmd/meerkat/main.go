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

	"github.com/urfave/cli/v2"
	_ "modernc.org/sqlite"

	_ "meerkat-v0/docs" // Swagger docs

	"meerkat-v0/internal/infrastructure/database/queries"
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

const version = "1.0"

func run(runtimeCfg *configapp.RuntimeConfig) error {
	// Initialize logger (after .env is loaded so it can read env vars)
	appLogger := logger.NewLogger()
	logger.SetDefaultLogger(appLogger)

	appLogger.Info("Starting Meerkat", "version", version)

	sigCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Read configuration file
	appLogger.Debug("Reading configuration file", "path", runtimeCfg.ConfigPath)
	rawCfg, err := os.ReadFile(runtimeCfg.ConfigPath)
	if err != nil {
		appLogger.Error("Failed to read config file", "path", runtimeCfg.ConfigPath, "err", err)
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Initialize database connections
	appLogger.Debug("Connecting to database", "file", runtimeCfg.DBPath)
	dbRead, err := database.ConnectSQLite(runtimeCfg.DBPath)
	if err != nil {
		appLogger.Error("Failed to connect to read database", "err", err)
		return fmt.Errorf("failed to connect to read database: %w", err)
	}
	defer dbRead.Close()
	dbRead.SetMaxOpenConns(runtime.NumCPU())
	appLogger.Debug("Read database configured", "max_open_conns", runtime.NumCPU())

	dbWrite, err := database.ConnectSQLite(runtimeCfg.DBPath)
	if err != nil {
		appLogger.Error("Failed to connect to write database", "err", err)
		return fmt.Errorf("failed to connect to write database: %w", err)
	}
	defer dbWrite.Close()
	dbWrite.SetMaxOpenConns(1)
	appLogger.Debug("Write database configured", "max_open_conns", 1)

	// Initialize schema
	appLogger.Debug("Initializing database schema")
	_, err = dbWrite.ExecContext(sigCtx, schema.DDL)
	if err != nil {
		appLogger.Error("Failed to initialize schema", "err", err)
		return fmt.Errorf("failed to initialize schema: %w", err)
	}
	appLogger.Debug("Database schema initialized")

	readDB := queries.New(dbRead)
	writeDB := queries.New(dbWrite)

	// Initialize shared entity repository
	appLogger.Debug("Initializing entity repository")
	entityRepo := entityinfra.NewRepository(readDB, writeDB)

	// Initialize monitoring services
	appLogger.Debug("Initializing monitoring service")
	monitorRepo := monitoringinfra.NewRepository(readDB, writeDB, dbRead, dbWrite, entityRepo)
	monitorLogger := logger.NewLogger()
	monitorService := monitoringapp.NewService(monitorLogger, monitorRepo, entityRepo)
	appLogger.Debug("Monitoring service initialized")

	// Initialize metrics services
	appLogger.Debug("Initializing metrics service")
	metricsRepo := metricsinfra.NewRepository(readDB, writeDB, dbRead, dbWrite, entityRepo)
	metricsLogger := logger.NewLogger()
	metricsService := metricsapp.NewService(metricsLogger, metricsRepo, entityRepo)
	appLogger.Debug("Metrics service initialized")

	// Initialize configuration loader
	appLogger.Debug("Initializing configuration loader")
	configLoader := configapp.NewLoader(appLogger, monitorService, metricsService)

	// Load configuration
	appLogger.Info("Loading configuration")
	err = configLoader.LoadConfig(sigCtx, rawCfg)
	if err != nil {
		appLogger.Error("Failed to load config", "err", err)
		return fmt.Errorf("failed to load config: %w", err)
	}
	appLogger.Info("Configuration loaded successfully")

	// Initialize API server
	appLogger.Debug("Initializing API server")
	apiServer, err := apiserver.NewServer(appLogger, runtimeCfg, configLoader, entityRepo, monitorRepo, metricsRepo)
	if err != nil {
		appLogger.Error("Failed to create API server", "err", err)
		return fmt.Errorf("failed to create API server: %w", err)
	}
	appLogger.Debug("API server initialized")

	// Start API server in a goroutine
	serverErrChan := make(chan error, 1)
	go func() {
		if err := apiServer.Start(); err != nil && err != http.ErrServerClosed {
			appLogger.Error("API server error", "err", err)
			serverErrChan <- fmt.Errorf("API server error: %w", err)
		}
	}()

	appLogger.Info("Meerkat started successfully, waiting for shutdown signal")

	// Wait for interrupt or server error
	select {
	case <-sigCtx.Done():
		appLogger.Info("Shutdown signal received, starting graceful shutdown")
		// Graceful shutdown
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		var shutdownErr error
		if err := apiServer.Shutdown(shutdownCtx); err != nil {
			appLogger.Error("API server shutdown error", "err", err)
			shutdownErr = fmt.Errorf("API server shutdown error: %w", err)
		}

		if err := configLoader.Stop(shutdownCtx); err != nil {
			appLogger.Error("Config loader shutdown error", "err", err)
			if shutdownErr != nil {
				return fmt.Errorf("multiple shutdown errors: %v, %v", shutdownErr, err)
			}
			return fmt.Errorf("config loader shutdown error: %w", err)
		}

		appLogger.Info("Graceful shutdown completed")
		return shutdownErr
	case err := <-serverErrChan:
		appLogger.Error("Server error received", "err", err)
		return err
	}
}

func main() {
	app := &cli.App{
		Name:    "meerkat",
		Usage:   "Meerkat monitoring and metrics API server",
		Version: version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "config",
				Aliases:  []string{"c"},
				Usage:    "Path to configuration file (required)",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "api-key",
				Usage:   "API key for authentication (overrides MEERKAT_API_KEY)",
				EnvVars: []string{"MEERKAT_API_KEY"},
			},
			&cli.StringFlag{
				Name:    "port",
				Usage:   "API server port (overrides MEERKAT_API_PORT, default: 8080)",
				EnvVars: []string{"MEERKAT_API_PORT"},
				Value:   "8080",
			},
			&cli.BoolFlag{
				Name:    "dev-mode",
				Usage:   "Enable development mode (overrides MEERKAT_DEV_MODE)",
				EnvVars: []string{"MEERKAT_DEV_MODE"},
			},
			&cli.StringFlag{
				Name:    "log-level",
				Usage:   "Log level: DEBUG, INFO, WARN, ERROR (overrides MEERKAT_LOG_LEVEL, default: INFO)",
				EnvVars: []string{"MEERKAT_LOG_LEVEL"},
				Value:   "INFO",
			},
			&cli.StringFlag{
				Name:    "log-format",
				Usage:   "Log format: json or text (overrides MEERKAT_LOG_FORMAT, default: text)",
				EnvVars: []string{"MEERKAT_LOG_FORMAT"},
				Value:   "text",
			},
			&cli.StringFlag{
				Name:    "log-output",
				Usage:   "Log output: stdout, stderr, or file path (overrides MEERKAT_LOG_OUTPUT, default: stdout)",
				EnvVars: []string{"MEERKAT_LOG_OUTPUT"},
				Value:   "stdout",
			},
			&cli.StringFlag{
				Name:    "db-path",
				Usage:   "Database file path (overrides MEERKAT_DB_PATH, default: observations.db)",
				EnvVars: []string{"MEERKAT_DB_PATH"},
				Value:   "observations.db",
			},
			&cli.StringFlag{
				Name:    "env-file",
				Usage:   "Path to .env file (default: .env in current directory)",
				Value:   ".env",
			},
		},
		Action: func(c *cli.Context) error {
			// Load .env file first (before any other initialization)
			// Use a temporary logger for .env loading messages
			tempLogger := logger.DefaultLogger()
			envFile := c.String("env-file")
			configapp.LoadEnvFile(tempLogger, envFile)

			// Build runtime configuration from CLI flags and environment
			runtimeCfg := configapp.LoadRuntimeConfig(
				c.String("api-key"),
				c.String("port"),
				c.String("log-level"),
				c.String("log-format"),
				c.String("log-output"),
				c.String("db-path"),
				c.String("config"),
				c.Bool("dev-mode"),
			)

			// Validate configuration
			if err := runtimeCfg.Validate(); err != nil {
				return fmt.Errorf("configuration error: %w", err)
			}

			return run(runtimeCfg)
		},
	}

	if err := app.Run(os.Args); err != nil {
		// Use default logger for final error message if run() failed early
		log := logger.DefaultLogger()
		log.Error("Application error", "err", err)
		os.Exit(1)
	}
}

