package ignorehigh

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// The fix needs six ignore names, which is one above the default. A
// high setting keeps the report.
func TestSixNames(t *testing.T) {
	got := build()
	require.Equal(t, "boot", got.Name) // want `assertions name 2 fields of got one at a time`
	require.Equal(t, 3, got.Count)
}
