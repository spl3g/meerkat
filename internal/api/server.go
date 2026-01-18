package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	httpSwagger "github.com/swaggo/http-swagger"

	configapp "meerkat-v0/internal/config/application"
	api "meerkat-v0/internal/api/application"
	"meerkat-v0/internal/api/handlers"
	"meerkat-v0/internal/api/middleware"
	entitydomain "meerkat-v0/internal/shared/entity/domain"
	monitoringdomain "meerkat-v0/internal/monitoring/domain"
	metricsdomain "meerkat-v0/internal/metrics/domain"
)

// Server represents the API server
type Server struct {
	httpServer *http.Server
}

// NewServer creates a new API server
func NewServer(
	configLoader *configapp.Loader,
	entityRepo entitydomain.Repository,
	monitorRepo monitoringdomain.Repository,
	metricsRepo metricsdomain.Repository,
) (*Server, error) {
	// Validate API key is set
	apiKey := os.Getenv("MEERKAT_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("MEERKAT_API_KEY environment variable is required")
	}

	// Initialize services
	entityService := api.NewEntityService(entityRepo)
	heartbeatService := api.NewHeartbeatService(monitorRepo)
	metricsService := api.NewMetricsService(metricsRepo)

	// Initialize handlers
	configHandler := handlers.NewConfigHandler(configLoader)
	entityHandler := handlers.NewEntityHandler(entityService)
	heartbeatHandler := handlers.NewHeartbeatHandler(heartbeatService)
	metricsHandler := handlers.NewMetricsHandler(metricsService)

	// Get port from environment or use default
	port := os.Getenv("MEERKAT_API_PORT")
	if port == "" {
		port = "8080"
	}

	// Check if dev mode is enabled
	devMode := os.Getenv("MEERKAT_DEV_MODE") == "true"

	// Setup router
	mux := http.NewServeMux()

	// Swagger UI (only in dev mode)
	if devMode {
		// Use relative path - http-swagger will handle serving all assets
		swaggerHandler := httpSwagger.Handler(
			httpSwagger.URL("/swagger/doc.json"),
		)
		// Use Handle to properly serve all sub-paths and assets
		mux.Handle("/swagger/", swaggerHandler)
		// Redirect /swagger to /swagger/
		mux.HandleFunc("/swagger", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
		})
	}

	// API v1 routes
	apiV1 := http.NewServeMux()
	apiV1.HandleFunc("/config", configHandler.LoadConfig)
	apiV1.HandleFunc("/entities", entityHandler.ListEntities)
	apiV1.HandleFunc("/entities/", entityHandler.GetEntity)
	apiV1.HandleFunc("/heartbeats", heartbeatHandler.ListHeartbeats)
	apiV1.HandleFunc("/metrics", metricsHandler.ListSamples)

	mux.Handle("/api/v1/", http.StripPrefix("/api/v1", apiV1))

	// Apply middleware - it will skip authentication for Swagger paths
	handler := middleware.APIKeyAuth(mux)

	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		httpServer: httpServer,
	}, nil
}

// Start starts the HTTP server
func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

