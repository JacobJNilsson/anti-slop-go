package a

import "testing"

// A test file gets no exemption. A test that reads the dynamic type of
// an any value builds the same table of shapes that production code
// builds, and the fix is the same domain value.
func TestSwitchInTestFile(t *testing.T) {
	var v any = 1
	switch value := v.(type) { // want `this type switch reads the dynamic type of an any value`
	case int:
		if value != 1 {
			t.Error("want the value the test wrote")
		}
	}
}
