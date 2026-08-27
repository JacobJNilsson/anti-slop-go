package structignore

import "testing"

// Two fields of one value, and a fix that needs six ignore names. The
// high setting keeps the report.
func TestTwoFields(t *testing.T) {
	var got Item
	if got.Name != "boot" { // want `assertions name 2 fields of got one at a time`
		t.Errorf("Name = %q", got.Name)
	}
	if got.Count != 3 {
		t.Errorf("Count = %d", got.Count)
	}
}
