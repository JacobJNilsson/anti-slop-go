package errtext

import "testing"

// The comparison reports only when the setting reaches the analyzer.
func TestName(t *testing.T) {
	if Name("").Error() != "name: the name is empty" { // want `compares the text of an error`
		t.Error("Name(\"\") reported another error")
	}
}
