// The external test package sees the exported names of the package
// under test, and the exported names of the test files of that
// package. The rule reads the file that declares the variable, so the
// two cases part here.
package a_test

import (
	"testing"
	"time"

	"a"
)

// TestExternalPatch rewires the package under test from outside it.
func TestExternalPatch(t *testing.T) {
	a.Now = time.Now // want `variable a.Now;`
	// A test file of package a declares this variable, so it is test
	// infrastructure and no production seam.
	a.TestSink = func(message string) {}
}
