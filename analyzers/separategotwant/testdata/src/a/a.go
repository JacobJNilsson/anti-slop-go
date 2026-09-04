// Package a holds the fixtures of rule G14. The test files of the
// package carry the assertions, and this file carries the helpers they
// call. The rule reads test files only, so no line of this file
// reports, and the assertion at the end of the file pins that.
package a

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Item is the value the fixtures assert on.
type Item struct {
	Name  string
	Count int
}

// storedRoutes is the helper the rule rejects inline. It takes the
// testing value, so it stops the test from inside an expression.
func storedRoutes(t *testing.T, name string) Item {
	t.Helper()
	if name == "" {
		t.Fatal("no name")
	}

	return Item{Name: name, Count: 3}
}

// countRows takes the interface form of the testing value. The rule
// reads testing.TB beside the three pointer types.
func countRows(tb testing.TB) int {
	tb.Helper()

	return 3
}

// benchRows takes the benchmark form of the testing value.
func benchRows(b *testing.B) int {
	b.Helper()

	return 3
}

// fuzzRows takes the fuzz form of the testing value.
func fuzzRows(f *testing.F) int {
	f.Helper()

	return 3
}

// subTest returns a testing value. A test writes it in the first
// argument of an assertion, where the rule reads nothing.
func subTest(t *testing.T) *testing.T {
	return t
}

// failMessage builds the failure message of an assertion. It takes the
// testing value, so it can stop the test from inside the message
// arguments of the assertion.
func failMessage(t *testing.T) string {
	t.Helper()

	return "the store holds another item"
}

// wrap takes the result of another call and no testing value. The rule
// reports the call that receives the testing value, and never the call
// around it.
func wrap(item Item) Item { return item }

// buildItem is a pure builder. It takes no testing value, so it cannot
// end the test, and it stays legal inside an assertion.
func buildItem(name string) Item { return Item{Name: name, Count: 3} }

// AssertItem holds the rejected form in production code. The rule reads
// test files only, so this line carries no want comment and must stay
// clean.
func AssertItem(t *testing.T, name string) {
	require.Equal(t, buildItem(name), storedRoutes(t, name))
}
