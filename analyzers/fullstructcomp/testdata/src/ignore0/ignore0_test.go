package ignore0

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// The assertions name every field, so the fix needs no ignore name and
// a setting of zero keeps the report.
func TestWholeValue(t *testing.T) {
	got := pair()
	require.Equal(t, "boot", got.Name) // want `assertions name 2 fields of got one at a time`
	require.Equal(t, 3, got.Count)
}

// One field stays outside the assertions, so the fix needs one ignore
// name and a setting of zero rejects the group.
func TestOneFieldOutside(t *testing.T) {
	got := trio()
	require.Equal(t, "boot", got.Name)
	require.Equal(t, 3, got.Count)
}
