package a

import (
	"testing"

	"example.com/fake/strings"
)

// The rule resolves the object of a call to the package that declares
// it. A local package named strings declares another Contains, so this
// call is no finding.
func TestFakeStringsPackage(t *testing.T) {
	err := Seed("")
	if strings.Contains(err.Error(), "run id") {
		t.Error("Seed(\"\") reported the run id error")
	}
}
