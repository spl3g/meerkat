package handlers

import (
	"net/http"
	"strconv"
	"time"

	api "meerkat-v0/internal/api/application"
)

// MetricsHandler handles metrics sample queries
type MetricsHandler struct {
	service *api.MetricsService
}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler(service *api.MetricsService) *MetricsHandler {
	return &MetricsHandler{
		service: service,
	}
}

// ListSamples handles GET /api/v1/metrics
// @Summary      List metrics samples
// @Description  Get a list of metrics samples with optional filtering
// @Tags         metrics
// @Accept       json
// @Produce      json
// @Param        entity_id  query     string  false  "Filter by entity ID"
// @Param        from       query     string  false  "Start time (RFC3339)"
// @Param        to         query     string  false  "End time (RFC3339)"
// @Param        name       query     string  false  "Filter by metric name"
// @Param        type       query     string  false  "Filter by metric type"
// @Param        limit      query     int     false  "Limit results"
// @Param        offset     query     int     false  "Offset results"
// @Success      200        {array}   application.MetricsSampleResponse
// @Failure      500        {object}  application.ErrorResponse
// @Security     ApiKeyAuth
// @Router       /metrics [get]
func (h *MetricsHandler) ListSamples(w http.ResponseWriter, r *http.Request) {
	logger := getLogger(r)

	req := api.ListSamplesRequest{}

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

	if name := r.URL.Query().Get("name"); name != "" {
		req.Name = &name
	}

	if metricType := r.URL.Query().Get("type"); metricType != "" {
		req.Type = &metricType
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

	samples, err := h.service.ListSamples(r.Context(), req)
	if err != nil {
		logger.Error("Failed to list metrics", "err", err, "filters", req)
		respondJSONError(w, http.StatusInternalServerError, "Failed to list metrics: "+err.Error())
		return
	}

	logger.Debug("Listed metrics samples", "count", len(samples))
	respondJSON(w, http.StatusOK, samples)
}

