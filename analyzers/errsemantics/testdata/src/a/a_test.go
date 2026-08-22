package a

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// aliased is an alias of the predeclared error type, so it is the same
// type. The rule reads through it.
type aliased = error

// The strings predicates decide the result of the test from the words
// of the message.
func TestStringsPredicates(t *testing.T) {
	err := Seed("")
	if strings.Contains(err.Error(), "run id") { // want `reads the text of an error`
		t.Error("Seed(\"\") reported the run id error")
	}
	if strings.HasPrefix(err.Error(), "seed") { // want `reads the text of an error`
		t.Error("Seed(\"\") reported the run id error")
	}
	if strings.HasSuffix(err.Error(), "non-empty") { // want `reads the text of an error`
		t.Error("Seed(\"\") reported the run id error")
	}
	if strings.EqualFold(err.Error(), "SEED: RUN ID MUST BE NON-EMPTY") { // want `reads the text of an error`
		t.Error("Seed(\"\") reported the run id error")
	}
}

// An alias of error is the same type, so the report stands.
func TestAliasOfError(t *testing.T) {
	var err aliased = Seed("")
	if strings.Contains(err.Error(), "run id") { // want `reads the text of an error`
		t.Error("Seed(\"\") reported the run id error")
	}
}

// A regular expression over the message is fragile in the same way.
func TestRegexpMatchers(t *testing.T) {
	err := Seed("")
	if ok, _ := regexp.MatchString("run id", err.Error()); ok { // want `reads the text of an error`
		t.Error("Seed(\"\") reported the run id error")
	}
	if regexp.MustCompile("run id").MatchString(err.Error()) { // want `reads the text of an error`
		t.Error("Seed(\"\") reported the run id error")
	}
	rx := regexp.MustCompile("run id")
	if rx.MatchString(err.Error()) { // want `reads the text of an error`
		t.Error("Seed(\"\") reported the run id error")
	}
}

// The text assertions of testify take the error itself, so no Error
// call appears. The rule reads the argument that carries the error.
func TestTestifyTextAsserts(t *testing.T) {
	err := Seed("")
	assert.ErrorContains(t, err, "run id")                  // want `reads the text of an error`
	assert.ErrorContainsf(t, err, "run id", "Seed(%q)", "") // want `reads the text of an error`
	assert.Regexp(t, "run id", err.Error())                 // want `reads the text of an error`
	assert.Regexpf(t, "run id", err, "Seed(%q)", "")        // want `reads the text of an error`
	require.ErrorContains(t, err, "run id")                 // want `reads the text of an error`
}

// The receiver form of testify resolves to the same package.
func TestTestifyReceiverForm(t *testing.T) {
	err := Seed("")
	a := assert.New(t)
	a.ErrorContains(err, "run id") // want `reads the text of an error`
	a.Regexp("run id", err)        // want `reads the text of an error`
}
