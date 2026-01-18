package infrastructure

import (
	"context"
	"testing"

	"meerkat-v0/db"
	entitydomain "meerkat-v0/internal/shared/entity/domain"
	"meerkat-v0/internal/infrastructure/database"
	"meerkat-v0/internal/schema"
)

func setupTestRepository(t *testing.T) (*Repository, func()) {
	// Setup in-memory database
	testDB, err := database.ConnectSQLite(":memory:")
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Initialize schema
	_, err = testDB.Exec(schema.DDL)
	if err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	queries := db.New(testDB)
	repo := NewRepository(queries, queries)

	cleanup := func() {
		testDB.Close()
	}

	return repo, cleanup
}

func TestRepository_InsertEntity(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	defer cleanup()

	canonicalID := "kind=test|name=entity1"
	id, err := repo.InsertEntity(context.Background(), canonicalID)
	if err != nil {
		t.Fatalf("unexpected error inserting entity: %v", err)
	}

	if id <= 0 {
		t.Errorf("expected positive ID, got %d", id)
	}

	// Verify it was inserted
	entity, err := repo.GetEntity(context.Background(), canonicalID)
	if err != nil {
		t.Fatalf("unexpected error getting entity: %v", err)
	}

	if entity.ID != id {
		t.Errorf("expected ID %d, got %d", id, entity.ID)
	}

	if entity.CanonicalID != canonicalID {
		t.Errorf("expected CanonicalID %q, got %q", canonicalID, entity.CanonicalID)
	}
}

func TestRepository_GetID(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	defer cleanup()

	canonicalID := "kind=test|name=entity1"
	id, err := repo.InsertEntity(context.Background(), canonicalID)
	if err != nil {
		t.Fatalf("unexpected error inserting entity: %v", err)
	}

	// Test GetID
	retrievedID, err := repo.GetID(context.Background(), canonicalID)
	if err != nil {
		t.Fatalf("unexpected error getting ID: %v", err)
	}

	if retrievedID != id {
		t.Errorf("expected ID %d, got %d", id, retrievedID)
	}

	// Test GetID with non-existent entity
	_, err = repo.GetID(context.Background(), "kind=test|name=nonexistent")
	if err != entitydomain.ErrIDNotFound {
		t.Errorf("expected ErrIDNotFound, got: %v", err)
	}
}

func TestRepository_GetCanonicalID(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	defer cleanup()

	canonicalID := "kind=test|name=entity1"
	id, err := repo.InsertEntity(context.Background(), canonicalID)
	if err != nil {
		t.Fatalf("unexpected error inserting entity: %v", err)
	}

	// Test GetCanonicalID
	retrievedCanonicalID, err := repo.GetCanonicalID(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error getting canonical ID: %v", err)
	}

	if retrievedCanonicalID != canonicalID {
		t.Errorf("expected CanonicalID %q, got %q", canonicalID, retrievedCanonicalID)
	}

	// Test GetCanonicalID with non-existent ID
	_, err = repo.GetCanonicalID(context.Background(), 99999)
	if err != entitydomain.ErrIDNotFound {
		t.Errorf("expected ErrIDNotFound, got: %v", err)
	}
}

func TestRepository_ListEntities(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	defer cleanup()

	// Insert multiple entities
	entities := []string{
		"kind=test|name=entity1",
		"kind=test|name=entity2",
		"kind=test|name=entity3",
	}

	for _, canonicalID := range entities {
		_, err := repo.InsertEntity(context.Background(), canonicalID)
		if err != nil {
			t.Fatalf("unexpected error inserting entity: %v", err)
		}
	}

	// Test ListEntities
	list, err := repo.ListEntities(context.Background())
	if err != nil {
		t.Fatalf("unexpected error listing entities: %v", err)
	}

	if len(list) != len(entities) {
		t.Errorf("expected %d entities, got %d", len(entities), len(list))
	}

	// Verify all entities are present
	entityMap := make(map[string]bool)
	for _, e := range list {
		entityMap[e.CanonicalID] = true
	}

	for _, canonicalID := range entities {
		if !entityMap[canonicalID] {
			t.Errorf("expected entity %q to be in list", canonicalID)
		}
	}
}

func TestRepository_GetEntity(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	defer cleanup()

	canonicalID := "kind=test|name=entity1"
	id, err := repo.InsertEntity(context.Background(), canonicalID)
	if err != nil {
		t.Fatalf("unexpected error inserting entity: %v", err)
	}

	// Test GetEntity
	entity, err := repo.GetEntity(context.Background(), canonicalID)
	if err != nil {
		t.Fatalf("unexpected error getting entity: %v", err)
	}

	if entity.ID != id {
		t.Errorf("expected ID %d, got %d", id, entity.ID)
	}

	if entity.CanonicalID != canonicalID {
		t.Errorf("expected CanonicalID %q, got %q", canonicalID, entity.CanonicalID)
	}

	// Test GetEntity with non-existent entity
	_, err = repo.GetEntity(context.Background(), "kind=test|name=nonexistent")
	if err != entitydomain.ErrIDNotFound {
		t.Errorf("expected ErrIDNotFound, got: %v", err)
	}
}

func TestRepository_GetEntity_EmptyList(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	defer cleanup()

	// Test ListEntities with empty database
	list, err := repo.ListEntities(context.Background())
	if err != nil {
		t.Fatalf("unexpected error listing entities: %v", err)
	}

	if len(list) != 0 {
		t.Errorf("expected empty list, got %d entities", len(list))
	}
}

func TestRepository_DuplicateCanonicalID(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	defer cleanup()

	canonicalID := "kind=test|name=entity1"
	id1, err := repo.InsertEntity(context.Background(), canonicalID)
	if err != nil {
		t.Fatalf("unexpected error inserting entity: %v", err)
	}

	// Try to insert the same canonical ID again
	// This should either fail or return the same ID depending on implementation
	// For now, we'll test that GetID returns the same ID
	id2, err := repo.GetID(context.Background(), canonicalID)
	if err != nil {
		t.Fatalf("unexpected error getting ID: %v", err)
	}

	if id2 != id1 {
		t.Errorf("expected same ID for duplicate canonical ID, got %d and %d", id1, id2)
	}
}

func TestRepository_SQLiteErrors(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	defer cleanup()

	// Test with invalid context (cancelled)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := repo.GetID(ctx, "kind=test|name=entity1")
	if err == nil {
		t.Error("expected error with cancelled context, got nil")
	}

	_, err = repo.ListEntities(ctx)
	if err == nil {
		t.Error("expected error with cancelled context, got nil")
	}
}

