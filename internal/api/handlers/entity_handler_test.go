package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	api "meerkat-v0/internal/api/application"
	entitydomain "meerkat-v0/internal/shared/entity/domain"
)

// mockEntityRepository is a mock implementation of entitydomain.Repository
type mockEntityRepository struct {
	entities      []entitydomain.Entity
	getEntityErr  error
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

func TestEntityHandler_ListEntities(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		entities       []entitydomain.Entity
		repoErr        error
		expectedStatus int
		expectedCount  int
	}{
		{
			name:   "empty list",
			method: http.MethodGet,
			entities: []entitydomain.Entity{},
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
		{
			name:   "multiple entities",
			method: http.MethodGet,
			entities: []entitydomain.Entity{
				{ID: 1, CanonicalID: "kind=test|name=entity1"},
				{ID: 2, CanonicalID: "kind=test|name=entity2"},
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:   "repository error",
			method: http.MethodGet,
			repoErr: entitydomain.ErrIDNotFound,
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
			repo := &mockEntityRepository{
				entities:       tt.entities,
				listEntitiesErr: tt.repoErr,
			}
			service := api.NewEntityService(repo)
			handler := NewEntityHandler(service)

			req := httptest.NewRequest(tt.method, "/api/v1/entities", nil)
			w := httptest.NewRecorder()

			handler.ListEntities(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var entities []api.EntityResponse
				if err := json.NewDecoder(w.Body).Decode(&entities); err != nil {
					t.Errorf("failed to decode response: %v", err)
				}
				if len(entities) != tt.expectedCount {
					t.Errorf("expected %d entities, got %d", tt.expectedCount, len(entities))
				}
			}
		})
	}
}

func TestEntityHandler_GetEntity(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		entities       []entitydomain.Entity
		repoErr        error
		expectedStatus int
		expectedID     int64
	}{
		{
			name:   "entity found",
			method: http.MethodGet,
			path:   "/api/v1/entities/kind=test|name=entity1",
			entities: []entitydomain.Entity{
				{ID: 1, CanonicalID: "kind=test|name=entity1"},
			},
			expectedStatus: http.StatusOK,
			expectedID:     1,
		},
		{
			name:   "entity not found",
			method: http.MethodGet,
			path:   "/api/v1/entities/kind=test|name=nonexistent",
			entities: []entitydomain.Entity{
				{ID: 1, CanonicalID: "kind=test|name=entity1"},
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "missing entity ID",
			method: http.MethodGet,
			path:   "/api/v1/entities/",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "repository error",
			method: http.MethodGet,
			path:   "/api/v1/entities/kind=test|name=entity1",
			repoErr: entitydomain.ErrIDNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "method not allowed",
			method:         http.MethodPost,
			path:           "/api/v1/entities/test-id",
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockEntityRepository{
				entities:   tt.entities,
				getEntityErr: tt.repoErr,
			}
			service := api.NewEntityService(repo)
			handler := NewEntityHandler(service)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			// Set up chi router context for URL parameter extraction
			rctx := chi.NewRouteContext()
			// Extract ID from path: /api/v1/entities/{id}
			pathParts := strings.Split(strings.TrimPrefix(tt.path, "/api/v1/entities/"), "/")
			if len(pathParts) > 0 && pathParts[0] != "" {
				rctx.URLParams.Add("id", pathParts[0])
			}
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			handler.GetEntity(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var entity api.EntityResponse
				if err := json.NewDecoder(w.Body).Decode(&entity); err != nil {
					t.Errorf("failed to decode response: %v", err)
				}
				if entity.ID != tt.expectedID {
					t.Errorf("expected ID %d, got %d", tt.expectedID, entity.ID)
				}
			}
		})
	}
}

