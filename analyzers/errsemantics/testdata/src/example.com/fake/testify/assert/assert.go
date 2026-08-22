// Package assert carries the names of testify under another import
// path. The rule resolves the object of a call to the package that
// declares it, so no call of this package is a finding. A rule that
// read the name of the package, or the name of the function, would
// report every one of them.
package assert

// TestingT is the testing interface of the fake library.
type TestingT interface {
	Errorf(format string, args ...any)
}

// ErrorContains carries the name of the testify assertion.
func ErrorContains(t TestingT, theError error, contains string, msgAndArgs ...any) bool {
	return true
}

// Regexp carries the name of the testify assertion.
func Regexp(t TestingT, rx any, str any, msgAndArgs ...any) bool { return true }

// EqualError carries the name of the testify assertion.
func EqualError(t TestingT, theError error, errString string, msgAndArgs ...any) bool { return true }

// Equal carries the name of the testify assertion.
func Equal(t TestingT, expected, actual any, msgAndArgs ...any) bool { return true }
