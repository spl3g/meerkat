package infrastructure

import (
	"context"
	"database/sql"

	"meerkat-v0/db"
	"meerkat-v0/internal/monitoring/domain"
	entitydomain "meerkat-v0/internal/shared/entity/domain"
)

// Repository implements the monitor repository interface using SQLite
type Repository struct {
	readDB     *db.Queries
	writeDB    *db.Queries
	entityRepo entitydomain.Repository
}

// NewRepository creates a new SQLite monitor repository
func NewRepository(readDB *db.Queries, writeDB *db.Queries, entityRepo entitydomain.Repository) *Repository {
	return &Repository{
		readDB:     readDB,
		writeDB:    writeDB,
		entityRepo: entityRepo,
	}
}

// InsertHeartbeat inserts a heartbeat into the database
func (r *Repository) InsertHeartbeat(ctx context.Context, heartbeat domain.Heartbeat) error {
	eId, err := r.readDB.GetEntityID(ctx, heartbeat.MonitorID)
	if err != nil {
		return err
	}

	var successful bool = true
	var error sql.NullString
	if heartbeat.Error != nil {
		successful = false
		error.String = heartbeat.Error.Error()
		error.Valid = true
	}

	_, err = r.writeDB.InsertHeartbeat(ctx, db.InsertHeartbeatParams{
		EntityID:   eId,
		Ts:         heartbeat.Timestamp,
		Successful: successful,
		Error:      error,
	})
	if err != nil {
		return err
	}

	return nil
}

