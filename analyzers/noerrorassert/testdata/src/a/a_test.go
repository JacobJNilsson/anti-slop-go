package a

import "testing"

// A test file gets no exemption. Test code reads wrapped errors too,
// and errors.As works there without a change.
func TestAssertInTestFile(t *testing.T) {
	err := open()
	pe, ok := err.(*ParseError) // want `a wrapped error defeats this type assertion`
	if !ok || pe.Line != 7 {
		t.Fatal("want the parse error of open")
	}
}
