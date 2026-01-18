package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	api "meerkat-v0/internal/api/application"
)

// getLogger extracts the logger from the request context
// Falls back to slog.Default() if not found
func getLogger(r *http.Request) *slog.Logger {
	if ctxLogger := r.Context().Value("logger"); ctxLogger != nil {
		if l, ok := ctxLogger.(*slog.Logger); ok {
			return l
		}
	}
	return slog.Default()
}

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondJSONError sends a JSON error response
func respondJSONError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, api.ErrorResponse{Error: message})
}

