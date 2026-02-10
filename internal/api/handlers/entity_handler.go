package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

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
	logger := getLogger(r)

	entities, err := h.service.ListEntities(r.Context())
	if err != nil {
		logger.Error("Failed to list entities", "err", err)
		respondJSONError(w, http.StatusInternalServerError, "Failed to list entities: "+err.Error())
		return
	}

	logger.Debug("Listed entities", "count", len(entities))
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
	logger := getLogger(r)

	// Extract canonical ID from chi URL parameter
	id := chi.URLParam(r, "id")
	if id == "" {
		logger.Warn("Missing entity ID in request")
		respondJSONError(w, http.StatusBadRequest, "Missing entity ID")
		return
	}

	entity, err := h.service.GetEntity(r.Context(), id)
	if err != nil {
		if err.Error() == "could not find entity with this id" {
			logger.Debug("Entity not found", "id", id)
			respondJSONError(w, http.StatusNotFound, "Entity not found")
			return
		}
		logger.Error("Failed to get entity", "id", id, "err", err)
		respondJSONError(w, http.StatusInternalServerError, "Failed to get entity: "+err.Error())
		return
	}

	logger.Debug("Retrieved entity", "id", id)
	respondJSON(w, http.StatusOK, entity)
}

