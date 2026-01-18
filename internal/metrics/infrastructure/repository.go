package infrastructure

import (
	"context"
	"database/sql"
	"encoding/json"

	"meerkat-v0/internal/infrastructure/database/queries"
	"meerkat-v0/internal/metrics/domain"
	"meerkat-v0/pkg/utils"
	entitydomain "meerkat-v0/internal/shared/entity/domain"
)

// Repository implements the metrics repository interface using SQLite
type Repository struct {
	readDB     *queries.Queries
	writeDB    *queries.Queries
	rawReadDB  *sql.DB
	rawWriteDB *sql.DB
	entityRepo entitydomain.Repository
}

// NewRepository creates a new SQLite metrics repository
func NewRepository(readDB *queries.Queries, writeDB *queries.Queries, rawReadDB *sql.DB, rawWriteDB *sql.DB, entityRepo entitydomain.Repository) *Repository {
	return &Repository{
		readDB:     readDB,
		writeDB:    writeDB,
		rawReadDB:  rawReadDB,
		rawWriteDB: rawWriteDB,
		entityRepo: entityRepo,
	}
}

// InsertSample inserts a metrics sample into the database
func (r *Repository) InsertSample(ctx context.Context, sample domain.Sample) error {
	// Get entity ID (entities are created before metrics start running)
	eId, err := r.writeDB.GetEntityID(ctx, sample.ID.Canonical())
	if err != nil {
		return err
	}

	labels, err := json.Marshal(sample.Labels)
	if err != nil {
		return err
	}

	_, err = r.writeDB.InsertMetrics(ctx, queries.InsertMetricsParams{
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

// ListSamples queries metrics samples with optional filters using sqlc-generated query
func (r *Repository) ListSamples(ctx context.Context, filters domain.SampleFilters) ([]domain.Sample, error) {
	// Use sqlc-generated query with SQLite's NULL handling
	// Build parameters with proper NULL handling for SQLite
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

	var name sql.NullString
	if filters.Name != nil {
		name.String = *filters.Name
		name.Valid = true
	}

	var metricType sql.NullString
	if filters.Type != nil {
		metricType.String = string(*filters.Type)
		metricType.Valid = true
	}

	// Use sqlc-generated query (matching the generated listMetrics constant)
	// Execute through database connection with sql.Null types for proper SQLite NULL handling
	query := `select m.id, m.entity_id, m.ts, m.name, m.type, m.value, m.labels, e.canonical_id
from metrics m
join entities e on m.entity_id = e.id
where (e.canonical_id = ?1 or ?1 is null)
  and (m.ts >= ?2 or ?2 is null)
  and (m.ts <= ?3 or ?3 is null)
  and (m.name = ?4 or ?4 is null)
  and (m.type = ?5 or ?5 is null)
order by m.ts desc
limit ?6 offset ?7`

	rows, err := r.rawReadDB.QueryContext(ctx, query, entityID, from, to, name, metricType, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Use sqlc-generated row type for type-safe scanning
	var samples []domain.Sample
	for rows.Next() {
		var row queries.ListMetricsRow
		if err := rows.Scan(
			&row.ID,
			&row.EntityID,
			&row.Ts,
			&row.Name,
			&row.Type,
			&row.Value,
			&row.Labels,
			&row.CanonicalID,
		); err != nil {
			return nil, err
		}

		var labelsMap map[string]string
		if err := json.Unmarshal(row.Labels, &labelsMap); err != nil {
			return nil, err
		}

		entityIDObj := utils.ParseEntityID(row.CanonicalID)
		samples = append(samples, domain.NewSample(
			entityIDObj,
			row.Ts,
			domain.MetricType(row.Type),
			row.Name,
			row.Value,
			labelsMap,
		))
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return samples, nil
}

