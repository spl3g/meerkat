package validation

import (
	"errors"
	"strings"
	"testing"
)

func TestValidationError(t *testing.T) {
	tests := []struct {
		name      string
		problems  map[string]string
		path      []string
		wantError bool
		wantMsg   string
	}{
		{
			name: "single problem",
			problems: map[string]string{
				"name": "name is required",
			},
			path:      []string{"config"},
			wantError: true,
			wantMsg:   "validation errors found in 'config'",
		},
		{
			name: "multiple problems",
			problems: map[string]string{
				"name":     "name is required",
				"services": "services cannot be empty",
			},
			path:      []string{"config"},
			wantError: true,
			wantMsg:   "validation errors found in 'config'",
		},
		{
			name:      "empty problems",
			problems:  map[string]string{},
			path:      []string{"config"},
			wantError: true,
			wantMsg:   "validation errors found in 'config'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidationError(tt.problems, tt.path...)

			if err == nil {
				if tt.wantError {
					t.Errorf("expected error, got nil")
				}
				return
			}

			msg := err.Error()
			if !strings.Contains(msg, tt.wantMsg) {
				t.Errorf("expected error message to contain %q, got %q", tt.wantMsg, msg)
			}

			// Check that all problems are in the error message
			for field, problem := range tt.problems {
				if !strings.Contains(msg, field) {
					t.Errorf("expected error message to contain field %q", field)
				}
				if !strings.Contains(msg, problem) {
					t.Errorf("expected error message to contain problem %q", problem)
				}
			}
		})
	}
}

func TestValidationError_Is(t *testing.T) {
	err1 := NewValidationError(map[string]string{"name": "required"}, "config")
	err2 := NewValidationError(map[string]string{"services": "empty"}, "config")
	var validationErr *ValidationError

	if !errors.Is(err1, err2) {
		t.Error("expected ValidationError.Is to return true for another ValidationError")
	}

	if !errors.As(err1, &validationErr) {
		t.Error("expected errors.As to work with ValidationError")
	}
}

func TestValidationError_PrependPath(t *testing.T) {
	err := NewValidationError(map[string]string{"name": "required"}, "service")
	err = err.PrependPath("instance").(*ValidationError)

	msg := err.Error()
	if !strings.Contains(msg, "instance.service") {
		t.Errorf("expected error message to contain 'instance.service', got %q", msg)
	}
}

func TestValidationError_AppendPath(t *testing.T) {
	err := NewValidationError(map[string]string{"name": "required"}, "service")
	err = err.AppendPath("monitor").(*ValidationError)

	msg := err.Error()
	if !strings.Contains(msg, "service.monitor") {
		t.Errorf("expected error message to contain 'service.monitor', got %q", msg)
	}
}

func TestDuplicateFoundError(t *testing.T) {
	tests := []struct {
		name    string
		path    []string
		wantMsg string
	}{
		{
			name:    "single path",
			path:    []string{"service"},
			wantMsg: "duplicate entity in 'service'",
		},
		{
			name:    "multiple path segments",
			path:    []string{"instance", "service", "0"},
			wantMsg: "duplicate entity in 'instance.service.0'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewDuplicateFoundError(tt.path...)
			if err == nil {
				t.Error("expected error, got nil")
				return
			}

			msg := err.Error()
			if msg != tt.wantMsg {
				t.Errorf("expected error message %q, got %q", tt.wantMsg, msg)
			}
		})
	}
}

func TestDuplicateFoundError_PrependPath(t *testing.T) {
	err := NewDuplicateFoundError("service", "0")
	err = err.PrependPath("instance").(*DuplicateFoundError)

	msg := err.Error()
	if !strings.Contains(msg, "instance.service.0") {
		t.Errorf("expected error message to contain 'instance.service.0', got %q", msg)
	}
}

func TestNoNameError(t *testing.T) {
	tests := []struct {
		name    string
		path    []string
		index   int
		wantMsg string
	}{
		{
			name:    "no index",
			path:    []string{"service"},
			index:   -1,
			wantMsg: "entity in 'service' has no name",
		},
		{
			name:    "with index",
			path:    []string{"service"},
			index:   0,
			wantMsg: "entity in 'service[0]' has no name",
		},
		{
			name:    "multiple path segments",
			path:    []string{"instance", "service"},
			index:   1,
			wantMsg: "entity in 'instance.service[1]' has no name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewNoNameError(tt.path...)
			if tt.index >= 0 {
				err.SetIndex(tt.index)
			}

			if err == nil {
				t.Error("expected error, got nil")
				return
			}

			msg := err.Error()
			if msg != tt.wantMsg {
				t.Errorf("expected error message %q, got %q", tt.wantMsg, msg)
			}
		})
	}
}

func TestNoNameError_PrependPath(t *testing.T) {
	err := NewNoNameError("service", "0")
	err.SetIndex(1)
	err = err.PrependPath("instance").(*NoNameError)

	msg := err.Error()
	if !strings.Contains(msg, "instance.service.0[1]") {
		t.Errorf("expected error message to contain 'instance.service.0[1]', got %q", msg)
	}
}

