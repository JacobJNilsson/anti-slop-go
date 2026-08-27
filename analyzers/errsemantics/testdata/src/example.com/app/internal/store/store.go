// Package store holds the same helper as the suite package beside it.
// No pattern of the setting names this package, so it stays production
// code and the rule reads no line of it.
package store

import (
	"strings"
	"testing"
)

// AssertFailure reads the text of an error in production code.
func AssertFailure(t *testing.T, err error) {
	if !strings.Contains(err.Error(), "run id") {
		t.Errorf("err = %v", err)
	}
}
