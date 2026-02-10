package handlers

import (
	"encoding/json"
	"net/http"

	configapp "meerkat-v0/internal/config/application"
	api "meerkat-v0/internal/api/application"
)

// ConfigHandler handles configuration loading
type ConfigHandler struct {
	configLoader *configapp.Loader
}

// NewConfigHandler creates a new config handler
func NewConfigHandler(configLoader *configapp.Loader) *ConfigHandler {
	return &ConfigHandler{
		configLoader: configLoader,
	}
}

// LoadConfig handles POST /api/v1/config
// @Summary      Load configuration
// @Description  Load a new monitoring and metrics configuration
// @Tags         config
// @Accept       json
// @Produce      json
// @Param        config  body      application.LoadConfigRequest  true  "Configuration object"
// @Success      200     {object}  map[string]string
// @Failure      400     {object}  application.ErrorResponse
// @Security     ApiKeyAuth
// @Router       /config [post]
func (h *ConfigHandler) LoadConfig(w http.ResponseWriter, r *http.Request) {
	logger := getLogger(r)

	var req api.LoadConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("Invalid request body", "err", err)
		respondJSONError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if err := h.configLoader.LoadConfig(r.Context(), req.Config); err != nil {
		logger.Error("Failed to load config", "err", err)
		respondJSONError(w, http.StatusBadRequest, "Failed to load config: "+err.Error())
		return
	}

	logger.Info("Configuration loaded successfully")
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

