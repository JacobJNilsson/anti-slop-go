package inlinemin3

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Two element comparisons stay below a min setting of 3.
func TestTwoElementsStayClean(t *testing.T) {
	want := items()
	got := items()
	require.Equal(t, want[0], got[0])
	require.Equal(t, want[1], got[1])
}

// Three element comparisons reach the min setting.
func TestThreeElementsReport(t *testing.T) {
	want := items()
	got := items()
	require.Equal(t, want[0], got[0]) // want `assertions compare 3 elements of want one at a time` `assertions compare 3 elements of got one at a time`
	require.Equal(t, want[1], got[1])
	require.Equal(t, want[2], got[2])
}
