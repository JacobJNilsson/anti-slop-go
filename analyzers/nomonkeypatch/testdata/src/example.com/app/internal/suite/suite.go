// Package suite holds the shared test suite of the fixture. No file of
// the package carries a name that ends in _test.go, and the package
// serves the tests of the project alone. The test-packages setting
// names it, so the rule reads the assignments below as test code.
package suite

import "other"

// Sink collects the messages of a test. The suite package declares it,
// and the setting names that package, so the suite owns the variable.
var Sink = func(message string) {}

// Reset restores the sink of the suite. The target belongs to the test
// code, so this assignment stands.
func Reset() {
	Sink = func(message string) {}
}

// Patch rewires a package-level function variable of production code.
// The owner of that code cannot see the change, so the rule reports it.
func Patch() {
	other.Send = func(message string) error { return nil } // want `test assigns to the package-level variable other.Send`
}
