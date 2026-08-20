package a

import (
	r "reflect"
	"testing"
)

// The exemption follows the object and not the name, so a renamed
// import of reflect keeps it.
func TestRenamedEqual(t *testing.T) {
	if !r.DeepEqual("x", "x") {
		t.Fatal("DeepEqual reported two equal strings as different")
	}
}
