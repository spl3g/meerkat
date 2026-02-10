package handlers

import (
	"encoding/json"
	"net/http"

	api "meerkat-v0/internal/api/application"
)

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

