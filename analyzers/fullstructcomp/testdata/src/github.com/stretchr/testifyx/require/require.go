// Package require sits at an import path that starts with the letters
// of the testify module and names another module. The rule reads the
// path segment by segment, so no call of this package counts.
package require

// TestingT mirrors the interface of the real package.
type TestingT interface {
	Errorf(format string, args ...any)
}

// Equal carries the name of the testify assertion.
func Equal(t TestingT, expected, actual any, msgAndArgs ...any) {}
