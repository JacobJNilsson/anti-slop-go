// Package assert fakes github.com/stretchr/testify/assert for the
// fixtures of rule G14. The rule reads this package and the require
// package beside it. Both hold the same assertions, and they differ
// only in what they do after a failure.
package assert

// TestingT is the interface the real package takes.
type TestingT interface {
	Errorf(format string, args ...any)
}

// Equal is the assertion the fixtures write.
func Equal(t TestingT, expected, actual any, msgAndArgs ...any) bool { return true }
