package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	chimiddleware "github.com/go-chi/chi/v5/middleware"

	api "meerkat-v0/internal/api/application"
)

// APIKeyAuth middleware validates the X-API-Key header (reads from environment for backward compatibility)
func APIKeyAuth(next http.Handler) http.Handler {
	expectedKey := os.Getenv("MEERKAT_API_KEY")
	return APIKeyAuthWithKey(expectedKey)(next)
}

// APIKeyAuthWithKey middleware validates the X-API-Key header with a provided API key
func APIKeyAuthWithKey(expectedKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if expectedKey == "" {
			// If no API key is set, reject all requests
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				logger := slog.Default()
				logger.Error("API key not configured",
					"path", r.URL.Path,
					"method", r.Method,
					"remote_addr", r.RemoteAddr,
				)
				respondJSONError(w, http.StatusInternalServerError, "API key not configured")
			})
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get logger from context (set by httplog middleware)
			logger := slog.Default()
			if ctxLogger := r.Context().Value("logger"); ctxLogger != nil {
				if l, ok := ctxLogger.(*slog.Logger); ok {
					logger = l
				}
			}

			// Get request ID from context (set by chi middleware)
			requestID := chimiddleware.GetReqID(r.Context())
			if requestID == "" {
				requestID = "unknown"
			}

			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" || apiKey != expectedKey {
				logger.Warn("Authentication failed",
					"request_id", requestID,
					"path", r.URL.Path,
					"method", r.Method,
					"remote_addr", r.RemoteAddr,
					"has_api_key", apiKey != "",
				)
				respondJSONError(w, http.StatusUnauthorized, "Invalid or missing API key")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// respondJSONError sends a JSON error response
func respondJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	response := api.ErrorResponse{Error: message}
	json.NewEncoder(w).Encode(response)
}

