package a

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The identity assertions are the fix the rule names, so they stay
// clean.
func TestIdentityAssertions(t *testing.T) {
	err := Seed("")
	if !errors.Is(err, ErrEmptyRunID) {
		t.Errorf("Seed(\"\") error = %v, want ErrEmptyRunID", err)
	}
	var pe *ParseError
	if !errors.As(Parse(), &pe) {
		t.Errorf("Parse() error = %v, want a ParseError", Parse())
	}
	assert.ErrorIs(t, err, ErrEmptyRunID)
	assert.ErrorAs(t, Parse(), &pe)
	require.ErrorIs(t, err, ErrEmptyRunID)
	assert.New(t).ErrorIs(err, ErrEmptyRunID)
}

// Diagnostic output is no assertion. A message that a failure prints
// helps the reader, and it decides nothing.
func TestDiagnosticOutput(t *testing.T) {
	err := Seed("")
	if !errors.Is(err, ErrEmptyRunID) {
		t.Errorf("Seed(\"\") error = %s, want ErrEmptyRunID", err.Error())
		t.Logf("the message was %s", err.Error())
		t.Fatalf("Seed(\"\") error = %s", err.Error())
	}
	_ = fmt.Sprintf("the message was %s", err.Error())
	_ = err.Error()
	func(string) {}(err.Error())
}

// The rule reads the direct argument of a reported call. A message
// that flows through a variable is a known gap.
func TestMessageThroughAVariable(t *testing.T) {
	msg := Seed("").Error()
	if strings.Contains(msg, "run id") {
		t.Error("Seed(\"\") reported the run id error")
	}
}

// A conversion is no direct argument either. regexp.Match takes bytes,
// so an error message reaches it through a conversion, and the rule
// leaves it alone.
func TestConversionOfTheMessage(t *testing.T) {
	err := Seed("")
	if ok, _ := regexp.Match("run id", []byte(err.Error())); ok {
		t.Error("Seed(\"\") reported the run id error")
	}
}

// A plain string is no error. The rule reads types, and no history of
// the value.
func TestPlainString(t *testing.T) {
	msg := "run id must be non-empty"
	if strings.Contains(msg, "run id") {
		t.Error("the message holds the words")
	}
}

// failure declares Error() string and is no error value. Rule G10
// reads the same narrow test.
type failure struct{}

// Error returns the message of the failure.
func (failure) Error() string { return "failure of the run id" }

// A type with an Error method is another type, so the rule leaves it
// alone.
func TestNonErrorType(t *testing.T) {
	var f failure
	if strings.Contains(f.Error(), "run id") {
		t.Error("the failure holds the words")
	}
}

// Message returns a string and is no Error method.
func (failure) Message() string { return "failure of the run id" }

// message returns a string and names no method at all.
func message() string { return "the run id is empty" }

// The rule reads the Error method of an error value. Another method
// and a plain function both return a string that no error owns.
func TestAnotherSourceOfAString(t *testing.T) {
	var f failure
	if strings.Contains(f.Message(), "run id") {
		t.Error("the failure holds the words")
	}
	if strings.Contains(message(), "run id") {
		t.Error("the message holds the words")
	}
}

// matcher belongs to this package and carries the name of a method of
// regexp.Regexp. The rule reads the object, so this method is another
// one.
type matcher struct{}

// MatchString reports whether a string holds the run id.
func (matcher) MatchString(s string) bool { return false }

// A method of the project that carries the name of a regexp method is
// no regexp match.
func TestHomeGrownMatcher(t *testing.T) {
	if (matcher{}).MatchString(Seed("").Error()) {
		t.Error("Seed(\"\") reported the run id error")
	}
}

// The variadic tail of a testify assertion is the failure message. An
// error there is diagnostic output, and it decides nothing, so the
// rule reads the assert arguments only. The receiver form drops the
// testing value, so its assert arguments are the first two.
func TestTestifyFailureMessage(t *testing.T) {
	err := Seed("")
	assert.Regexp(t, "run id", "a plain string", "Seed(\"\") error = %v", err)
	assert.Regexpf(t, "run id", "a plain string", "Seed(\"\") error = %s", err.Error())
	assert.New(t).Regexp("run id", "a plain string", "Seed(\"\") error = %v", err)

	// The error sits in the first slot of the failure message. The
	// package form holds three assert arguments, and the receiver form
	// holds two, so both calls stay clean.
	assert.Regexp(t, "run id", "a plain string", err)
	assert.New(t).Regexp("run id", "a plain string", err)
}

// contains holds a function value, so the call resolves to a variable
// and not to the function of package strings. The rule reads the
// object, so this shape is a known gap.
var contains = strings.Contains

// A call through a function value names no package.
func TestFunctionValue(t *testing.T) {
	if contains(Seed("").Error(), "run id") {
		t.Error("Seed(\"\") reported the run id error")
	}
}

// A builtin call and a comparison of two plain strings hold no
// assertion about the identity of an error.
func TestNoComparisonOfAMessage(t *testing.T) {
	msg := Seed("").Error() + "!"
	if len(msg) == 0 {
		t.Error("the message is empty")
	}
	if msg == "" {
		t.Error("the message is empty")
	}
}

// A testify assertion that carries no error stays clean.
func TestTestifyWithoutAnError(t *testing.T) {
	assert.Equal(t, "run id", "run id")
	assert.Regexp(t, "run id", "the run id is empty")
}
