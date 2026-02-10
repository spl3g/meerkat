package infrastructure

import (
	"context"
	"encoding/json"

	"meerkat-v0/db"
	"meerkat-v0/internal/metrics/domain"
	entitydomain "meerkat-v0/internal/shared/entity/domain"
)

// Repository implements the metrics repository interface using SQLite
type Repository struct {
	readDB     *db.Queries
	writeDB    *db.Queries
	entityRepo entitydomain.Repository
}

// NewRepository creates a new SQLite metrics repository
func NewRepository(readDB *db.Queries, writeDB *db.Queries, entityRepo entitydomain.Repository) *Repository {
	return &Repository{
		readDB:     readDB,
		writeDB:    writeDB,
		entityRepo: entityRepo,
	}
}

// InsertSample inserts a metrics sample into the database
func (r *Repository) InsertSample(ctx context.Context, sample domain.Sample) error {
	eId, err := r.readDB.GetEntityID(ctx, sample.ID.Canonical())
	if err != nil {
		return err
	}

	labels, err := json.Marshal(sample.Labels)
	if err != nil {
		return err
	}

	_, err = r.writeDB.InsertMetrics(ctx, db.InsertMetricsParams{
		EntityID: eId,
		Ts:       sample.Timestamp,
		Type:     string(sample.Type),
		Value:    sample.Value,
		Name:     sample.Name,
		Labels:   labels,
	})
	if err != nil {
		return err
	}

	return nil
}

