package handlers

import (
	"net/http"
	"strconv"
	"time"

	api "meerkat-v0/internal/api/application"
)

// HeartbeatHandler handles heartbeat queries
type HeartbeatHandler struct {
	service *api.HeartbeatService
}

// NewHeartbeatHandler creates a new heartbeat handler
func NewHeartbeatHandler(service *api.HeartbeatService) *HeartbeatHandler {
	return &HeartbeatHandler{
		service: service,
	}
}

// ListHeartbeats handles GET /api/v1/heartbeats
// @Summary      List heartbeats
// @Description  Get a list of heartbeats with optional filtering
// @Tags         heartbeats
// @Accept       json
// @Produce      json
// @Param        entity_id   query     string  false  "Filter by entity ID"
// @Param        from        query     string  false  "Start time (RFC3339)"
// @Param        to          query     string  false  "End time (RFC3339)"
// @Param        successful  query     boolean false  "Filter by success status"
// @Param        limit       query     int     false  "Limit results"
// @Param        offset      query     int     false  "Offset results"
// @Success      200         {array}   application.HeartbeatResponse
// @Failure      500         {object}  application.ErrorResponse
// @Security     ApiKeyAuth
// @Router       /heartbeats [get]
func (h *HeartbeatHandler) ListHeartbeats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	logger := getLogger(r)

	req := api.ListHeartbeatsRequest{}

	// Parse query parameters
	if entityID := r.URL.Query().Get("entity_id"); entityID != "" {
		req.EntityID = &entityID
	}

	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		if from, err := time.Parse(time.RFC3339, fromStr); err == nil {
			req.From = &from
		}
	}

	if toStr := r.URL.Query().Get("to"); toStr != "" {
		if to, err := time.Parse(time.RFC3339, toStr); err == nil {
			req.To = &to
		}
	}

	if successfulStr := r.URL.Query().Get("successful"); successfulStr != "" {
		if successful, err := strconv.ParseBool(successfulStr); err == nil {
			req.Successful = &successful
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			req.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			req.Offset = offset
		}
	}

	heartbeats, err := h.service.ListHeartbeats(r.Context(), req)
	if err != nil {
		logger.Error("Failed to list heartbeats", "err", err, "filters", req)
		respondJSONError(w, http.StatusInternalServerError, "Failed to list heartbeats: "+err.Error())
		return
	}

	logger.Debug("Listed heartbeats", "count", len(heartbeats))
	respondJSON(w, http.StatusOK, heartbeats)
}

