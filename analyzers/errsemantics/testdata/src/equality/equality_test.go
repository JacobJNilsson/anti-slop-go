package equality

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The setting adds the equality forms. It replaces nothing, so the
// default forms report here as well.
func TestTheDefaultFormsStillReport(t *testing.T) {
	err := Seed("")
	if strings.Contains(err.Error(), "run id") { // want `reads the text of an error`
		t.Error("Seed(\"\") reported the run id error")
	}
	assert.ErrorContains(t, err, "run id") // want `reads the text of an error`
}

// A comparison of the message against a string is the equality form.
func TestComparisonOfTheMessage(t *testing.T) {
	err := Seed("")
	if err.Error() == "seed: run id must be non-empty" { // want `compares the text of an error`
		t.Error("Seed(\"\") reported the run id error")
	}
	if err.Error() != "seed: run id must be non-empty" { // want `compares the text of an error`
		t.Error("Seed(\"\") reported another error")
	}
	want := "seed: run id must be non-empty"
	if want == err.Error() { // want `compares the text of an error`
		t.Error("Seed(\"\") reported the run id error")
	}
}

// The equality asserts of testify are the same decision.
func TestTestifyEqualityAsserts(t *testing.T) {
	err := Seed("")
	assert.EqualError(t, err, "seed: run id must be non-empty")                     // want `compares the text of an error`
	assert.EqualErrorf(t, err, "seed: run id must be non-empty", "Seed(%q)", "")    // want `compares the text of an error`
	require.EqualError(t, err, "seed: run id must be non-empty")                    // want `compares the text of an error`
	assert.Equal(t, "seed: run id must be non-empty", err.Error())                  // want `compares the text of an error`
	assert.Equalf(t, "seed: run id must be non-empty", err.Error(), "Seed(%q)", "") // want `compares the text of an error`
	assert.New(t).EqualError(err, "seed: run id must be non-empty")                 // want `compares the text of an error`
	assert.New(t).Equal("seed: run id must be non-empty", err.Error())              // want `compares the text of an error`
}

// The variadic tail of a testify assertion is the failure message
// under this setting as well.
func TestTestifyFailureMessage(t *testing.T) {
	err := Seed("")
	assert.Equal(t, "a name", "another name", "Seed(\"\") error = %s", err.Error())
	assert.Equalf(t, "a name", "another name", "Seed(\"\") error = %s", err.Error())
	assert.New(t).Equal("a name", "another name", "Seed(\"\") error = %s", err.Error())
	assert.New(t).Equal("a name", "another name", err.Error())
	assert.Regexp(t, "run id", "a plain string", "Seed(\"\") error = %v", err)
}

// A binary expression that is no comparison holds no assertion. A
// comparison of two plain strings holds none either.
func TestBinaryExpressionsWithoutAMessage(t *testing.T) {
	msg := Seed("").Error() + "!"
	if msg == "" {
		t.Error("the message is empty")
	}
}

// The setting reads the text of an error and no error value. An
// assertion on the value itself stays clean, and errorlint and
// testifylint own that ground.
func TestValueAssertionsStayClean(t *testing.T) {
	err := Seed("")
	assert.Equal(t, ErrEmptyRunID, errors.Unwrap(err))
	if !errors.Is(err, ErrEmptyRunID) {
		t.Errorf("Seed(\"\") error = %v, want ErrEmptyRunID", err)
	}
}
