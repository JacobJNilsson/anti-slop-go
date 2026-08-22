package a

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Two fields of one base, one field for each call. The report sits at
// the first of the two calls, and it names the whole fix. An Item holds
// a Stamp, whose Equal method stops the walk of cmp, so the message
// names no option and the expectation below anchors both ends.
func TestChecklist(t *testing.T) {
	got := build()
	require.Equal(t, "boot", got.Name) // want `^assertions name 2 fields of got one at a time; compare the whole value with cmp.Diff against a want value; cmpopts.IgnoreFields skips a field the test cannot predict$`
	require.Equal(t, 3, got.Count)
}

// One field is the whole claim of the test, and the rule stays quiet.
func TestOneField(t *testing.T) {
	got := build()
	require.Equal(t, "boot", got.Name)
}

// The rule counts distinct fields, so one field twice is one field.
func TestSameFieldTwice(t *testing.T) {
	got := build()
	require.Equal(t, "boot", got.Name)
	require.Equalf(t, "boot", got.Name, "the name is %q", "boot")
}

// Every member of the equality family counts.
func TestEqualFamily(t *testing.T) {
	got := build()
	require.EqualValues(t, "boot", got.Name) // want `assertions name 5 fields of got one at a time`
	require.EqualValuesf(t, 3, got.Count, "the count is %d", 3)
	require.Exactly(t, "run", got.Inner.Label)
	require.Exactlyf(t, Inner{Label: "run"}, got.Inner, "the inner value is %v", "run")
	require.Equal(t, Stamp{}, got.Stamp)
}

// A call outside the equality family states another claim, and the rule
// reads none of them.
func TestOtherCalls(t *testing.T) {
	got := build()
	require.True(t, got.Name == "boot")
	require.Len(t, got.Cells, 2)
	require.Contains(t, got.Name, "oo")
	require.NotEqual(t, 3, got.Count)
}

// The receiver form resolves through the type of the receiver, which
// the testify module declares.
func TestReceiverForm(t *testing.T) {
	r := require.New(t)
	got := build()
	r.Equal("boot", got.Name) // want `assertions name 2 fields of got one at a time`
	r.Equal(3, got.Count)
}

// A nested path is a field of its own, so "Inner" and "Inner.Label" are
// two fields of one base.
func TestNestedField(t *testing.T) {
	got := build()
	require.Equal(t, Inner{Label: "run"}, got.Inner) // want `assertions name 2 fields of got one at a time`
	require.Equal(t, "run", got.Inner.Label)
}

// A subtest closure belongs to the declaration around it, so the two
// calls count together.
func TestSubtest(t *testing.T) {
	got := build()
	t.Run("name", func(t *testing.T) {
		require.Equal(t, "boot", got.Name) // want `assertions name 2 fields of got one at a time`
	})
	t.Run("count", func(t *testing.T) {
		require.Equal(t, 3, got.Count)
	})
}

// The rule groups by the object and never by the name. Each closure
// declares a value of its own, so the two values stay apart.
func TestClosuresHoldTwoValues(t *testing.T) {
	t.Run("first", func(t *testing.T) {
		got := build()
		require.Equal(t, "boot", got.Name)
	})
	t.Run("second", func(t *testing.T) {
		got := build()
		require.Equal(t, 3, got.Count)
	})
}

// Two bases in one function are two groups, and each one gets a report
// of its own at its own first site.
func TestTwoBases(t *testing.T) {
	first := build()
	second := build()
	require.Equal(t, "boot", first.Name)  // want `assertions name 2 fields of first one at a time`
	require.Equal(t, "boot", second.Name) // want `assertions name 2 fields of second one at a time`
	require.Equal(t, 3, first.Count)
	require.Equal(t, 3, second.Count)
}

// A helper is a declaration of its own, and its base is a parameter.
// Each call names one field of two bases, so the two bases both report,
// and one cmp.Diff answers both.
func compare(t *testing.T, want, got Item) {
	require.Equal(t, want.Name, got.Name) // want `assertions name 2 fields of want one at a time` `assertions name 2 fields of got one at a time`
	require.Equal(t, want.Count, got.Count)
}

// TestHelper calls the helper, so the fixture holds no dead code.
func TestHelper(t *testing.T) {
	compare(t, build(), build())
}

// The two shapes count together on one base.
func TestMixedShapes(t *testing.T) {
	got := build()
	require.Equal(t, "boot", got.Name) // want `assertions name 2 fields of got one at a time`
	if got.Count != 3 {
		t.Errorf("Count = %d", got.Count)
	}
}

// The type of the base holds an unexported field, so cmp.Diff panics
// without cmp.AllowUnexported and the message names the field.
func TestUnexportedField(t *testing.T) {
	got := secret()
	require.Equal(t, "boot", got.Name) // want `assertions name 2 fields of got one at a time.*cmp.Diff panics on the unexported field Secret.hidden, so the comparison needs cmp.AllowUnexported$`
	require.Equal(t, 3, got.Count)
}

// The walk reads the whole graph of the type. It meets the unexported
// field through the map of the type, after the pointer, the slice, and
// the array of a type that holds none.
func TestUnexportedInTheGraph(t *testing.T) {
	got := nest()
	require.Equal(t, "boot", got.Equal) // want `assertions name 2 fields of got one at a time.*unexported field Leaf.hidden`
	require.Equal(t, "cell", got.Ptr.Name)
}

// The Equal method of the field takes a pointer, and the field holds a
// value. cmp reads the methods of the value receivers of a value, so it
// meets the unexported field under this one.
func TestPointerReceiverEqualOnAValue(t *testing.T) {
	got := sealedValue()
	require.Equal(t, "boot", got.Name) // want `assertions name 2 fields of got one at a time.*unexported field Sealed.hidden`
	require.Equal(t, 3, got.Count)
}

// The same method answers for a pointer field, because a pointer holds
// the methods of both receivers. cmp stops there, so the message names
// no option and the expectation anchors its end.
func TestPointerReceiverEqualOnAPointer(t *testing.T) {
	got := sealedPointer()
	require.Equal(t, "boot", got.Name) // want `cmpopts.IgnoreFields skips a field the test cannot predict$`
	require.Equal(t, 3, got.Count)
}

// The Equal method of the field takes a value and has a pointer
// receiver. The method set of a value holds the methods of its value
// receivers only, so cmp calls no method here.
func TestPointerReceiverEqualTakesAValue(t *testing.T) {
	got := looseValue()
	require.Equal(t, "boot", got.Name) // want `assertions name 2 fields of got one at a time.*unexported field Loose.hidden`
	require.Equal(t, 3, got.Count)
}

// The Equal method of the field takes another type, so cmp calls it for
// no comparison and meets the unexported field under it.
func TestUnusableEqualSignature(t *testing.T) {
	got := mismatchValue()
	require.Equal(t, "boot", got.Name) // want `assertions name 2 fields of got one at a time.*unexported field Mismatch.hidden`
	require.Equal(t, 3, got.Count)
}
