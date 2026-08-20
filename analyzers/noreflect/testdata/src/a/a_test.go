package a

import (
	"reflect"
	"testing"
)

// A test file that only calls reflect.DeepEqual is clean. Go gives a
// test no other way to compare two values of a composite type.
func TestEqualLists(t *testing.T) {
	if !reflect.DeepEqual([]string{"x"}, []string{"x"}) {
		t.Fatal("DeepEqual reported two equal slices as different")
	}
}
