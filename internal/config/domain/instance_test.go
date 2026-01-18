package domain

import (
	"context"
	"encoding/json"
	"testing"
)

func TestInstanceConfig_Valid(t *testing.T) {
	tests := []struct {
		name      string
		config    InstanceConfig
		wantError bool
		errorKeys []string
	}{
		{
			name: "valid config",
			config: InstanceConfig{
				Name: "test-instance",
				Services: []json.RawMessage{
					json.RawMessage(`{"name": "service1"}`),
				},
			},
			wantError: false,
		},
		{
			name: "empty name",
			config: InstanceConfig{
				Name: "",
				Services: []json.RawMessage{
					json.RawMessage(`{"name": "service1"}`),
				},
			},
			wantError: true,
			errorKeys: []string{"name"},
		},
		{
			name: "empty services",
			config: InstanceConfig{
				Name:     "test-instance",
				Services: []json.RawMessage{},
			},
			wantError: true,
			errorKeys: []string{"services"},
		},
		{
			name: "nil services",
			config: InstanceConfig{
				Name:     "test-instance",
				Services: nil,
			},
			wantError: true,
			errorKeys: []string{"services"},
		},
		{
			name: "multiple validation errors",
			config: InstanceConfig{
				Name:     "",
				Services: []json.RawMessage{},
			},
			wantError: true,
			errorKeys: []string{"name", "services"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			problems := tt.config.Valid(context.Background())

			if tt.wantError {
				if len(problems) == 0 {
					t.Errorf("expected validation errors, got none")
					return
				}

				for _, key := range tt.errorKeys {
					if _, ok := problems[key]; !ok {
						t.Errorf("expected error for key %q, but it was not found", key)
					}
				}
			} else {
				if len(problems) > 0 {
					t.Errorf("expected no validation errors, got: %v", problems)
				}
			}
		})
	}
}

