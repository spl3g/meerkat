package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	api "meerkat-v0/internal/api/application"
	monitoringdomain "meerkat-v0/internal/monitoring/domain"
)

// mockMonitorRepository is a mock implementation of monitoringdomain.Repository
type mockMonitorRepository struct {
	heartbeats []monitoringdomain.Heartbeat
	err        error
}

func (m *mockMonitorRepository) InsertHeartbeat(ctx context.Context, heartbeat monitoringdomain.Heartbeat) error {
	if m.err != nil {
		return m.err
	}
	m.heartbeats = append(m.heartbeats, heartbeat)
	return nil
}

func (m *mockMonitorRepository) ListHeartbeats(ctx context.Context, filters monitoringdomain.HeartbeatFilters) ([]monitoringdomain.Heartbeat, error) {
	if m.err != nil {
		return nil, m.err
	}
	result := m.heartbeats
	// Apply offset
	if filters.Offset > 0 && filters.Offset < len(result) {
		result = result[filters.Offset:]
	}
	// Apply limit
	if filters.Limit > 0 && filters.Limit < len(result) {
		result = result[:filters.Limit]
	}
	return result, nil
}

func TestHeartbeatHandler_ListHeartbeats(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name           string
		method         string
		queryParams    map[string]string
		heartbeats     []monitoringdomain.Heartbeat
		repoErr        error
		expectedStatus int
		expectedCount  int
	}{
		{
			name:   "empty list",
			method: http.MethodGet,
			heartbeats: []monitoringdomain.Heartbeat{},
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
		{
			name:   "multiple heartbeats",
			method: http.MethodGet,
			heartbeats: []monitoringdomain.Heartbeat{
				monitoringdomain.NewHeartbeat("monitor1", now, nil),
				monitoringdomain.NewHeartbeat("monitor2", now.Add(-time.Hour), nil),
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:   "with entity_id filter",
			method: http.MethodGet,
			queryParams: map[string]string{
				"entity_id": "monitor1",
			},
			heartbeats: []monitoringdomain.Heartbeat{
				monitoringdomain.NewHeartbeat("monitor1", now, nil),
			},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:   "with from filter",
			method: http.MethodGet,
			queryParams: map[string]string{
				"from": now.Add(-2 * time.Hour).Format(time.RFC3339),
			},
			heartbeats: []monitoringdomain.Heartbeat{
				monitoringdomain.NewHeartbeat("monitor1", now, nil),
			},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:   "with to filter",
			method: http.MethodGet,
			queryParams: map[string]string{
				"to": now.Add(time.Hour).Format(time.RFC3339),
			},
			heartbeats: []monitoringdomain.Heartbeat{
				monitoringdomain.NewHeartbeat("monitor1", now, nil),
			},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:   "with successful filter",
			method: http.MethodGet,
			queryParams: map[string]string{
				"successful": "true",
			},
			heartbeats: []monitoringdomain.Heartbeat{
				monitoringdomain.NewHeartbeat("monitor1", now, nil),
			},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:   "with limit",
			method: http.MethodGet,
			queryParams: map[string]string{
				"limit": "1",
			},
			heartbeats: []monitoringdomain.Heartbeat{
				monitoringdomain.NewHeartbeat("monitor1", now, nil),
				monitoringdomain.NewHeartbeat("monitor2", now, nil),
			},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:   "with offset",
			method: http.MethodGet,
			queryParams: map[string]string{
				"offset": "1",
			},
			heartbeats: []monitoringdomain.Heartbeat{
				monitoringdomain.NewHeartbeat("monitor1", now, nil),
				monitoringdomain.NewHeartbeat("monitor2", now, nil),
			},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:   "invalid date format",
			method: http.MethodGet,
			queryParams: map[string]string{
				"from": "invalid-date",
			},
			heartbeats: []monitoringdomain.Heartbeat{
				monitoringdomain.NewHeartbeat("monitor1", now, nil),
			},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:   "invalid boolean",
			method: http.MethodGet,
			queryParams: map[string]string{
				"successful": "not-a-bool",
			},
			heartbeats: []monitoringdomain.Heartbeat{
				monitoringdomain.NewHeartbeat("monitor1", now, nil),
			},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:   "repository error",
			method: http.MethodGet,
			repoErr: context.DeadlineExceeded,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "method not allowed",
			method:         http.MethodPost,
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockMonitorRepository{
				heartbeats: tt.heartbeats,
				err:        tt.repoErr,
			}
			service := api.NewHeartbeatService(repo)
			handler := NewHeartbeatHandler(service)

			url := "/api/v1/heartbeats"
			if len(tt.queryParams) > 0 {
				url += "?"
				first := true
				for k, v := range tt.queryParams {
					if !first {
						url += "&"
					}
					url += k + "=" + v
					first = false
				}
			}

			req := httptest.NewRequest(tt.method, url, nil)
			w := httptest.NewRecorder()

			handler.ListHeartbeats(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var heartbeats []api.HeartbeatResponse
				if err := json.NewDecoder(w.Body).Decode(&heartbeats); err != nil {
					t.Errorf("failed to decode response: %v", err)
				}
				if len(heartbeats) != tt.expectedCount {
					t.Errorf("expected %d heartbeats, got %d", tt.expectedCount, len(heartbeats))
				}
			}
		})
	}
}

