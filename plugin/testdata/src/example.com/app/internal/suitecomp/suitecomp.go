// Package suitecomp is the fixture of rule G12 for the test-packages
// setting. The setting names the package, so the rule reads the helper
// below and reports its group.
package suitecomp

import "testing"

// Item is the value the helper asserts on.
type Item struct {
	Name  string
	Count int
}

// AssertItem checks an item field after field.
func AssertItem(t *testing.T, got Item) {
	if got.Name != "boot" { // want `assertions name 2 fields of got one at a time`
		t.Errorf("Name = %q", got.Name)
	}
	if got.Count != 3 {
		t.Errorf("Count = %d", got.Count)
	}
}
