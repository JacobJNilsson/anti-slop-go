// Package suite holds the shared test suite of the fixture. No file of
// the package carries a name that ends in _test.go, and the package
// serves the tests of the project alone. The test-packages setting
// names it, so the rule reads the helper below.
package suite

import (
	"strings"
	"testing"
)

// AssertFailure reads the words of a message to decide which error
// the call returned. A change to that message breaks the helper.
func AssertFailure(t *testing.T, err error) {
	if !strings.Contains(err.Error(), "run id") { // want `reads the text of an error`
		t.Errorf("err = %v", err)
	}
}
