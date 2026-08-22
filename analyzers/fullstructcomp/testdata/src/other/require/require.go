// Package require carries the name of the testify package and another
// import path. The rule resolves the import path of the call, so no
// call of this package counts.
package require

// TestingT mirrors the interface of the real package.
type TestingT interface {
	Errorf(format string, args ...any)
}

// Equal carries the name of the testify assertion.
func Equal(t TestingT, expected, actual any, msgAndArgs ...any) {}
