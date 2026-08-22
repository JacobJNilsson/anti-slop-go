package a

import (
	"testing"

	"example.com/fake/testify/assert"
)

// The rule resolves the object of a call to the package that declares
// it. A library that carries the names of testify under another import
// path declares other objects, so no call here is a finding.
func TestFakeTestifyPackage(t *testing.T) {
	err := Seed("")
	assert.ErrorContains(t, err, "run id")
	assert.Regexp(t, "run id", err.Error())
	assert.EqualError(t, err, "seed: run id must be non-empty")
	assert.Equal(t, "seed: run id must be non-empty", err.Error())
}
