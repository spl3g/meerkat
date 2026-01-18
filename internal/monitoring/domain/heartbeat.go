package domain

import "time"

// Heartbeat represents a monitoring heartbeat value object
type Heartbeat struct {
	MonitorID string
	Timestamp time.Time
	Error     error
}

// NewHeartbeat creates a new heartbeat
func NewHeartbeat(monitorID string, timestamp time.Time, err error) Heartbeat {
	return Heartbeat{
		MonitorID: monitorID,
		Timestamp: timestamp,
		Error:     err,
	}
}

// IsSuccessful returns whether the heartbeat represents a successful check
func (h Heartbeat) IsSuccessful() bool {
	return h.Error == nil
}

