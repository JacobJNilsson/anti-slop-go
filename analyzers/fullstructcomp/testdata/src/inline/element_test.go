package inline

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// --- Rejected: element-by-element assertions on collection variables ---

// Two indexed accesses on the same collection base, roundtrip form.
// Both want and got report.
func TestElementRoundtrip(t *testing.T) {
	want := items()
	got := items()
	require.Equal(t, want[0], got[0]) // want `assertions compare 2 elements of want one at a time` `assertions compare 2 elements of got one at a time`
	require.Equal(t, want[1], got[1])
}

// A plain variable against an indexed element is a spot check. The
// site has one indexed operand, so it does not count.
func TestElementPlainVariableCalls(t *testing.T) {
	table := items()
	line := Item{Name: "a"}
	require.Equal(t, line, table[0])
	require.Equal(t, line, table[1])
}

// The same spot checks in the if form stay clean as well.
func TestElementPlainVariableIfs(t *testing.T) {
	table := items()
	line := Item{Name: "a"}
	for i := range table {
		if line != table[0] {
			t.Fatal("first line differs")
		} else if line != table[i] {
			t.Fatal("line differs")
		}
	}
}

// The receiver form fires the same way.
func TestElementReceiverForm(t *testing.T) {
	r := require.New(t)
	want := items()
	got := items()
	r.Equal(want[0], got[0]) // want `assertions compare 2 elements of want one at a time` `assertions compare 2 elements of got one at a time`
	r.Equal(want[1], got[1])
}

// The stdlib if-form fires the same way.
func TestElementStdlibForm(t *testing.T) {
	want := items()
	got := items()
	if want[0] != got[0] { // want `assertions compare 2 elements of want one at a time` `assertions compare 2 elements of got one at a time`
		t.Errorf("element 0 differs")
	}
	if want[1] != got[1] {
		t.Errorf("element 1 differs")
	}
}

// A collection whose element type holds an unexported field triggers
// the AllowUnexported note.
func TestElementUnexportedField(t *testing.T) {
	want := secretItems()
	got := secretItems()
	require.Equal(t, want[0], got[0]) // want `assertions compare 2 elements of want one at a time.*cmp.AllowUnexported` `assertions compare 2 elements of got one at a time.*cmp.AllowUnexported`
	require.Equal(t, want[1], got[1])
}

// --- Accepted: patterns the element path must not report ---

// A trivial struct (2 scalar fields) stays clean.
func TestElementTrivialFlat(t *testing.T) {
	want := []Flat{{Name: "a", Count: 1}, {Name: "b", Count: 2}}
	got := []Flat{{Name: "a", Count: 1}, {Name: "b", Count: 2}}
	require.Equal(t, want[0], got[0])
	require.Equal(t, want[1], got[1])
}

// A range loop variable is a table case, not a produced value.
func TestElementRangeVariable(t *testing.T) {
	for _, tc := range [][]Item{items(), items()} {
		require.Equal(t, Item{}, tc[0])
		require.Equal(t, Item{}, tc[1])
	}
}

// One call that compares two elements of one collection is one call.
// The count stays below the min setting of 2.
func TestElementOneCallTwoElements(t *testing.T) {
	got := items()
	require.Equal(t, got[0], got[1])
}

// A selector after the index goes to the field-by-field path. The
// element path does not read it.
func TestElementFieldChainIsFieldPath(t *testing.T) {
	got := items()
	require.Equal(t, "a", got[0].Name) // want `assertions name 2 fields of got one at a time`
	require.Equal(t, 1, got[0].Count)
}

// Array elements compare in the if form, and the sites reach min 2.
func TestElementArrayIfForm(t *testing.T) {
	a := arrayItems()
	b := arrayItems()
	if a[0] != b[0] { // want `assertions compare 2 elements of a one at a time` `assertions compare 2 elements of b one at a time`
		t.Fatal("element 0 differs")
	}
	if a[1] != b[1] {
		t.Fatal("element 1 differs")
	}
}

// A loop compares every element with a variable index. Two loops over
// one pair stay clean.
func TestElementLoopsStayClean(t *testing.T) {
	want := items()
	got := items()
	for i := range want {
		require.Equal(t, want[i], got[i])
	}
	for i := range want {
		require.Equal(t, want[i], got[i])
	}
}

// A spot check of the last element has a computed index and stays
// clean.
func TestElementLastElementStaysClean(t *testing.T) {
	want := items()
	got := items()
	require.Equal(t, want[len(want)-1], got[len(got)-1])
	require.Equal(t, want[len(want)-2], got[len(got)-2])
}
