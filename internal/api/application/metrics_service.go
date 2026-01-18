package application

import (
	"context"

	metricsdomain "meerkat-v0/internal/metrics/domain"
)

// MetricsService handles metrics sample queries
type MetricsService struct {
	repo metricsdomain.Repository
}

// NewMetricsService creates a new metrics service
func NewMetricsService(repo metricsdomain.Repository) *MetricsService {
	return &MetricsService{
		repo: repo,
	}
}

// ListSamples returns metrics samples matching the filters
func (s *MetricsService) ListSamples(ctx context.Context, req ListSamplesRequest) ([]MetricsSampleResponse, error) {
	var metricType *metricsdomain.MetricType
	if req.Type != nil {
		mt := metricsdomain.MetricType(*req.Type)
		metricType = &mt
	}

	filters := metricsdomain.SampleFilters{
		EntityID: req.EntityID,
		From:     req.From,
		To:       req.To,
		Name:     req.Name,
		Type:     metricType,
		Limit:    req.Limit,
		Offset:   req.Offset,
	}

	if filters.Limit <= 0 {
		filters.Limit = 100
	}

	samples, err := s.repo.ListSamples(ctx, filters)
	if err != nil {
		return nil, err
	}

	responses := make([]MetricsSampleResponse, len(samples))
	for i, sample := range samples {
		responses[i] = ToMetricsSampleResponse(sample)
	}

	return responses, nil
}

