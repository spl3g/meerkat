package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v3"
	httpSwagger "github.com/swaggo/http-swagger"

	configapp "meerkat-v0/internal/config/application"
	api "meerkat-v0/internal/api/application"
	"meerkat-v0/internal/api/handlers"
	apimiddleware "meerkat-v0/internal/api/middleware"
	sharedlogger "meerkat-v0/internal/shared/logger"
	entitydomain "meerkat-v0/internal/shared/entity/domain"
	monitoringdomain "meerkat-v0/internal/monitoring/domain"
	metricsdomain "meerkat-v0/internal/metrics/domain"
)

// Server represents the API server
type Server struct {
	httpServer *http.Server
	logger     sharedlogger.Logger
}

// NewServer creates a new API server
func NewServer(
	logger sharedlogger.Logger,
	runtimeCfg *configapp.RuntimeConfig,
	configLoader *configapp.Loader,
	entityRepo entitydomain.Repository,
	monitorRepo monitoringdomain.Repository,
	metricsRepo metricsdomain.Repository,
) (*Server, error) {
	// Validate API key is set
	if runtimeCfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required (set MEERKAT_API_KEY or use --api-key flag)")
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

	// Setup chi router
	r := chi.NewRouter()

	// Middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	
	// HTTP logging middleware - need concrete slog.Logger for httplog
	// Type assert to infrastructure logger to get underlying slog.Logger
	var slogLogger *slog.Logger
	if infraLogger, ok := logger.(interface{ SLog() *slog.Logger }); ok {
		slogLogger = infraLogger.SLog()
	} else {
		// Fallback to default if type assertion fails
		slogLogger = slog.Default()
	}
	
	r.Use(httplog.RequestLogger(slogLogger, &httplog.Options{
		Level:            slog.LevelDebug,
		Schema:           httplog.SchemaECS.Concise(true),
		LogRequestHeaders: []string{}, // Log no headers by default to reduce verbosity
	}))

	// Swagger UI (only in dev mode, no auth required)
	if runtimeCfg.DevMode {
		swaggerHandler := httpSwagger.Handler(
			httpSwagger.URL("/swagger/doc.json"),
		)
		r.Handle("/swagger/*", swaggerHandler)
		r.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
		})
	}

	// API v1 routes (with authentication)
	r.Route("/api/v1", func(r chi.Router) {
		// Apply API key auth middleware with configured API key
		r.Use(apimiddleware.APIKeyAuthWithKey(runtimeCfg.APIKey))
		
		// Routes
		r.Get("/config", configHandler.GetConfig)
		r.Post("/config", configHandler.LoadConfig)
		r.Get("/entities", entityHandler.ListEntities)
		r.Get("/entities/{id}", entityHandler.GetEntity)
		r.Get("/heartbeats", heartbeatHandler.ListHeartbeats)
		r.Get("/metrics", metricsHandler.ListSamples)
	})

	httpServer := &http.Server{
		Addr:         ":" + runtimeCfg.APIPort,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logger.Debug("Server configured",
		"port", runtimeCfg.APIPort,
		"dev_mode", runtimeCfg.DevMode,
		"middleware", []string{"RequestID", "RealIP", "Recoverer", "httplog"},
	)

	return &Server{
		httpServer: httpServer,
		logger:     logger,
	}, nil
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.logger.Info("Starting HTTP server", "addr", s.httpServer.Addr)
	err := s.httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		s.logger.Error("Server error", "err", err)
	}
	return err
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down HTTP server")
	err := s.httpServer.Shutdown(ctx)
	if err != nil {
		s.logger.Error("Server shutdown error", "err", err)
	} else {
		s.logger.Info("Server shutdown complete")
	}
	return err
}

