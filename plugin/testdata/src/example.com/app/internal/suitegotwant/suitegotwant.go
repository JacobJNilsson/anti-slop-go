// Package suitegotwant is the fixture of rule G14 for the test-packages
// setting. The setting names the package, so the rule reads the helper
// below and reports the call inside its assertion.
package suitegotwant

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Item is the value the helper asserts on.
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
// value.
func AssertItem(t *testing.T, name string) {
	if diff := cmp.Diff(Item{Name: name, Count: 3}, Stored(t, name)); diff != "" { // want `calls a helper that takes a testing value`
		t.Errorf("Item mismatch (-want +got):\n%s", diff)
	}
}
