package a

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The rejected form of the specification. The assertion argument calls
// a helper that takes the testing value, so the helper can end the test
// from inside the check.
func TestInlineHelper(t *testing.T) {
	require.Equal(t, buildItem("boot"), storedRoutes(t, "boot")) // want `^the assertion argument calls a helper that takes a testing value; bind got and want first, then assert$`
}

// The accepted form binds the value first. The failure of the helper
// then points at the line that produced the value, and the assertion
// reads two variables.
func TestBoundForm(t *testing.T) {
	got := storedRoutes(t, "boot")
	require.Equal(t, buildItem("boot"), got)
}

// A pure builder takes no testing value. It cannot end the test, so it
// stays legal inside the assertion.
func TestPureBuilder(t *testing.T) {
	require.Equal(t, buildItem("boot"), buildItem("boot"))
}

// The interface form of the testing value carries the same control
// flow. The rule reads the type of the argument, so a testing.TB value
// reports beside the three pointer types.
func assertRows(tb testing.TB) {
	require.Equal(tb, 3, countRows(tb)) // want `calls a helper that takes a testing value`
}

// TestInterfaceHelper calls the helper above, so the fixture builds
// with no unused declaration.
func TestInterfaceHelper(t *testing.T) {
	assertRows(t)
}

// The assert package holds the same assertions as require, and the rule
// reads both.
func TestAssertVariant(t *testing.T) {
	assert.Equal(t, buildItem("boot"), storedRoutes(t, "boot")) // want `calls a helper that takes a testing value`
}

// The nested call sits inside a slice literal, one step under the
// argument. The rule reads the whole argument expression.
func TestDeeperExpression(t *testing.T) {
	require.Equal(t, []Item{buildItem("boot")}, []Item{storedRoutes(t, "boot")}) // want `calls a helper that takes a testing value`
}

// The receiver form holds the testing value in the Assertions value, so
// every argument of the call carries a value.
func TestReceiverForm(t *testing.T) {
	r := require.New(t)
	r.Equal(buildItem("boot"), storedRoutes(t, "boot")) // want `calls a helper that takes a testing value`
}

// The testing value that the assertion takes first counts for nothing.
// A call in that position builds the testing value and asserts nothing.
func TestFirstArgument(t *testing.T) {
	require.Equal(subTest(t), buildItem("boot"), buildItem("boot"))
}

// The report sits at the call that receives the testing value, and
// never at the call around it. The outer call takes an Item.
func TestNestedCall(t *testing.T) {
	require.Equal(t, buildItem("boot"), wrap(storedRoutes(t, "boot"))) // want `calls a helper that takes a testing value`
}

// The variadic message arguments count as value arguments. A message
// helper that takes the testing value stops the test from inside the
// expression, so the rule reads that position beside the two values.
func TestVariadicMessage(t *testing.T) {
	got := storedRoutes(t, "boot")
	require.Equal(t, buildItem("boot"), got, failMessage(t)) // want `calls a helper that takes a testing value`
}

// A call inside a function literal runs after the assertion built its
// arguments. The closure holds the statement, so the fix of the rule is
// unavailable there and the rule reads no line of such a body.
func TestFunctionLiteral(t *testing.T) {
	require.Equal(t, 3, func() int { return countRows(t) }())
}

// A type conversion names no callable, so it earns no report of its
// own. The line holds one report, at the helper that takes the
// converted testing value.
func TestTypeConversion(t *testing.T) {
	require.Equal(t, 3, countRows(testing.TB(t))) // want `calls a helper that takes a testing value`
}

// A method value hides the callee, and the rule resolves no function
// for the later call. Such a call escapes the rule, and review must
// catch it.
func TestMethodValueEscape(t *testing.T) {
	f := require.Equal
	f(t, buildItem("boot"), storedRoutes(t, "boot"))
}

// A benchmark carries the same control flow through *testing.B.
func BenchmarkRows(b *testing.B) {
	require.Equal(b, 3, benchRows(b)) // want `calls a helper that takes a testing value`
}

// A fuzz target carries it through *testing.F.
func FuzzRows(f *testing.F) {
	require.Equal(f, 3, fuzzRows(f)) // want `calls a helper that takes a testing value`
}
