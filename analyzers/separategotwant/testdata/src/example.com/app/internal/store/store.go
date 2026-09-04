// Package store holds the same helper as the suite package beside it.
// No pattern of the setting names this package, so it stays production
// code and the rule reads no line of it.
package store

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Item is the value the helper reads.
type Item struct {
	Name  string
	Count int
}

// Stored reads the value under test.
func Stored(t *testing.T, name string) Item {
	t.Helper()

	return Item{Name: name, Count: 3}
}

// AssertItem holds the rejected form in production code.
func AssertItem(t *testing.T, name string) {
	require.Equal(t, Item{Name: name, Count: 3}, Stored(t, name))
}
