package validation

import (
	"context"
	"fmt"
	"strings"
)

type ConfigError interface {
	error
	PrependPath(path string) ConfigError
}

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

func (e *ValidationError) PrependPath(path string) ConfigError {
	e.Path = fmt.Sprint(path, ".", e.Path)
	return e
}

func (e *ValidationError) AppendPath(path string) ConfigError {
	e.Path = fmt.Sprint(e.Path, ".", path)
	return e
}

type Validator interface {
	// Returns a map of field and human readable explanation of what's wrong
	Valid(ctx context.Context) (problems map[string]string)
}

type DuplicateFoundError struct {
	Path string
}

func NewDuplicateFoundError(path ...string) *DuplicateFoundError {
	return &DuplicateFoundError{strings.Join(path, ".")}
}

func (e *DuplicateFoundError) Error() string {
	return fmt.Sprintf("duplicate entity in '%s'", e.Path)
}

type NoNameError struct {
	Path  string
	Index int
}

func NewNoNameError(path ...string) *NoNameError {
	return &NoNameError{strings.Join(path, "."), -1}
}

func (e *NoNameError) Error() string {
	var path string
	if e.Index >= 0 {
		path = fmt.Sprintf("%s[%d]", e.Path, e.Index)
	} else {
		path = e.Path
	}

	return fmt.Sprintf("entity in '%s' has no name", path)
}

func (e *NoNameError) SetIndex(i int) {
	e.Index = i
}

func (e *NoNameError) PrependPath(path string) ConfigError {
	e.Path = fmt.Sprint(path, ".", e.Path)
	return e
}

