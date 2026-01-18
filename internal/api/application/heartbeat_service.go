package application

import (
	"context"

	monitoringdomain "meerkat-v0/internal/monitoring/domain"
)

// HeartbeatService handles heartbeat queries
type HeartbeatService struct {
	repo monitoringdomain.Repository
}

// NewHeartbeatService creates a new heartbeat service
func NewHeartbeatService(repo monitoringdomain.Repository) *HeartbeatService {
	return &HeartbeatService{
		repo: repo,
	}
}

// ListHeartbeats returns heartbeats matching the filters
func (s *HeartbeatService) ListHeartbeats(ctx context.Context, req ListHeartbeatsRequest) ([]HeartbeatResponse, error) {
	filters := monitoringdomain.HeartbeatFilters{
		EntityID:   req.EntityID,
		From:       req.From,
		To:         req.To,
		Successful: req.Successful,
		Limit:      req.Limit,
		Offset:     req.Offset,
	}

	if filters.Limit <= 0 {
		filters.Limit = 100
	}

	heartbeats, err := s.repo.ListHeartbeats(ctx, filters)
	if err != nil {
		return nil, err
	}

	responses := make([]HeartbeatResponse, len(heartbeats))
	for i, h := range heartbeats {
		responses[i] = ToHeartbeatResponse(h)
	}

	return responses, nil
}

