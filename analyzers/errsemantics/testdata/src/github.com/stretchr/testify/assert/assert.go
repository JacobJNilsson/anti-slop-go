// Package assert fakes github.com/stretchr/testify/assert.
// analysistest loads no module from the network, so the fixture owns
// this import path in its own GOPATH tree. The rule reads the import
// path of the object it resolves, so only the path and the signatures
// matter here. The real library writes interface{} where this file
// writes any, and both spell the same type.
package assert

// TestingT is the testing interface of the real library.
type TestingT interface {
	Errorf(format string, args ...any)
}

// ErrorContains asserts that the message of an error holds a substring.
func ErrorContains(t TestingT, theError error, contains string, msgAndArgs ...any) bool {
	return true
}

// ErrorContainsf is ErrorContains with a formatted failure message.
func ErrorContainsf(t TestingT, theError error, contains string, msg string, args ...any) bool {
	return true
}

// Regexp asserts that a value matches a regular expression.
func Regexp(t TestingT, rx any, str any, msgAndArgs ...any) bool { return true }

// Regexpf is Regexp with a formatted failure message.
func Regexpf(t TestingT, rx any, str any, msg string, args ...any) bool { return true }

// EqualError asserts that the message of an error equals a string.
func EqualError(t TestingT, theError error, errString string, msgAndArgs ...any) bool { return true }

// EqualErrorf is EqualError with a formatted failure message.
func EqualErrorf(t TestingT, theError error, errString string, msg string, args ...any) bool {
	return true
}

// Equal asserts that two values are equal.
func Equal(t TestingT, expected, actual any, msgAndArgs ...any) bool { return true }

// Equalf is Equal with a formatted failure message.
func Equalf(t TestingT, expected, actual any, msg string, args ...any) bool { return true }

// ErrorIs asserts that an error matches a target through errors.Is.
func ErrorIs(t TestingT, err, target error, msgAndArgs ...any) bool { return true }

// ErrorAs asserts that an error matches a target through errors.As.
func ErrorAs(t TestingT, err error, target any, msgAndArgs ...any) bool { return true }

// Assertions carries the testing value, so a test writes it once.
type Assertions struct {
	t TestingT
}

// New returns the receiver form of the assertions.
func New(t TestingT) *Assertions { return &Assertions{t: t} }

// ErrorContains is the receiver form of ErrorContains.
func (a *Assertions) ErrorContains(theError error, contains string, msgAndArgs ...any) bool {
	return true
}

// Regexp is the receiver form of Regexp.
func (a *Assertions) Regexp(rx any, str any, msgAndArgs ...any) bool { return true }

// EqualError is the receiver form of EqualError.
func (a *Assertions) EqualError(theError error, errString string, msgAndArgs ...any) bool {
	return true
}

// Equal is the receiver form of Equal.
func (a *Assertions) Equal(expected, actual any, msgAndArgs ...any) bool { return true }

// ErrorIs is the receiver form of ErrorIs.
func (a *Assertions) ErrorIs(err, target error, msgAndArgs ...any) bool { return true }
