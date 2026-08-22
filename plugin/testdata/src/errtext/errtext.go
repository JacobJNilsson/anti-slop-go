// Package errtext is the fixture package of the errsemantics-equality
// setting. Its test file compares the message of an error, which rule
// G13 reports only when the setting is true.
package errtext

import (
	"errors"
	"fmt"
)

// ErrEmpty is the sentinel of the package.
var ErrEmpty = errors.New("the name is empty")

// Name wraps the sentinel for an empty name.
func Name(name string) error {
	if name == "" {
		return fmt.Errorf("name: %w", ErrEmpty)
	}

	return nil
}
