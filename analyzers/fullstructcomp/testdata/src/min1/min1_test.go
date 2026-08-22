package min1

import "testing"

// One field reaches a setting of one, and the message states one field.
func TestOneField(t *testing.T) {
	got := build()
	if got.Name != "boot" { // want `assertions name 1 field of got one at a time`
		t.Errorf("Name = %q", got.Name)
	}
}
