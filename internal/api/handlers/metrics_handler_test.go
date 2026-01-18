package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	api "meerkat-v0/internal/api/application"
	metricsdomain "meerkat-v0/internal/metrics/domain"
	"meerkat-v0/pkg/utils"
)

// mockMetricsRepository is a mock implementation of metricsdomain.Repository
type mockMetricsRepository struct {
	samples []metricsdomain.Sample
	err     error
}

func (m *mockMetricsRepository) InsertSample(ctx context.Context, sample metricsdomain.Sample) error {
	if m.err != nil {
		return m.err
	}
	m.samples = append(m.samples, sample)
	return nil
}

func (m *mockMetricsRepository) ListSamples(ctx context.Context, filters metricsdomain.SampleFilters) ([]metricsdomain.Sample, error) {
	if m.err != nil {
		return nil, m.err
	}
	result := m.samples
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

func TestMetricsHandler_ListSamples(t *testing.T) {
	now := time.Now()
	entityID := utils.EntityID{
		Kind: "metric",
		Labels: map[string]string{
			"instance": "test",
			"service":  "test-service",
			"type":     "cpu",
			"name":     "usage",
		},
	}

	tests := []struct {
		name           string
		method         string
		queryParams    map[string]string
		samples        []metricsdomain.Sample
		repoErr        error
		expectedStatus int
		expectedCount  int
	}{
		{
			name:   "empty list",
			method: http.MethodGet,
			samples: []metricsdomain.Sample{},
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
		{
			name:   "multiple samples",
			method: http.MethodGet,
			samples: []metricsdomain.Sample{
				metricsdomain.NewSample(entityID, now, metricsdomain.MetricGauge, "cpu_usage", 0.5, nil),
				metricsdomain.NewSample(entityID, now.Add(-time.Hour), metricsdomain.MetricGauge, "cpu_usage", 0.6, nil),
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:   "with entity_id filter",
			method: http.MethodGet,
			queryParams: map[string]string{
				"entity_id": entityID.Canonical(),
			},
			samples: []metricsdomain.Sample{
				metricsdomain.NewSample(entityID, now, metricsdomain.MetricGauge, "cpu_usage", 0.5, nil),
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
			samples: []metricsdomain.Sample{
				metricsdomain.NewSample(entityID, now, metricsdomain.MetricGauge, "cpu_usage", 0.5, nil),
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
			samples: []metricsdomain.Sample{
				metricsdomain.NewSample(entityID, now, metricsdomain.MetricGauge, "cpu_usage", 0.5, nil),
			},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:   "with name filter",
			method: http.MethodGet,
			queryParams: map[string]string{
				"name": "cpu_usage",
			},
			samples: []metricsdomain.Sample{
				metricsdomain.NewSample(entityID, now, metricsdomain.MetricGauge, "cpu_usage", 0.5, nil),
			},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:   "with type filter",
			method: http.MethodGet,
			queryParams: map[string]string{
				"type": "gauge",
			},
			samples: []metricsdomain.Sample{
				metricsdomain.NewSample(entityID, now, metricsdomain.MetricGauge, "cpu_usage", 0.5, nil),
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
			samples: []metricsdomain.Sample{
				metricsdomain.NewSample(entityID, now, metricsdomain.MetricGauge, "cpu_usage", 0.5, nil),
				metricsdomain.NewSample(entityID, now, metricsdomain.MetricGauge, "memory_usage", 0.7, nil),
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
			samples: []metricsdomain.Sample{
				metricsdomain.NewSample(entityID, now, metricsdomain.MetricGauge, "cpu_usage", 0.5, nil),
				metricsdomain.NewSample(entityID, now, metricsdomain.MetricGauge, "memory_usage", 0.7, nil),
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
			samples: []metricsdomain.Sample{
				metricsdomain.NewSample(entityID, now, metricsdomain.MetricGauge, "cpu_usage", 0.5, nil),
			},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:   "invalid limit",
			method: http.MethodGet,
			queryParams: map[string]string{
				"limit": "invalid",
			},
			samples: []metricsdomain.Sample{
				metricsdomain.NewSample(entityID, now, metricsdomain.MetricGauge, "cpu_usage", 0.5, nil),
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
			repo := &mockMetricsRepository{
				samples: tt.samples,
				err:     tt.repoErr,
			}
			service := api.NewMetricsService(repo)
			handler := NewMetricsHandler(service)

			url := "/api/v1/metrics"
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

			handler.ListSamples(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var samples []api.MetricsSampleResponse
				if err := json.NewDecoder(w.Body).Decode(&samples); err != nil {
					t.Errorf("failed to decode response: %v", err)
				}
				if len(samples) != tt.expectedCount {
					t.Errorf("expected %d samples, got %d", tt.expectedCount, len(samples))
				}
			}
		})
	}
}

