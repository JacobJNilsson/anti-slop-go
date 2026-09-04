// Package require fakes github.com/stretchr/testify/require for the
// fixtures of rule G14. analysistest builds a fixture from the files
// under testdata and fetches no module, so the fixture owns the import
// path that the rule resolves. The signatures match the real package at
// the positions the rule reads: the assertion takes the testing value
// first, then the values it asserts on.
package require

// TestingT is the interface the real package takes. *testing.T,
// *testing.B, and *testing.F all satisfy it.
type TestingT interface {
	Errorf(format string, args ...any)
	FailNow()
}

// Equal is the assertion at the centre of the rule.
func Equal(t TestingT, expected, actual any, msgAndArgs ...any) {}

// NoError asserts that a call returned no error.
func NoError(t TestingT, err error, msgAndArgs ...any) {}

// Empty asserts that a value is the zero value of its type. A test
// writes it around cmp.Diff.
func Empty(t TestingT, object any, msgAndArgs ...any) {}

// Assertions is the receiver form of the package. A test writes
// r := require.New(t) and calls r.Equal after it.
type Assertions struct {
	t TestingT
}

// New returns the receiver form. It is a function of the package, so
// the rule reads its arguments after the testing value.
func New(t TestingT) *Assertions { return &Assertions{t: t} }

// Equal is the method form of the assertion. The Assertions value holds
// the testing value, so every argument here carries a value.
func (a *Assertions) Equal(expected, actual any, msgAndArgs ...any) {}
