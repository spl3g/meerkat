package utils

import (
	"testing"
)

func TestCheckName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{
			name:      "valid name",
			input:     "test-name",
			wantError: false,
		},
		{
			name:      "valid name with numbers",
			input:     "test123",
			wantError: false,
		},
		{
			name:      "empty name",
			input:     "",
			wantError: true,
		},
		{
			name:      "single character",
			input:     "a",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckName(tt.input)
			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				if err != EmptyNameError {
					t.Errorf("expected EmptyNameError, got %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestEntityID_Canonical(t *testing.T) {
	tests := []struct {
		name     string
		entityID EntityID
		wantKind string
		wantKeys []string
	}{
		{
			name: "simple entity ID",
			entityID: EntityID{
				Kind: "test",
				Labels: map[string]string{
					"name": "entity1",
				},
			},
			wantKind: "test",
			wantKeys: []string{"kind", "name"},
		},
		{
			name: "entity ID with multiple labels",
			entityID: EntityID{
				Kind: "monitor",
				Labels: map[string]string{
					"instance": "prod",
					"service":  "api",
					"type":     "tcp",
					"name":     "check1",
				},
			},
			wantKind: "monitor",
			wantKeys: []string{"kind", "instance", "name", "service", "type"},
		},
		{
			name: "entity ID with no labels",
			entityID: EntityID{
				Kind:   "test",
				Labels: map[string]string{},
			},
			wantKind: "test",
			wantKeys: []string{"kind"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canonical := tt.entityID.Canonical()

			// Verify it contains kind
			if !contains(canonical, "kind="+tt.wantKind) {
				t.Errorf("canonical form %q should contain kind=%q", canonical, tt.wantKind)
			}

			// Verify it contains all expected keys
			for _, key := range tt.wantKeys {
				if key != "kind" {
					if !contains(canonical, key+"=") {
						t.Errorf("canonical form %q should contain key %q", canonical, key)
					}
				}
			}

			// Verify it uses the separator
			if len(tt.entityID.Labels) > 0 && !contains(canonical, IDSeparator) {
				t.Errorf("canonical form %q should contain separator %q", canonical, IDSeparator)
			}
		})
	}
}

func TestParseEntityID(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantKind    string
		wantLabels  map[string]string
		wantError   bool
	}{
		{
			name:     "simple entity ID",
			input:    "kind=test|name=entity1",
			wantKind: "test",
			wantLabels: map[string]string{
				"name": "entity1",
			},
		},
		{
			name:     "entity ID with multiple labels",
			input:    "kind=monitor|instance=prod|service=api|type=tcp|name=check1",
			wantKind: "monitor",
			wantLabels: map[string]string{
				"instance": "prod",
				"service":  "api",
				"type":     "tcp",
				"name":     "check1",
			},
		},
		{
			name:     "entity ID with only kind",
			input:    "kind=test",
			wantKind: "test",
			wantLabels: map[string]string{},
		},
		{
			name:     "empty string",
			input:    "",
			wantKind: "",
			wantLabels: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entityID := ParseEntityID(tt.input)

			if entityID.Kind != tt.wantKind {
				t.Errorf("expected Kind %q, got %q", tt.wantKind, entityID.Kind)
			}

			if len(entityID.Labels) != len(tt.wantLabels) {
				t.Errorf("expected %d labels, got %d", len(tt.wantLabels), len(entityID.Labels))
			}

			for k, v := range tt.wantLabels {
				if entityID.Labels[k] != v {
					t.Errorf("expected label %q=%q, got %q", k, v, entityID.Labels[k])
				}
			}
		})
	}
}

func TestEntityID_RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		id   EntityID
	}{
		{
			name: "simple ID",
			id: EntityID{
				Kind: "test",
				Labels: map[string]string{
					"name": "entity1",
				},
			},
		},
		{
			name: "complex ID",
			id: EntityID{
				Kind: "monitor",
				Labels: map[string]string{
					"instance": "prod",
					"service":  "api",
					"type":     "tcp",
					"name":     "check1",
				},
			},
		},
		{
			name: "ID with no labels",
			id: EntityID{
				Kind:   "test",
				Labels: map[string]string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to canonical
			canonical := tt.id.Canonical()

			// Parse back
			parsed := ParseEntityID(canonical)

			// Verify kind matches
			if parsed.Kind != tt.id.Kind {
				t.Errorf("round-trip failed: expected Kind %q, got %q", tt.id.Kind, parsed.Kind)
			}

			// Verify labels match (note: order may differ in canonical form)
			if len(parsed.Labels) != len(tt.id.Labels) {
				t.Errorf("round-trip failed: expected %d labels, got %d", len(tt.id.Labels), len(parsed.Labels))
			}

			for k, v := range tt.id.Labels {
				if parsed.Labels[k] != v {
					t.Errorf("round-trip failed: expected label %q=%q, got %q", k, v, parsed.Labels[k])
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

