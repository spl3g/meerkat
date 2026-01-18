package domain

import "meerkat-v0/pkg/utils"

// Entity represents a tracked entity in the system
type Entity struct {
	ID          int64
	CanonicalID string
}

// NewEntity creates a new entity with a canonical ID
func NewEntity(canonicalID string) *Entity {
	return &Entity{
		CanonicalID: canonicalID,
	}
}

// EntityID creates an EntityID from the canonical ID string
func (e *Entity) EntityID() utils.EntityID {
	return utils.ParseEntityID(e.CanonicalID)
}

