package main

import (
	"context"
	"fmt"
	"io"
	"time"
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
