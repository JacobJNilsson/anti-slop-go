// Package suiteerr is the fixture of rule G13 for the test-packages
// setting. The setting names the package, so the rule reads the helper
// below.
package suiteerr

import (
	"strings"
	"testing"
)

// AssertFailure decides which error came back from the words of the
// message.
func AssertFailure(t *testing.T, err error) {
	if !strings.Contains(err.Error(), "run id") { // want `reads the text of an error`
		t.Errorf("err = %v", err)
	}
}
