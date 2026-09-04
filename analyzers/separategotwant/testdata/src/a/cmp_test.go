package a

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

// cmp.Diff takes no testing value, so a nested call there hides the
// same act as one inside a testify assertion.
func TestDiffInline(t *testing.T) {
	if diff := cmp.Diff(buildItem("boot"), storedRoutes(t, "boot")); diff != "" { // want `calls a helper that takes a testing value`
		t.Errorf("Item mismatch (-want +got):\n%s", diff)
	}
}

// The rule judges every argument of cmp.Diff, the first one included.
// No argument of this call carries the testing value of the assertion.
func TestDiffFirstArgument(t *testing.T) {
	got := buildItem("boot")
	if diff := cmp.Diff(storedRoutes(t, "boot"), got); diff != "" { // want `calls a helper that takes a testing value`
		t.Errorf("Item mismatch (-want +got):\n%s", diff)
	}
}

// The accepted form binds the value before the comparison.
func TestDiffBound(t *testing.T) {
	got := storedRoutes(t, "boot")
	if diff := cmp.Diff(buildItem("boot"), got); diff != "" {
		t.Errorf("Item mismatch (-want +got):\n%s", diff)
	}
}

// The rule reads Diff of go-cmp and no other function of the package.
func TestCmpEqual(t *testing.T) {
	if !cmp.Equal(buildItem("boot"), storedRoutes(t, "boot")) {
		t.Error("the items differ")
	}
}

// One nested call gets one report. Both readings of this line reach the
// same call: the argument of require.Empty holds it, and cmp.Diff is an
// assertion of its own.
func TestDiffInsideRequire(t *testing.T) {
	require.Empty(t, cmp.Diff(buildItem("boot"), storedRoutes(t, "boot"))) // want `calls a helper that takes a testing value`
}

// A builtin call names no function of a package, so the rule reads no
// argument of it.
func TestBuiltinCall(t *testing.T) {
	require.Equal(t, 3, len([]int{1, 2, 3}))
}
