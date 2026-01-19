package handlers

import (
	"encoding/json"
	"io"
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

	// Check HTTP method - only POST is allowed
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Read the request body
	bodyBytes := make([]byte, 0)
	if r.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			logger.Warn("Failed to read request body", "err", err)
			respondJSONError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
			return
		}
	}

	if len(bodyBytes) == 0 {
		logger.Warn("Empty request body")
		respondJSONError(w, http.StatusBadRequest, "Invalid request body: request body is required")
		return
	}

	// Try to parse as LoadConfigRequest first (wrapped format: {"config": {...}})
	var req api.LoadConfigRequest
	var configBytes []byte
	if err := json.Unmarshal(bodyBytes, &req); err == nil && len(req.Config) > 0 {
		// Successfully parsed as wrapped format
		configBytes = req.Config
		logger.Debug("Parsed config as wrapped format")
	} else {
		// Try to parse as direct config format
		// Validate it's valid JSON first
		var testConfig map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &testConfig); err != nil {
			logger.Warn("Invalid JSON in request body", "err", err)
			respondJSONError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
			return
		}
		// Use the body directly as the config
		configBytes = bodyBytes
		logger.Debug("Parsed config as direct format")
	}

	if err := h.configLoader.LoadConfig(r.Context(), configBytes); err != nil {
		logger.Error("Failed to load config", "err", err)
		respondJSONError(w, http.StatusBadRequest, "Failed to load config: "+err.Error())
		return
	}

	logger.Info("Configuration loaded successfully")
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// GetConfig handles GET /api/v1/config
// @Summary      Get current configuration
// @Description  Retrieve the current monitoring and metrics configuration
// @Tags         config
// @Produce      json
// @Success      200     {object}  map[string]interface{}
// @Failure      404     {object}  application.ErrorResponse
// @Security     ApiKeyAuth
// @Router       /config [get]
func (h *ConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	logger := getLogger(r)

	config := h.configLoader.GetConfig()
	if config == nil || len(config) == 0 {
		logger.Warn("No configuration loaded")
		respondJSONError(w, http.StatusNotFound, "No configuration loaded")
		return
	}

	var configObj map[string]interface{}
	if err := json.Unmarshal(config, &configObj); err != nil {
		logger.Error("Failed to parse stored config", "err", err)
		respondJSONError(w, http.StatusInternalServerError, "Failed to retrieve config: "+err.Error())
		return
	}

	logger.Debug("Configuration retrieved successfully")
	respondJSON(w, http.StatusOK, configObj)
}

