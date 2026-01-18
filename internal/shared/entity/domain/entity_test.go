package domain

import (
	"testing"

	"meerkat-v0/pkg/utils"
)

func TestNewEntity(t *testing.T) {
	canonicalID := "kind=test|name=entity1"
	entity := NewEntity(canonicalID)

	if entity.CanonicalID != canonicalID {
		t.Errorf("expected CanonicalID %q, got %q", canonicalID, entity.CanonicalID)
	}

	if entity.ID != 0 {
		t.Errorf("expected ID to be 0 for new entity, got %d", entity.ID)
	}
}

func TestEntity_EntityID(t *testing.T) {
	tests := []struct {
		name        string
		canonicalID string
		expectedKind string
		expectedLabels map[string]string
	}{
		{
			name:        "valid entity ID",
			canonicalID: "kind=test|name=entity1|instance=prod",
			expectedKind: "test",
			expectedLabels: map[string]string{
				"name":     "entity1",
				"instance": "prod",
			},
		},
		{
			name:        "minimal entity ID",
			canonicalID: "kind=test",
			expectedKind: "test",
			expectedLabels: map[string]string{},
		},
		{
			name:        "empty canonical ID",
			canonicalID: "",
			expectedKind: "",
			expectedLabels: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entity := NewEntity(tt.canonicalID)
			entityID := entity.EntityID()

			if entityID.Kind != tt.expectedKind {
				t.Errorf("expected Kind %q, got %q", tt.expectedKind, entityID.Kind)
			}

			if len(entityID.Labels) != len(tt.expectedLabels) {
				t.Errorf("expected %d labels, got %d", len(tt.expectedLabels), len(entityID.Labels))
			}

			for k, v := range tt.expectedLabels {
				if entityID.Labels[k] != v {
					t.Errorf("expected label %q=%q, got %q", k, v, entityID.Labels[k])
				}
			}
		})
	}
}

func TestEntity_RoundTrip(t *testing.T) {
	// Test that EntityID conversion is consistent
	canonicalID := "kind=test|name=entity1|instance=prod"
	entity := NewEntity(canonicalID)
	entityID := entity.EntityID()

	// Convert back to canonical
	canonical := entityID.Canonical()

	// Parse the canonical ID to verify it matches
	parsed := utils.ParseEntityID(canonical)
	if parsed.Kind != entityID.Kind {
		t.Errorf("round-trip failed: expected Kind %q, got %q", entityID.Kind, parsed.Kind)
	}

	// Note: The canonical form may have different key ordering, so we check labels individually
	for k, v := range entityID.Labels {
		if parsed.Labels[k] != v {
			t.Errorf("round-trip failed: expected label %q=%q, got %q", k, v, parsed.Labels[k])
		}
	}
}

