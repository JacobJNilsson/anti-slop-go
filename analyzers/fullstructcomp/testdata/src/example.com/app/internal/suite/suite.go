// Package suite holds the shared test suite of the fixture. No file of
// the package carries a name that ends in _test.go, and the package
// serves the tests of the project alone. The test-packages setting
// names it, so the rule reads the helper below and reports its group.
package suite

import "testing"

// Item is the value the suite asserts on.
type Item struct {
	Name  string
	Count int
}

// AssertItem checks an item field after field. The helper states one
// claim for each field it names, and no claim about the rest.
func AssertItem(t *testing.T, got Item) {
	if got.Name != "boot" { // want `assertions name 2 fields of got one at a time`
		t.Errorf("Name = %q", got.Name)
	}
	if got.Count != 3 {
		t.Errorf("Count = %d", got.Count)
	}
}
