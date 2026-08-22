package a

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The equality forms stay clean until the equality setting turns them
// on. A package that tests its own message text writes them, and the
// guide accepts that. Package "equality" holds the same shapes under
// an analyzer that reads the setting, and its expectations sit there.
func TestEqualityFormsStayCleanByDefault(t *testing.T) {
	err := Seed("")
	if err.Error() == "seed: run id must be non-empty" {
		t.Error("Seed(\"\") reported the run id error")
	}
	if err.Error() != "seed: run id must be non-empty" {
		t.Error("Seed(\"\") reported another error")
	}
	assert.EqualError(t, err, "seed: run id must be non-empty")
	assert.EqualErrorf(t, err, "seed: run id must be non-empty", "Seed(%q)", "")
	require.EqualError(t, err, "seed: run id must be non-empty")
	assert.Equal(t, "seed: run id must be non-empty", err.Error())
	assert.Equalf(t, "seed: run id must be non-empty", err.Error(), "Seed(%q)", "")
	assert.New(t).EqualError(err, "seed: run id must be non-empty")
	assert.New(t).Equal("seed: run id must be non-empty", err.Error())
}
