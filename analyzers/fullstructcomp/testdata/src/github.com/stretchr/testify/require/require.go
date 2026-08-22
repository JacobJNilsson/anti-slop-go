// Package require fakes github.com/stretchr/testify/require for the
// fixtures of rule G12. analysistest builds a fixture from the files
// under testdata and fetches no module, so the fixture owns the import
// path that the rule resolves. The signatures match the real package at
// the positions the rule reads: the assertion takes the test value
// first, then the expected value and the actual value.
package require

// TestingT is the interface the real package takes. *testing.T
// satisfies it.
type TestingT interface {
	Errorf(format string, args ...any)
	FailNow()
}

// Equal is the assertion at the centre of the rule.
func Equal(t TestingT, expected, actual any, msgAndArgs ...any) {}

// Equalf carries a format message.
func Equalf(t TestingT, expected, actual any, msg string, args ...any) {}

// EqualValues compares after a conversion.
func EqualValues(t TestingT, expected, actual any, msgAndArgs ...any) {}

// EqualValuesf carries a format message.
func EqualValuesf(t TestingT, expected, actual any, msg string, args ...any) {}

// Exactly compares the value and the type.
func Exactly(t TestingT, expected, actual any, msgAndArgs ...any) {}

// Exactlyf carries a format message.
func Exactlyf(t TestingT, expected, actual any, msg string, args ...any) {}

// True states another claim, so the rule counts no call of it.
func True(t TestingT, value bool, msgAndArgs ...any) {}

// Len states another claim.
func Len(t TestingT, object any, length int, msgAndArgs ...any) {}

// Contains states another claim.
func Contains(t TestingT, s, contains any, msgAndArgs ...any) {}

// NotEqual states another claim.
func NotEqual(t TestingT, expected, actual any, msgAndArgs ...any) {}

// Assertions is the receiver form of the package. A test writes
// r := require.New(t) and calls r.Equal after it.
type Assertions struct {
	t TestingT
}

// New returns the receiver form.
func New(t TestingT) *Assertions { return &Assertions{t: t} }

// Equal is the method form of the assertion.
func (a *Assertions) Equal(expected, actual any, msgAndArgs ...any) {}
