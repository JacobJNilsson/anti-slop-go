package a

import (
	_ "reflect"
	"testing"
)

// A blank import uses no object of reflect, so a test file with one
// stays inside the DeepEqual exemption. Such an import reflects
// nothing.
func TestBlankImport(t *testing.T) {
	if testing.Short() {
		t.Skip("the fixture needs no work")
	}
}
