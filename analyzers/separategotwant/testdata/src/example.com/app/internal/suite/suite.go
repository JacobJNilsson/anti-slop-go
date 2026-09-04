// Package suite holds the shared test suite of the fixture. No file of
// the package carries a name that ends in _test.go, and the package
// serves the tests of the project alone. The test-packages setting
// names it, so the rule reads the helper below and reports it.
package suite

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Item is the value the suite asserts on.
type Item struct {
	Name  string
	Count int
}

// Stored reads the value under test. It takes the testing value, so it
// stops the test from inside an expression.
func Stored(t *testing.T, name string) Item {
	t.Helper()

	return Item{Name: name, Count: 3}
}

// AssertItem asserts on the result of a call that takes the testing
// value. The reader of the assertion sees no value at all.
func AssertItem(t *testing.T, name string) {
	require.Equal(t, Item{Name: name, Count: 3}, Stored(t, name)) // want `calls a helper that takes a testing value`
}
