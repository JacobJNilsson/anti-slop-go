// Package require fakes github.com/stretchr/testify/require.
// The real require package holds the same function names as assert and
// stops the test on a failure. The rule reads the import path, and the
// prefix of both packages is the same, so this file proves that the
// second package of the library resolves as well.
package require

// TestingT is the testing interface of the real library.
type TestingT interface {
	Errorf(format string, args ...any)
	FailNow()
}

// ErrorContains asserts that the message of an error holds a substring.
func ErrorContains(t TestingT, theError error, contains string, msgAndArgs ...any) {}

// EqualError asserts that the message of an error equals a string.
func EqualError(t TestingT, theError error, errString string, msgAndArgs ...any) {}

// ErrorIs asserts that an error matches a target through errors.Is.
func ErrorIs(t TestingT, err, target error, msgAndArgs ...any) {}
