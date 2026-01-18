package application

import (
	"context"

	entitydomain "meerkat-v0/internal/shared/entity/domain"
)

// EntityService handles entity queries
type EntityService struct {
	repo entitydomain.Repository
}

// NewEntityService creates a new entity service
func NewEntityService(repo entitydomain.Repository) *EntityService {
	return &EntityService{
		repo: repo,
	}
}

// ListEntities returns all entities
func (s *EntityService) ListEntities(ctx context.Context) ([]EntityResponse, error) {
	entities, err := s.repo.ListEntities(ctx)
	if err != nil {
		return nil, err
	}

	responses := make([]EntityResponse, len(entities))
	for i, e := range entities {
		responses[i] = ToEntityResponse(e)
	}

	return responses, nil
}

// GetEntity returns an entity by canonical ID
func (s *EntityService) GetEntity(ctx context.Context, canonicalID string) (*EntityResponse, error) {
	entity, err := s.repo.GetEntity(ctx, canonicalID)
	if err != nil {
		return nil, err
	}

	response := ToEntityResponse(*entity)
	return &response, nil
}

