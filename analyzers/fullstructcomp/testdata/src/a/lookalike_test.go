package a

import (
	"testing"

	testifyx "github.com/stretchr/testifyx/require"
	"other/require"
)

// The name of the package is no evidence. The rule resolves the import
// path of the call, so these two calls count for nothing.
func TestLookalikePackage(t *testing.T) {
	got := build()
	require.Equal(t, "boot", got.Name)
	require.Equal(t, 3, got.Count)
}

// A module whose path starts with the letters of the testify module is
// another module. The rule reads the path segment by segment, and the
// name of the import answers nothing.
func TestLookalikeModule(t *testing.T) {
	got := build()
	testifyx.Equal(t, "boot", got.Name)
	testifyx.Equal(t, 3, got.Count)
}
