package infrastructure

import (
	"context"
	"database/sql"
	"errors"

	"meerkat-v0/db"
	entitydomain "meerkat-v0/internal/shared/entity/domain"
)

type Repository struct {
	readDB  *db.Queries
	writeDB *db.Queries
}

func NewRepository(readDB *db.Queries, writeDB *db.Queries) *Repository {
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

