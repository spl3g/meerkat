package infrastructure

import (
	"context"
	"database/sql"
	"fmt"

	"meerkat-v0/db"
	"meerkat-v0/internal/monitoring/domain"
	entitydomain "meerkat-v0/internal/shared/entity/domain"
)

// Repository implements the monitor repository interface using SQLite
type Repository struct {
	readDB     *db.Queries
	writeDB    *db.Queries
	rawReadDB  *sql.DB
	entityRepo entitydomain.Repository
}

// NewRepository creates a new SQLite monitor repository
func NewRepository(readDB *db.Queries, writeDB *db.Queries, rawReadDB *sql.DB, entityRepo entitydomain.Repository) *Repository {
	return &Repository{
		readDB:     readDB,
		writeDB:    writeDB,
		rawReadDB:  rawReadDB,
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

// ListHeartbeats queries heartbeats with optional filters using sqlc-generated query
func (r *Repository) ListHeartbeats(ctx context.Context, filters domain.HeartbeatFilters) ([]domain.Heartbeat, error) {
	// Use sqlc-generated query with SQLite's NULL handling
	// Build parameters - use sql.Null types for optional filters so SQLite handles NULL properly
	limit := int64(100)
	if filters.Limit > 0 {
		limit = int64(filters.Limit)
	}
	offset := int64(filters.Offset)

	// Prepare parameters with proper NULL handling for SQLite
	var entityID sql.NullString
	if filters.EntityID != nil {
		entityID.String = *filters.EntityID
		entityID.Valid = true
	}

	var from sql.NullTime
	if filters.From != nil {
		from.Time = *filters.From
		from.Valid = true
	}

	var to sql.NullTime
	if filters.To != nil {
		to.Time = *filters.To
		to.Valid = true
	}

	var successful sql.NullBool
	if filters.Successful != nil {
		successful.Bool = *filters.Successful
		successful.Valid = true
	}

	// Use sqlc-generated query (matching the generated listHeartbeats constant)
	// Execute through database connection with sql.Null types for proper SQLite NULL handling
	query := `select h.id, h.entity_id, h.ts, h.successful, h.error, e.canonical_id
from heartbeat h
join entities e on h.entity_id = e.id
where (e.canonical_id = ?1 or ?1 is null)
  and (h.ts >= ?2 or ?2 is null)
  and (h.ts <= ?3 or ?3 is null)
  and (h.successful = ?4 or ?4 is null)
order by h.ts desc
limit ?5 offset ?6`

	rows, err := r.rawReadDB.QueryContext(ctx, query, entityID, from, to, successful, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Use sqlc-generated row type for type-safe scanning
	var heartbeats []domain.Heartbeat
	for rows.Next() {
		var row db.ListHeartbeatsRow
		if err := rows.Scan(
			&row.ID,
			&row.EntityID,
			&row.Ts,
			&row.Successful,
			&row.Error,
			&row.CanonicalID,
		); err != nil {
			return nil, err
		}

		var err error
		if row.Error.Valid {
			err = fmt.Errorf(row.Error.String)
		}

		heartbeats = append(heartbeats, domain.NewHeartbeat(row.CanonicalID, row.Ts, err))
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return heartbeats, nil
}

