package a

import (
	"reflect"
	"testing"
)

// The exemption covers DeepEqual alone. One other use of reflect in the
// file takes the whole import back into the rule. The report sits at
// that use, which is the line the author must change.
func TestKind(t *testing.T) {
	if !reflect.DeepEqual(1, 1) {
		t.Fatal("DeepEqual reported two equal integers as different")
	}
	if reflect.TypeOf(1).Kind() != reflect.Int { // want `this test file uses reflect beyond DeepEqual`
		t.Fatal("the kind of an integer is not Int")
	}
	// The file holds a second use of reflect on a later line, and it
	// gets no report of its own. One file gives one report, and it
	// sits at the first use.
	if reflect.ValueOf(1).IsZero() {
		t.Fatal("the value 1 reads as the zero value")
	}
}
