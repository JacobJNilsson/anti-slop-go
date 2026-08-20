package a

import (
	r "reflect"
	"testing"
)

// A renamed import loses the exemption on the same terms as a plain
// one: the file uses an object of reflect that is not DeepEqual. The
// report follows the use and not the name of the import.
func TestRenamedValue(t *testing.T) {
	if r.ValueOf(1).Kind() != r.Int { // want `this test file uses reflect beyond DeepEqual`
		t.Fatal("the kind of an integer is not Int")
	}
}
