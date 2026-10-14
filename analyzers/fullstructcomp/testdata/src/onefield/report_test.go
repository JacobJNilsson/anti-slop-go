package onefield

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// The form of the issue. One field remains.
func TestPointer(t *testing.T) {
	got := Refuse()
	want := &Refusal{StatusCode: 400}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(Refusal{}, "Body")); diff != "" { // want `^cmp.Diff compares only the field StatusCode after cmpopts.IgnoreFields; assert that field with a plain assertion$`
		t.Error(diff)
	}
}

// The want value comes from a call.
func TestWantFromCall(t *testing.T) {
	got := Refuse()
	want := Refuse()
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(Refusal{}, "Body")); diff != "" { // want `^cmp.Diff compares only the field StatusCode after cmpopts.IgnoreFields; assert that field with a plain assertion$`
		t.Error(diff)
	}
}

// The ignore list drops every exported field, so the diff compares
// nothing.
func TestZeroRemaining(t *testing.T) {
	got := Refuse()
	want := &Refusal{}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(Refusal{}, "StatusCode", "Body")); diff != "" { // want `^cmp.Diff compares no field after cmpopts.IgnoreFields; remove the diff or assert the field the test needs with a plain assertion$`
		t.Error(diff)
	}
}

// A type with an Equal method still reports. The report asks for a
// plain assertion and reads no option of cmp.
func TestEqualMethod(t *testing.T) {
	got := Equaler{}
	want := Equaler{Code: 1}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(Equaler{}, "Note")); diff != "" { // want `^cmp.Diff compares only the field Code after cmpopts.IgnoreFields; assert that field with a plain assertion$`
		t.Error(diff)
	}
}

// The diff names the value of a call and the want value by a literal.
func TestInlineOperands(t *testing.T) {
	if diff := cmp.Diff(Refusal{StatusCode: 1}, *Refuse(), cmpopts.IgnoreFields(Refusal{}, "Body")); diff != "" { // want `^cmp.Diff compares only the field StatusCode after cmpopts.IgnoreFields; assert that field with a plain assertion$`
		t.Error(diff)
	}
}

// A want literal that sets one field with no ignore list is a claim
// about the whole value.
func TestCleanNoIgnoreList(t *testing.T) {
	got := Refuse()
	want := &Refusal{StatusCode: 400}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Error(diff)
	}
}

// A type with one exported field has no shorter form.
func TestCleanSingleField(t *testing.T) {
	got := Single{}
	want := Single{Name: "a"}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(Single{}, "note")); diff != "" {
		t.Error(diff)
	}
}

// The ignore name lists an unexported field, so the exported count
// stays at one.
func TestCleanHiddenField(t *testing.T) {
	got := Hidden{}
	want := Hidden{Code: 1}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(Hidden{}, "note")); diff != "" {
		t.Error(diff)
	}
}

// A nested name does not drop the parent field, so two fields remain.
func TestCleanNestedName(t *testing.T) {
	got := Nested{}
	want := Nested{Extra: "x"}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(Nested{}, "Body.Code")); diff != "" {
		t.Error(diff)
	}
}

// The nested name and one top-level name together leave the parent
// field, and that count is one. The report names it.
func TestNestedNameBesideTopLevelName(t *testing.T) {
	got := Nested{}
	want := Nested{}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(Nested{}, "Body.Code", "Extra")); diff != "" { // want `^cmp.Diff compares only the field Body after cmpopts.IgnoreFields; assert that field with a plain assertion$`
		t.Error(diff)
	}
}

// IgnoreUnexported changes no count of exported fields.
func TestCleanIgnoreUnexported(t *testing.T) {
	got := Refusal{}
	want := Refusal{StatusCode: 1}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreUnexported(Refusal{})); diff != "" {
		t.Error(diff)
	}
}

// An ignore list of another type leaves the count of this type alone.
func TestCleanOtherType(t *testing.T) {
	got := Refusal{}
	want := Refusal{StatusCode: 1}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(Triple{}, "A", "B")); diff != "" {
		t.Error(diff)
	}
}

// Two fields remain.
func TestCleanTwoRemain(t *testing.T) {
	got := Triple{}
	want := Triple{A: 1, B: 2}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(Triple{}, "C")); diff != "" {
		t.Error(diff)
	}
}

// A name that is not a constant cannot be counted, so the site stays
// silent.
func TestCleanDynamicName(t *testing.T) {
	name := "Body"
	got := Refusal{}
	want := Refusal{StatusCode: 1}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(Refusal{}, name)); diff != "" {
		t.Error(diff)
	}
}

// The operands are slices, which this report does not read.
func TestCleanSlices(t *testing.T) {
	got := []Refusal{}
	want := []Refusal{{StatusCode: 1}}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(Refusal{}, "Body")); diff != "" {
		t.Error(diff)
	}
}

// The operands hold different types.
func TestCleanDifferentTypes(t *testing.T) {
	if diff := cmp.Diff(Refusal{}, Inner{}, cmpopts.IgnoreFields(Refusal{}, "Body")); diff != "" {
		t.Error(diff)
	}
}

// A spread of options cannot be counted.
func TestCleanSpread(t *testing.T) {
	opts := []cmp.Option{cmpopts.IgnoreFields(Refusal{}, "Body")}
	got := Refusal{}
	want := Refusal{StatusCode: 1}
	if diff := cmp.Diff(want, got, opts...); diff != "" {
		t.Error(diff)
	}
}

// An ignore call without a type value reads nothing.
func TestCleanOwnIgnoreInOptionVariable(t *testing.T) {
	opt := cmpopts.IgnoreFields(Refusal{}, "Body")
	got := Refusal{}
	want := Refusal{StatusCode: 1}
	if diff := cmp.Diff(want, got, opt); diff != "" {
		t.Error(diff)
	}
}

// A call of cmp.Diff with two arguments carries no option.
func TestCleanNoOptions(t *testing.T) {
	_ = cmp.Diff(Refusal{}, Refusal{})
}

// A spread of names cannot be counted.
func TestCleanSpreadNames(t *testing.T) {
	names := []string{"Body"}
	got := Refusal{}
	want := Refusal{StatusCode: 1}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(Refusal{}, names...)); diff != "" {
		t.Error(diff)
	}
}

// An anonymous struct type works with IgnoreFields as well.
func TestAnonymousStruct(t *testing.T) {
	type pair = struct {
		Code int
		Text string
	}
	got := pair{}
	want := pair{Code: 1}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(pair{}, "Text")); diff != "" { // want `^cmp.Diff compares only the field Code after cmpopts.IgnoreFields; assert that field with a plain assertion$`
		t.Error(diff)
	}
}

// cmp applies the option to the type it names, so the twin type does
// not count.
func TestCleanTwinType(t *testing.T) {
	if diff := cmp.Diff(TwinA{}, TwinA{}, cmpopts.IgnoreFields(TwinB{}, "Y")); diff != "" {
		t.Error(diff)
	}
}

// A promoted name drops a field of the embedded value and not a field
// of the struct, so the count stays unknown.
func TestCleanPromotedName(t *testing.T) {
	if diff := cmp.Diff(Derived{}, Derived{}, cmpopts.IgnoreFields(Derived{}, "Code", "Text")); diff != "" {
		t.Error(diff)
	}
}

// A name that no field carries cannot be counted.
func TestCleanUnknownName(t *testing.T) {
	if diff := cmp.Diff(Refusal{}, Refusal{}, cmpopts.IgnoreFields(Refusal{}, "Body", "Nope")); diff != "" {
		t.Error(diff)
	}
}

// cmp panics on a pointer option type, so the option applies to no
// field and the count stays unknown.
func TestCleanPointerOptionType(t *testing.T) {
	if diff := cmp.Diff(Refusal{}, Refusal{}, cmpopts.IgnoreFields(&Refusal{}, "Body")); diff != "" {
		t.Error(diff)
	}
}
