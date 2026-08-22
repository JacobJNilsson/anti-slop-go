package min3

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Two fields stay clean when the setting asks for three.
func TestTwoFields(t *testing.T) {
	got := build()
	require.Equal(t, "boot", got.Name)
	require.Equal(t, 3, got.Count)
}

// Three fields reach the setting, and the report sits at the first
// site of the run.
func TestThreeFields(t *testing.T) {
	got := build()
	require.Equal(t, "boot", got.Name) // want `assertions name 3 fields of got one at a time`
	require.Equal(t, 3, got.Count)
	require.Equal(t, "run", got.Label)
}
