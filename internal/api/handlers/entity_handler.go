package handlers

import (
	"net/http"
	"strings"

	api "meerkat-v0/internal/api/application"
)

// EntityHandler handles entity queries
type EntityHandler struct {
	service *api.EntityService
}

// NewEntityHandler creates a new entity handler
func NewEntityHandler(service *api.EntityService) *EntityHandler {
	return &EntityHandler{
		service: service,
	}
}

// ListEntities handles GET /api/v1/entities
// @Summary      List all entities
// @Description  Get a list of all entities
// @Tags         entities
// @Accept       json
// @Produce      json
// @Success      200  {array}   application.EntityResponse
// @Failure      500  {object}  application.ErrorResponse
// @Security     ApiKeyAuth
// @Router       /entities [get]
func (h *EntityHandler) ListEntities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	entities, err := h.service.ListEntities(r.Context())
	if err != nil {
		respondJSONError(w, http.StatusInternalServerError, "Failed to list entities: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, entities)
}

// GetEntity handles GET /api/v1/entities/{id}
// @Summary      Get entity by ID
// @Description  Get a specific entity by its canonical ID
// @Tags         entities
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Entity Canonical ID"
// @Success      200  {object}  application.EntityResponse
// @Failure      400  {object}  application.ErrorResponse
// @Failure      404  {object}  application.ErrorResponse
// @Failure      500  {object}  application.ErrorResponse
// @Security     ApiKeyAuth
// @Router       /entities/{id} [get]
func (h *EntityHandler) GetEntity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract canonical ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/entities/")
	if path == "" || path == r.URL.Path {
		respondJSONError(w, http.StatusBadRequest, "Missing entity ID")
		return
	}

	entity, err := h.service.GetEntity(r.Context(), path)
	if err != nil {
		if err.Error() == "could not find entity with this id" {
			respondJSONError(w, http.StatusNotFound, "Entity not found")
			return
		}
		respondJSONError(w, http.StatusInternalServerError, "Failed to get entity: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, entity)
}

