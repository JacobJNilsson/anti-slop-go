// Package suite holds the shared test suite of the fixture. No file of
// the package carries a name that ends in _test.go, and the package
// serves the tests of the project alone. The test-packages setting
// names it, so a call that stops the process needs no justification
// here. No line of this file carries a want comment.
package suite

import "os"

// Fixture holds the state that a test of the project reads.
type Fixture struct{ Name string }

// Setup builds the fixture. A bad name is a fault of the test, and the
// test binary stops.
func Setup(name string) Fixture {
	if name == "" {
		panic("the suite needs a name")
	}

	return Fixture{Name: name}
}

// Teardown ends the test binary after the suite ran.
func Teardown(code int) {
	os.Exit(code)
}
