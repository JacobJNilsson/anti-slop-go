package gate

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Two assertions reach one leaf inside two large subtrees. One
// comparison of the whole response needs an ignore name for every other
// field of both subtrees, so the fix states far more than the test. The
// gate stops the report.
func TestTwoLargeSubtrees(t *testing.T) {
	got := response()
	require.Equal(t, 2, got.First.Amount)
	require.Equal(t, 3, got.Second.Amount)
}

// A small value stays inside the setting. The fix ignores the one field
// the assertions leave out, so the rule reports as before.
func TestSmallValue(t *testing.T) {
	got := order()
	require.Equal(t, "boot", got.Name) // want `^assertions name 2 fields of got one at a time; compare got as a whole with cmp.Diff against a want value; cmpopts.IgnoreFields skips a field the test cannot predict$`
	require.Equal(t, 3, got.Count)
}

// Every assertion sits under one field, so the comparison the message
// asks for sits there as well. The cost of that subtree is one name,
// although the whole response costs many.
func TestAnchorNamesTheSubtree(t *testing.T) {
	got := response()
	require.Equal(t, 1, got.Pagination.Page) // want `^assertions name 2 fields of got one at a time; compare got.Pagination as a whole with cmp.Diff against a want value; cmpopts.IgnoreFields skips a field the test cannot predict$`
	require.Equal(t, 20, got.Pagination.Size)
}

// Each site compares one field path against the same path of a value of
// the same type. Such a test holds a want value already, so the gate
// asks no cost of it. The other base arrives through a slice, an array
// and a map, and cmp compares the element of each one.
func TestRoundtripThroughContainers(t *testing.T) {
	got := response()
	list := []Response{response()}
	pair := [2]Response{}
	byKey := map[string]Response{}
	require.Equal(t, list[0].First.Amount, got.First.Amount) // want `assertions name 3 fields of got one at a time`
	require.Equal(t, pair[0].Second.Amount, got.Second.Amount)
	require.Equal(t, byKey["a"].Status, got.Status)
}

// A pointer to the type of the base carries the same want value.
func TestRoundtripThroughAPointer(t *testing.T) {
	got := response()
	want := responsePointer()
	require.Equal(t, want.First.Amount, got.First.Amount) // want `assertions name 2 fields of want one at a time` `assertions name 2 fields of got one at a time`
	require.Equal(t, want.Second.Amount, got.Second.Amount)
}

// The expected side names a value of another type. No want value of the
// type of the base stands beside it, so the fix writes one by hand and
// pays the cost of it. Both bases stay above the setting.
func TestOtherTypedPartner(t *testing.T) {
	got := response()
	other := summary()
	require.Equal(t, other.First.Amount, got.First.Amount)
	require.Equal(t, other.Second.Amount, got.Second.Amount)
}

// The two bases hold one type, and each site pairs two different field
// paths. A roundtrip needs both signals, so the gate reads the cost
// here and stops both reports.
func TestSameTypeDifferentPaths(t *testing.T) {
	got := response()
	want := response()
	require.Equal(t, want.First.Amount, got.Second.Amount)
	require.Equal(t, want.Second.Amount, got.First.Amount)
}

// The message of the first site names a second field of the actual
// value, so that site states two claims about it and counts for
// nothing there. The base of the expected value keeps one field of that
// site, and the site is no roundtrip for it.
func TestMessageNamesASecondField(t *testing.T) {
	got := order()
	want := order()
	require.Equalf(t, want.Name, got.Name, "the state is %s", got.State) // want `assertions name 2 fields of want one at a time`
	require.Equal(t, want.Count, got.Count)
}

// The test names a promoted field, and the rule resolves the field of
// the embedded structure that holds it. Both assertions therefore sit
// under one header, and the message names it.
func TestPromotedField(t *testing.T) {
	got := envelope()
	require.Equal(t, 200, got.Code) // want `assertions name 2 fields of got one at a time; compare got.Header as a whole`
	require.Equal(t, "ok", got.Header.Reason)
}

// One assertion names a field of the value itself, and the other names
// a promoted field of the first embedded structure. The two paths sit
// apart, so the comparison reads the whole envelope.
func TestEmbeddedSiblings(t *testing.T) {
	got := envelope()
	require.Equal(t, "text", got.Body) // want `assertions name 2 fields of got one at a time; compare got as a whole`
	require.Equal(t, "trace", got.Trace)
}

// One assertion names a field of the stamp, so the cut reads the stamp.
// cmp calls the Equal method there and reads no field under it, so the
// whole stamp costs one name. A walk into its seven fields would carry
// the group above the setting.
func TestAssertionThroughAnEqualType(t *testing.T) {
	got := ticket()
	require.Equal(t, "boot", got.Ref) // want `assertions name 2 fields of got one at a time; compare got as a whole`
	require.Equal(t, 2026, got.Stamp.Year)
}

// The chain names one type again and again. Every level of it holds one
// field that the assertions leave out, so every level costs one name.
// The group stays above the setting and the rule is silent.
func TestLongChain(t *testing.T) {
	got := ring()
	require.Equal(t, "boot", got.Name)
	require.Equal(t, "deep", got.Next.Next.Next.Next.Next.Next.Next.Next.Next.Next.Next.Next.Name)
}

// The type of the third field holds itself through a slice, and it
// holds no field. The walk that removes the containers of a type ends
// at its bound, and the field costs one name.
func TestTypeThatHoldsItself(t *testing.T) {
	got := loop()
	require.Equal(t, "boot", got.Name) // want `assertions name 2 fields of got one at a time; compare got as a whole`
	require.Equal(t, 3, got.Count)
}

// cmp calls the Equal method of the value at the anchor, so no option of
// cmp skips a field there. The assertions name every field, so one
// comparison states the same claims, and the message names no option.
func TestEqualAnchorNamesEveryField(t *testing.T) {
	got := point()
	require.Equal(t, 1, got.X) // want `^assertions name 2 fields of got one at a time; compare got as a whole with cmp.Diff against a want value$`
	require.Equal(t, 2, got.Y)
}

// One field of the value stays outside the assertions, and cmp reads no
// option for a type that answers a comparison itself. The rewrite
// cannot skip that field, so the rule states nothing.
func TestEqualAnchorLeavesAFieldOut(t *testing.T) {
	got := circle()
	require.Equal(t, 1, got.X)
	require.Equal(t, 2, got.Y)
}

// A want value of the same type stands beside the produced one, so both
// sites are a roundtrip. The type answers a comparison itself, and one
// field stays outside the assertions. No ignore name reaches a field of
// such a value, so the roundtrip carries no fix either.
func TestRoundtripOnAnEqualTypeLeavesAFieldOut(t *testing.T) {
	got := circle()
	want := circle()
	require.Equal(t, want.X, got.X)
	require.Equal(t, want.Y, got.Y)
}

// The same roundtrip names every field of the type. One comparison of
// the two values states both claims, and it needs no option.
func TestRoundtripOnAnEqualTypeNamesEveryField(t *testing.T) {
	got := point()
	want := point()
	require.Equal(t, want.X, got.X) // want `^assertions name 2 fields of want one at a time; compare want as a whole with cmp.Diff against a want value$` `^assertions name 2 fields of got one at a time; compare got as a whole with cmp.Diff against a want value$`
	require.Equal(t, want.Y, got.Y)
}

// The base holds an unexported field, and the anchor sits under it. The
// comparison the message asks for reads the subtree alone, so it needs
// no cmp.AllowUnexported and the message names none.
func TestAnchorHidesTheUnexportedField(t *testing.T) {
	got := record()
	require.Equal(t, 200, got.Detail.Code) // want `assertions name 2 fields of got one at a time; compare got.Detail as a whole with cmp.Diff against a want value; cmpopts.IgnoreFields skips a field the test cannot predict$`
	require.Equal(t, "ok", got.Detail.Reason)
}

// One site of the run is a roundtrip, and the setting asks for two. The
// gate therefore reads the cost of the whole response and stops the
// report.
func TestOneRoundtripSite(t *testing.T) {
	got := response()
	want := response()
	require.Equal(t, want.First.Amount, got.First.Amount)
	require.Equal(t, 3, got.Second.Amount)
}

// The expected side of the first site names two fields of its base, so
// that site compares two values already. It is no roundtrip for the
// actual value either, and one roundtrip site is below the setting.
func TestPartnerNamesTwoFields(t *testing.T) {
	got := response()
	want := response()
	require.Equalf(t, want.First.Amount, got.First.Amount, "second is %d", want.Second.Amount)
	require.Equal(t, want.Status, got.Status)
}

// One asserted path holds the other. The anchor is the parent of both,
// which is the base itself, and the cost of the whole response stops
// the report. An anchor at the shared path would read a cost of zero
// and report.
func TestPathHoldsAnotherPath(t *testing.T) {
	got := response()
	require.Equal(t, Pagination{}, got.Pagination)
	require.Equal(t, 1, got.Pagination.Page)
}
