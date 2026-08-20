package codec_test

import (
	"reflect"
	"testing"

	"example.com/app/internal/codec"
)

// The external test package of an allowed package carries "_test" at
// the end of its path. The rule drops that suffix before it reads the
// allow patterns, so one entry covers the package and its external
// test package.
func TestFields(t *testing.T) {
	got := codec.Fields(struct{ A int }{})
	if reflect.TypeOf(got).Kind() != reflect.Slice {
		t.Fatalf("Fields returned %T, want a slice", got)
	}
}
