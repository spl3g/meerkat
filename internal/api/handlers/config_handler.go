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
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req api.LoadConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSONError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if err := h.configLoader.LoadConfig(r.Context(), req.Config); err != nil {
		respondJSONError(w, http.StatusBadRequest, "Failed to load config: "+err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

