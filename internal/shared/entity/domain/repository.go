package domain

import (
	"context"
	"errors"
)

var ErrIDNotFound = errors.New("could not find entity with this id")

type Repository interface {
	GetID(ctx context.Context, canonID string) (int64, error)
	InsertEntity(ctx context.Context, canonID string) (int64, error)
	GetCanonicalID(ctx context.Context, id int64) (string, error)
}

