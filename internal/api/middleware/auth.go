package middleware

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	api "meerkat-v0/internal/api/application"
)

// APIKeyAuth middleware validates the X-API-Key header
func APIKeyAuth(next http.Handler) http.Handler {
	expectedKey := os.Getenv("MEERKAT_API_KEY")
	if expectedKey == "" {
		// If no API key is set, reject all requests
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			respondJSONError(w, http.StatusInternalServerError, "API key not configured")
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for Swagger UI and all its assets
		path := r.URL.Path
		if path == "/swagger" || strings.HasPrefix(path, "/swagger/") {
			next.ServeHTTP(w, r)
			return
		}

		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" || apiKey != expectedKey {
			respondJSONError(w, http.StatusUnauthorized, "Invalid or missing API key")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// respondJSONError sends a JSON error response
func respondJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	response := api.ErrorResponse{Error: message}
	json.NewEncoder(w).Encode(response)
}

