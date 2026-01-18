package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"time"

	"meerkat-v0/db"
)

type Heartbeat struct {
	MonitorID string
	Timestamp time.Time
	Error     error
}

type HeartbeatRepo interface {
	InsertHeartbeat(context.Context, Heartbeat) error
}

type WriterHeartbeat struct {
	w io.Writer
}

func (h *WriterHeartbeat) InsertHeartbeat(ctx context.Context, heartbeat Heartbeat) error {
	fmt.Fprintf(h.w, "%s [%s]: ", heartbeat.Timestamp.String(), heartbeat.MonitorID)
	if heartbeat.Error != nil {
		fmt.Fprintf(h.w, "error: %s\n", heartbeat.Error)
	} else {
		fmt.Fprintf(h.w, "OK\n")
	}
	return nil
}

type SqliteHeartbeatRepo struct {
	readDB     *db.Queries
	writeDB    *db.Queries
	entityRepo EntityRepo
}

func NewSqliteHeartbeatRepo(readDB *db.Queries, writeDB *db.Queries, entityRepo EntityRepo) *SqliteHeartbeatRepo {
	return &SqliteHeartbeatRepo{
		readDB:     readDB,
		writeDB:    writeDB,
		entityRepo: entityRepo,
	}
}

func (r *SqliteHeartbeatRepo) InsertHeartbeat(ctx context.Context, heartbeat Heartbeat) error {
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
