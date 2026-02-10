package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"time"

	_ "modernc.org/sqlite"

	"meerkat-v0/db"
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
	monitorRepo := monitoringinfra.NewRepository(readDB, writeDB, entityRepo)
	monitorLogger := logger.DefaultLogger()
	monitorService := monitoringapp.NewService(monitorLogger, monitorRepo, entityRepo)

	// Initialize metrics services
	metricsRepo := metricsinfra.NewRepository(readDB, writeDB, entityRepo)
	metricsLogger := logger.DefaultLogger()
	metricsService := metricsapp.NewService(metricsLogger, metricsRepo, entityRepo)

	// Initialize configuration loader
	configLoader := configapp.NewLoader(monitorService, metricsService)

	// Load configuration
	err = configLoader.LoadConfig(sigCtx, rawCfg)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Wait for interrupt
	<-sigCtx.Done()

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 350*time.Millisecond)
	defer cancel()
	return configLoader.Stop(ctx)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

