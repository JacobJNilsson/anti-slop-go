package structmin

import "testing"

// Two fields stay clean when the setting asks for three.
func TestTwoFields(t *testing.T) {
	var got Item
	if got.Name != "boot" {
		t.Errorf("Name = %q", got.Name)
	}
	if got.Count != 3 {
		t.Errorf("Count = %d", got.Count)
	}
}

// Three fields reach the setting.
func TestThreeFields(t *testing.T) {
	var got Item
	if got.Name != "boot" { // want `assertions name 3 fields of got one at a time`
		t.Errorf("Name = %q", got.Name)
	}
	if got.Count != 3 {
		t.Errorf("Count = %d", got.Count)
	}
	if got.Label != "run" {
		t.Errorf("Label = %q", got.Label)
	}
}
