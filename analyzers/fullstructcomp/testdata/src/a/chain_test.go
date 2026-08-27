package a

import (
	"testing"

	"b"
	"github.com/stretchr/testify/require"
)

// A call breaks the chain, because a method result is no field. The
// value "got" is therefore no base of these two calls. The expected
// value is a base, and its two fields report.
func TestCallBreaksTheChain(t *testing.T) {
	got := build()
	want := build()
	require.Equal(t, want.Name, got.cell().Name) // want `assertions name 2 fields of want one at a time`
	require.Equal(t, want.Count, got.cell().Count)
}

// An index step stays inside one chain, and the whole path is the
// field. Both assertions sit under one field, so the message names that
// field and not the whole value.
func TestIndexStep(t *testing.T) {
	got := build()
	require.Equal(t, "a", got.Cells[0].Name) // want `assertions name 2 fields of got one at a time; compare got.Cells as a whole`
	require.Equal(t, 1, got.Cells[0].Count)
}

// A pointer step and parentheses stay inside one chain as well, and
// they name another field of the value than the index step does.
func TestPointerStep(t *testing.T) {
	got := build()
	require.Equal(t, "b", (*got.Cell).Name) // want `assertions name 2 fields of got one at a time; compare got.Cell as a whole`
	require.Equal(t, 2, (*got.Cell).Count)
}

// The root of the chain names a package, and a package is no variable,
// so these two calls name no base.
func TestPackageRoot(t *testing.T) {
	require.Equal(t, "boot", b.Global.Name)
	require.Equal(t, 3, b.Global.Count)
}

// A whole value is no field. The rule reads a compare of two values as
// the fix it asks for, so such a call counts for nothing.
func TestWholeValue(t *testing.T) {
	got := build()
	want := build()
	require.Equal(t, want, got)
	require.Equal(t, 3, got.Count)
}

// A table case is no produced value, so the rule skips a base that a
// range clause declares.
func TestRangeDeclaresTheBase(t *testing.T) {
	for range 1 {
		t.Log("a clause with no variable declares no base")
	}
	for _, tc := range []Item{build()} {
		require.Equal(t, "boot", tc.Name)
		require.Equal(t, 3, tc.Count)
	}
}

// A range clause that writes to a variable of an outer statement makes
// that variable a table case as well.
func TestRangeAssignsTheBase(t *testing.T) {
	var tc Item
	for _, tc = range []Item{build()} {
		require.Equal(t, "boot", tc.Name)
		require.Equal(t, 3, tc.Count)
	}
}
