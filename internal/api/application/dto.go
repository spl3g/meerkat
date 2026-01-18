package application

import (
	"encoding/json"
	"time"

	metricsdomain "meerkat-v0/internal/metrics/domain"
	monitoringdomain "meerkat-v0/internal/monitoring/domain"
	entitydomain "meerkat-v0/internal/shared/entity/domain"
)

// EntityResponse represents an entity in API responses
type EntityResponse struct {
	ID          int64  `json:"id"`
	CanonicalID string `json:"canonical_id"`
}

// HeartbeatResponse represents a heartbeat in API responses
type HeartbeatResponse struct {
	MonitorID   string    `json:"monitor_id"`
	Timestamp   time.Time `json:"timestamp"`
	Successful  bool      `json:"successful"`
	Error       *string   `json:"error,omitempty"`
}

// MetricsSampleResponse represents a metrics sample in API responses
type MetricsSampleResponse struct {
	EntityID  string            `json:"entity_id"`
	Timestamp time.Time         `json:"timestamp"`
	Type      string            `json:"type"`
	Name      string            `json:"name"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels"`
}

// ListHeartbeatsRequest represents query parameters for listing heartbeats
type ListHeartbeatsRequest struct {
	EntityID   *string    `json:"entity_id,omitempty"`
	From       *time.Time `json:"from,omitempty"`
	To         *time.Time `json:"to,omitempty"`
	Successful *bool      `json:"successful,omitempty"`
	Limit      int        `json:"limit,omitempty"`
	Offset     int        `json:"offset,omitempty"`
}

// ListSamplesRequest represents query parameters for listing metrics samples
type ListSamplesRequest struct {
	EntityID *string    `json:"entity_id,omitempty"`
	From     *time.Time `json:"from,omitempty"`
	To       *time.Time `json:"to,omitempty"`
	Name     *string    `json:"name,omitempty"`
	Type     *string    `json:"type,omitempty"`
	Limit    int        `json:"limit,omitempty"`
	Offset   int        `json:"offset,omitempty"`
}

// LoadConfigRequest represents the configuration payload
type LoadConfigRequest struct {
	Config json.RawMessage `json:"config"`
}

// ErrorResponse represents an error in API responses
type ErrorResponse struct {
	Error string `json:"error"`
}

// ToEntityResponse converts a domain entity to an API response
func ToEntityResponse(e entitydomain.Entity) EntityResponse {
	return EntityResponse{
		ID:          e.ID,
		CanonicalID: e.CanonicalID,
	}
}

// ToHeartbeatResponse converts a domain heartbeat to an API response
func ToHeartbeatResponse(h monitoringdomain.Heartbeat) HeartbeatResponse {
	resp := HeartbeatResponse{
		MonitorID:  h.MonitorID,
		Timestamp:  h.Timestamp,
		Successful: h.IsSuccessful(),
	}
	if h.Error != nil {
		errMsg := h.Error.Error()
		resp.Error = &errMsg
	}
	return resp
}

// ToMetricsSampleResponse converts a domain sample to an API response
func ToMetricsSampleResponse(s metricsdomain.Sample) MetricsSampleResponse {
	return MetricsSampleResponse{
		EntityID:  s.ID.Canonical(),
		Timestamp: s.Timestamp,
		Type:      string(s.Type),
		Name:      s.Name,
		Value:     s.Value,
		Labels:    s.Labels,
	}
}

