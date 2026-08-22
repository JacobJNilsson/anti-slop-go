// Package equality holds the equality forms of rule G13. The analyzer
// instance of the equality setting reads this package, and the default
// instance reads package "a", which holds the same shapes with no want
// comment.
package equality

import (
	"errors"
	"fmt"
)

// ErrEmptyRunID is the sentinel that a test asserts with errors.Is.
var ErrEmptyRunID = errors.New("run id must be non-empty")

// Seed wraps the sentinel for an empty run id.
func Seed(id string) error {
	if id == "" {
		return fmt.Errorf("seed: %w", ErrEmptyRunID)
	}

	return nil
}
