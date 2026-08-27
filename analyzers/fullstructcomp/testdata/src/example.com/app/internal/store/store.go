// Package store holds the same helper as the suite package beside it.
// No pattern of the setting names this package, so it stays production
// code and the rule reads no line of it.
package store

import "testing"

// Item is the value the helper reads.
type Item struct {
	Name  string
	Count int
}

// AssertItem checks an item field after field in production code.
func AssertItem(t *testing.T, got Item) {
	if got.Name != "boot" {
		t.Errorf("Name = %q", got.Name)
	}
	if got.Count != 3 {
		t.Errorf("Count = %d", got.Count)
	}
}
