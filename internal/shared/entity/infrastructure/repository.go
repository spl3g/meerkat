package infrastructure

import (
	"context"
	"database/sql"
	"errors"

	"meerkat-v0/internal/infrastructure/database/queries"
	entitydomain "meerkat-v0/internal/shared/entity/domain"
)

type Repository struct {
	readDB  *queries.Queries
	writeDB *queries.Queries
}

func NewRepository(readDB *queries.Queries, writeDB *queries.Queries) *Repository {
	return &Repository{
		readDB:  readDB,
		writeDB: writeDB,
	}
}

func (r *Repository) GetID(ctx context.Context, canonID string) (int64, error) {
	id, err := r.readDB.GetEntityID(ctx, canonID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, entitydomain.ErrIDNotFound
	}
	return id, err
}

func (r *Repository) GetCanonicalID(ctx context.Context, id int64) (string, error) {
	canon, err := r.readDB.GetCanonicalID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", entitydomain.ErrIDNotFound
	}
	return canon, err
}

func (r *Repository) InsertEntity(ctx context.Context, canonID string) (int64, error) {
	id, err := r.writeDB.InsertEntity(ctx, canonID)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) ListEntities(ctx context.Context) ([]entitydomain.Entity, error) {
	entities, err := r.readDB.ListEntities(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]entitydomain.Entity, len(entities))
	for i, e := range entities {
		result[i] = entitydomain.Entity{
			ID:          e.ID,
			CanonicalID: e.CanonicalID,
		}
	}

	return result, nil
}

func (r *Repository) GetEntity(ctx context.Context, canonicalID string) (*entitydomain.Entity, error) {
	entity, err := r.readDB.GetEntityByCanonicalID(ctx, canonicalID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, entitydomain.ErrIDNotFound
	}
	if err != nil {
		return nil, err
	}

	return &entitydomain.Entity{
		ID:          entity.ID,
		CanonicalID: entity.CanonicalID,
	}, nil
}

