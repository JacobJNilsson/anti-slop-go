package inline

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

// --- Rejected: inline composite literals of non-trivial struct collections ---

// A slice literal of a nested struct on the expected side.
func TestSliceLiteralOnExpectedSide(t *testing.T) {
	got := items()
	require.Equal(t, []Item{{Name: "a", Count: 1, Inner: Inner{Label: "x"}}, {Name: "b", Count: 2, Inner: Inner{Label: "y"}}}, got) // want `the assertion inlines a \[\]Item composite literal`
}

// A slice literal on the actual side fires the same way.
func TestSliceLiteralOnActualSide(t *testing.T) {
	got := items()
	require.Equal(t, got, []Item{{Name: "a", Count: 1, Inner: Inner{Label: "x"}}}) // want `the assertion inlines a \[\]Item composite literal`
}

// An array literal of a nested struct.
func TestArrayLiteral(t *testing.T) {
	got := arrayItems()
	require.Equal(t, [2]Item{{Name: "a", Count: 1, Inner: Inner{Label: "x"}}, {Name: "b", Count: 2, Inner: Inner{Label: "y"}}}, got) // want `the assertion inlines a \[2\]Item composite literal`
}

// A slice literal of a large scalar struct (>= 4 exported fields).
func TestWideLiteral(t *testing.T) {
	got := wideItems()
	require.Equal(t, []Wide{{Name: "a", Count: 1, State: "new", Kind: "order", Priority: 1}}, got) // want `the assertion inlines a \[\]Wide composite literal`
}

// The format variant of the equality family fires the same way.
func TestEqualf(t *testing.T) {
	got := items()
	require.Equalf(t, []Item{{Name: "a", Count: 1, Inner: Inner{Label: "x"}}}, got, "items = %v", got) // want `the assertion inlines a \[\]Item composite literal`
}

// The receiver form fires the same way.
func TestReceiverForm(t *testing.T) {
	r := require.New(t)
	got := items()
	r.Equal([]Item{{Name: "a", Count: 1, Inner: Inner{Label: "x"}}, {Name: "b", Count: 2, Inner: Inner{Label: "y"}}}, got) // want `the assertion inlines a \[\]Item composite literal`
}

// A collection whose element type holds an unexported field triggers
// the AllowUnexported note in the message.
func TestUnexportedField(t *testing.T) {
	got := secretItems()
	require.Equal(t, []WithSecret{{Label: "a", Detail: Secret{Name: "x", Count: 1}}}, got) // want `the assertion inlines a \[\]WithSecret composite literal.*cmp.Diff panics on the unexported field Secret.hidden, so the comparison needs cmp.AllowUnexported`
}

// --- Accepted: patterns the inline path must not report ---

// A trivial struct (2 scalar fields, no nesting) stays clean.
func TestTrivialFlat(t *testing.T) {
	got := []Flat{{Name: "a", Count: 1}}
	require.Equal(t, []Flat{{Name: "a", Count: 1}}, got)
}

// A slice of a non-struct type stays clean.
func TestStringSlice(t *testing.T) {
	got := []string{"a", "b"}
	require.Equal(t, []string{"a", "b"}, got)
}

// An empty literal carries nothing to diff.
func TestEmptyLiteral(t *testing.T) {
	got := items()
	require.Equal(t, []Item{}, got)
}

// A named variable is no inline literal.
func TestNamedVariable(t *testing.T) {
	want := items()
	got := items()
	require.Equal(t, want, got)
}

// A single struct (not a collection) stays clean.
func TestSingleStruct(t *testing.T) {
	got := items()[0]
	require.Equal(t, Item{Name: "a", Count: 1, Inner: Inner{Label: "x"}}, got)
}

// A time.Time field has an Equal method and is no nested struct.
func TestEqualMethodFieldIsNotNested(t *testing.T) {
	got := []Stamped{{Name: "a", Count: 1}}
	require.Equal(t, []Stamped{{Name: "a", Count: 1}}, got)
}

// Four exported fields make a struct non-trivial, even when one is a
// time.Time.
func TestEqualMethodFieldCountsForSize(t *testing.T) {
	got := []StampedWide{{Name: "a"}}
	require.Equal(t, []StampedWide{{Name: "a"}}, got) // want `the assertion inlines a \[\]StampedWide composite literal`
}

// A struct in an unexported field does not make a struct non-trivial.
func TestUnexportedNestedStructIsIgnored(t *testing.T) {
	got := []Hidden{{Name: "a"}}
	require.Equal(t, []Hidden{{Name: "a"}}, got)
}

// A literal on each side gives one report, at the first literal.
func TestTwoLiteralsGiveOneReport(t *testing.T) {
	require.Equal(t, []Item{{Name: "a", Count: 1}}, []Item{{Name: "a", Count: 1}}) // want `the assertion inlines a \[\]Item composite literal`
}

// Parentheses around the literal do not hide it.
func TestParenthesizedLiteral(t *testing.T) {
	got := items()
	require.Equal(t, ([]Item{{Name: "a", Count: 1, Inner: Inner{Label: "x"}}}), got) // want `the assertion inlines a \[\]Item composite literal`
}

// A named slice type of non-trivial structs is a collection as well.
func TestNamedSliceType(t *testing.T) {
	got := Items(items())
	require.Equal(t, Items{{Name: "a", Count: 1, Inner: Inner{Label: "x"}}}, got) // want `the assertion inlines a Items composite literal`
}

// The inline path reads equality calls only. An if statement that
// holds a literal stays clean.
func TestIfStatementWithLiteral(t *testing.T) {
	got := items()
	if !reflect.DeepEqual([]Item{{Name: "a", Count: 1, Inner: Inner{Label: "x"}}}, got) {
		t.Fatal("items differ")
	}
}
