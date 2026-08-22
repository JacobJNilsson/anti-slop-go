package a

import (
	"errors"
	"fmt"
	"strings"
)

// ErrEmptyRunID is the sentinel that a test asserts with errors.Is.
var ErrEmptyRunID = errors.New("run id must be non-empty")

// ParseError carries a line, so a test asserts it with errors.As.
type ParseError struct{ Line int }

// Error returns the message of the parse error.
func (e *ParseError) Error() string { return fmt.Sprintf("parse error on line %d", e.Line) }

// Seed wraps the sentinel for an empty run id.
func Seed(id string) error {
	if id == "" {
		return fmt.Errorf("seed: %w", ErrEmptyRunID)
	}

	return nil
}

// Parse wraps a parse error.
func Parse() error { return fmt.Errorf("parse: %w", &ParseError{Line: 7}) }

// Describe holds the shape of a finding in a production file. The rule
// reads test files only, so this function stays clean.
func Describe(err error) bool { return strings.Contains(err.Error(), "run id") }

// Compare holds the equality shape in a production file. It stays
// clean as well, under every setting.
func Compare(err error) bool { return err.Error() == "seed: run id must be non-empty" }
