package a

import (
	"testing"

	"other/cmp"
	"other/require"
)

// Both packages below carry the name of a library the rule reads, and
// another import path. The rule resolves the import path of the call,
// so no line of this file reports.
func TestLookalikeRequire(t *testing.T) {
	require.Equal(t, buildItem("boot"), storedRoutes(t, "boot"))
}

// The lookalike of go-cmp carries the name of the one function the rule
// reads.
func TestLookalikeDiff(t *testing.T) {
	if diff := cmp.Diff(buildItem("boot"), storedRoutes(t, "boot")); diff != "" {
		t.Errorf("Item mismatch (-want +got):\n%s", diff)
	}
}
