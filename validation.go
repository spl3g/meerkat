package main

import (
	"context"
	"fmt"
	"strings"
)

type ValidationError struct {
	Path     string
	Problems map[string]string
}

func NewValidationError(problems map[string]string, path ...string) *ValidationError {
	return &ValidationError{strings.Join(path, "."), problems}
}

func (e *ValidationError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "validation errors found in '%s':\n", e.Path)
	for field, problem := range e.Problems {
		fmt.Fprintf(&b, "  %s: %s\n", field, problem)
	}
	return b.String()
}

func (e *ValidationError) Is(other error) bool {
	_, ok := other.(*ValidationError)
	return ok
}

func (e *ValidationError) PrependPath(path string) *ValidationError {
	e.Path = fmt.Sprint(path, ".", e.Path)
	return e
}

type Validator interface {
	// Returns a map of field and human readable explanation of what's wrong
	Valid(ctx context.Context) (problems map[string]string)
}
