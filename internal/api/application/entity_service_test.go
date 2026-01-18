package application

import (
	"context"
	"errors"
	"testing"

	entitydomain "meerkat-v0/internal/shared/entity/domain"
)

// mockEntityRepository is a mock implementation of entitydomain.Repository
type mockEntityRepository struct {
	entities       []entitydomain.Entity
	getEntityErr   error
	listEntitiesErr error
}

func (m *mockEntityRepository) GetID(ctx context.Context, canonID string) (int64, error) {
	for _, e := range m.entities {
		if e.CanonicalID == canonID {
			return e.ID, nil
		}
	}
	return 0, entitydomain.ErrIDNotFound
}

func (m *mockEntityRepository) InsertEntity(ctx context.Context, canonID string) (int64, error) {
	id := int64(len(m.entities) + 1)
	m.entities = append(m.entities, entitydomain.Entity{
		ID:          id,
		CanonicalID: canonID,
	})
	return id, nil
}

func (m *mockEntityRepository) GetCanonicalID(ctx context.Context, id int64) (string, error) {
	for _, e := range m.entities {
		if e.ID == id {
			return e.CanonicalID, nil
		}
	}
	return "", entitydomain.ErrIDNotFound
}

func (m *mockEntityRepository) ListEntities(ctx context.Context) ([]entitydomain.Entity, error) {
	if m.listEntitiesErr != nil {
		return nil, m.listEntitiesErr
	}
	return m.entities, nil
}

func (m *mockEntityRepository) GetEntity(ctx context.Context, canonicalID string) (*entitydomain.Entity, error) {
	if m.getEntityErr != nil {
		return nil, m.getEntityErr
	}
	for _, e := range m.entities {
		if e.CanonicalID == canonicalID {
			return &e, nil
		}
	}
	return nil, entitydomain.ErrIDNotFound
}

func TestEntityService_ListEntities(t *testing.T) {
	tests := []struct {
		name           string
		entities       []entitydomain.Entity
		repoErr        error
		expectedCount  int
		expectError    bool
	}{
		{
			name:          "empty list",
			entities:      []entitydomain.Entity{},
			expectedCount: 0,
		},
		{
			name: "multiple entities",
			entities: []entitydomain.Entity{
				{ID: 1, CanonicalID: "kind=test|name=entity1"},
				{ID: 2, CanonicalID: "kind=test|name=entity2"},
			},
			expectedCount: 2,
		},
		{
			name:        "repository error",
			repoErr:     errors.New("database error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockEntityRepository{
				entities:       tt.entities,
				listEntitiesErr: tt.repoErr,
			}
			service := NewEntityService(repo)

			entities, err := service.ListEntities(context.Background())

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(entities) != tt.expectedCount {
				t.Errorf("expected %d entities, got %d", tt.expectedCount, len(entities))
			}

			// Verify DTO conversion
			for i, entity := range entities {
				if entity.ID != tt.entities[i].ID {
					t.Errorf("expected ID %d, got %d", tt.entities[i].ID, entity.ID)
				}
				if entity.CanonicalID != tt.entities[i].CanonicalID {
					t.Errorf("expected CanonicalID %q, got %q", tt.entities[i].CanonicalID, entity.CanonicalID)
				}
			}
		})
	}
}

func TestEntityService_GetEntity(t *testing.T) {
	tests := []struct {
		name           string
		canonicalID    string
		entities       []entitydomain.Entity
		repoErr        error
		expectedID     int64
		expectError    bool
		expectedErrMsg string
	}{
		{
			name:        "entity found",
			canonicalID: "kind=test|name=entity1",
			entities: []entitydomain.Entity{
				{ID: 1, CanonicalID: "kind=test|name=entity1"},
			},
			expectedID: 1,
		},
		{
			name:        "entity not found",
			canonicalID: "kind=test|name=nonexistent",
			entities: []entitydomain.Entity{
				{ID: 1, CanonicalID: "kind=test|name=entity1"},
			},
			expectError:    true,
			expectedErrMsg: "could not find entity with this id",
		},
		{
			name:        "repository error",
			canonicalID: "kind=test|name=entity1",
			repoErr:     errors.New("database error"),
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockEntityRepository{
				entities:     tt.entities,
				getEntityErr: tt.repoErr,
			}
			service := NewEntityService(repo)

			entity, err := service.GetEntity(context.Background(), tt.canonicalID)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}
				if tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
					t.Errorf("expected error message %q, got %q", tt.expectedErrMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if entity.ID != tt.expectedID {
				t.Errorf("expected ID %d, got %d", tt.expectedID, entity.ID)
			}

			if entity.CanonicalID != tt.canonicalID {
				t.Errorf("expected CanonicalID %q, got %q", tt.canonicalID, entity.CanonicalID)
			}
		})
	}
}

func TestToEntityResponse(t *testing.T) {
	entity := entitydomain.Entity{
		ID:          42,
		CanonicalID: "kind=test|name=entity1",
	}

	response := ToEntityResponse(entity)

	if response.ID != entity.ID {
		t.Errorf("expected ID %d, got %d", entity.ID, response.ID)
	}

	if response.CanonicalID != entity.CanonicalID {
		t.Errorf("expected CanonicalID %q, got %q", entity.CanonicalID, response.CanonicalID)
	}
}

