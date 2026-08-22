// The external test package holds test code of the same package. The
// rule reads the name of the file and never the name of the package, so
// the two kinds of test file follow one rule.
package a_test

import (
	"testing"

	"a"
	"github.com/stretchr/testify/require"
)

// The rule reads this file, because its name ends in "_test.go".
func TestExternalPackage(t *testing.T) {
	got := a.Item{Name: "boot", Count: 3}
	require.Equal(t, "boot", got.Name) // want `assertions name 2 fields of got one at a time`
	require.Equal(t, 3, got.Count)
}
