// Package a holds the fixtures for the noerrorassert analyzer. The
// code is deliberately bad: it exists to trigger diagnostics.
package a

import (
	"errors"
	"fmt"
	"io/fs"
)

// ParseError is a concrete error type. Most fixtures name it as the
// target of the assertion.
type ParseError struct{ Line int }

func (e *ParseError) Error() string { return "parse error" }

// Code makes *ParseError satisfy Coded, so the fixtures that read a
// Coded value can name it.
func (e *ParseError) Code() int { return 2 }

// Coded is a user interface that embeds error. It is another type, so
// an assertion on a Coded value stays clean.
type Coded interface {
	error
	Code() int
}

// Wide has the method set of error under another name. It is another
// type too, so an assertion on a Wide value stays clean.
type Wide interface{ Error() string }

// ErrAlias names the predeclared error type, so it is the same type.
type ErrAlias = error

// ErrMissing is the sentinel that the errors.Is fixture compares.
var ErrMissing = errors.New("missing")

// open returns an error value, so an assertion on the result of the
// call has the static type error.
func open() error { return fmt.Errorf("open: %w", &ParseError{Line: 7}) }

func SingleResult(err error) int {
	return err.(*ParseError).Line // want `a wrapped error defeats this type assertion; use errors.As with a pointer to a target variable, or errors.Is for a sentinel`
}

func CommaOk(err error) int {
	pe, ok := err.(*ParseError) // want `a wrapped error defeats this type assertion`
	if !ok {
		return 0
	}
	return pe.Line
}

// Justified carries the SAFETY comment that rule G01 asks for. Rule
// G10 still reports: errors.As is the fix, and no comment is.
func Justified(err error) int {
	// SAFETY: the caller passes the value that open returned.
	return err.(*ParseError).Line // want `a wrapped error defeats this type assertion`
}

// AliasOperand takes an alias of error, which is the same type.
func AliasOperand(err ErrAlias) int {
	pe, ok := err.(*ParseError) // want `a wrapped error defeats this type assertion`
	if !ok {
		return 0
	}
	return pe.Line
}

// InlineCall asserts the error result of a call, with no variable in
// between.
func InlineCall() int {
	return open().(*ParseError).Line // want `a wrapped error defeats this type assertion`
}

// Chained holds two assertions on one line, and the operand of each
// one has the static type error. The rule reports at the .( token, so
// the two diagnostics sit at two columns. A report at the start of the
// operand would put both at one position.
func Chained() int {
	return open().(error).(*ParseError).Line // want `declare a variable of the interface type` `use errors.As with a pointer to a target variable`
}

// InterfaceTarget is the idiom that the net package used before
// errors.As. Wrapping defeats it too, and errors.As takes a pointer to
// an interface variable.
func InterfaceTarget(err error) bool {
	t, ok := err.(interface{ Timeout() bool }) // want `a wrapped error defeats this type assertion; declare a variable of the interface type and use errors.As with a pointer to it; code that must read exactly one level disables the rule instead`
	return ok && t.Timeout()
}

// NamedInterfaceTarget names the interface instead of writing it out.
func NamedInterfaceTarget(err error) int {
	c, ok := err.(Coded) // want `declare a variable of the interface type`
	if !ok {
		return 0
	}
	return c.Code()
}

// TypeParamTarget names one type at each instantiation, so the message
// is the message of a concrete target.
func TypeParamTarget[T error](err error) T {
	return err.(T) // want `use errors.As with a pointer to a target variable`
}

// SwitchOnError gets one diagnostic, at the guard of the switch. Two
// cases must not give two diagnostics.
func SwitchOnError(err error) int {
	switch e := err.(type) { // want `a wrapped error defeats this type switch; use errors.As with a pointer to a target variable, or errors.Is for a sentinel; code that must read exactly one level disables the rule instead`
	case *ParseError:
		return e.Line
	case *fs.PathError:
		return 1
	}
	return 0
}

// SwitchWithoutBinding drops the value and reads the type only.
func SwitchWithoutBinding(err error) int {
	switch err.(type) { // want `a wrapped error defeats this type switch`
	case *ParseError:
		return 1
	}
	return 0
}

// SwitchWithInit puts a statement before the guard. The operand of the
// guard still has the static type error.
func SwitchWithInit() int {
	switch err := open(); e := err.(type) { // want `a wrapped error defeats this type switch`
	case *ParseError:
		return e.Line
	}
	return 0
}

// UnwrapOnce walks the wrap chain itself, like errors.Unwrap. The rule
// reports it: the rule holds no list of exempt shapes. A package that
// implements the chain itself disables the rule.
func UnwrapOnce(err error) error {
	u, ok := err.(interface{ Unwrap() error }) // want `declare a variable of the interface type`
	if !ok {
		return nil
	}
	return u.Unwrap()
}

// UsesAs is the accepted form: errors.As walks the wrap chain.
func UsesAs(err error) int {
	var pe *ParseError
	if errors.As(err, &pe) {
		return pe.Line
	}
	return 0
}

// UsesAsInterface pins the fix that the interface message names:
// errors.As accepts a pointer to an interface variable.
func UsesAsInterface(err error) bool {
	var timeout interface{ Timeout() bool }
	return errors.As(err, &timeout) && timeout.Timeout()
}

// UsesIs is the accepted form for a sentinel.
func UsesIs(err error) bool {
	return errors.Is(err, ErrMissing)
}

// AnyOperand asserts a value of the empty interface. The static type is
// not error, so rule G10 stays quiet. Rules G01 and G05 own that shape.
func AnyOperand(v any) int {
	pe, ok := v.(*ParseError)
	if !ok {
		return 0
	}
	return pe.Line
}

// CodedOperand asserts a value of an interface that embeds error. The
// static type is not error, so the rule stays quiet.
func CodedOperand(c Coded) int {
	pe, ok := c.(*ParseError)
	if !ok {
		return 0
	}
	return pe.Line
}

// WideOperand asserts a value of an interface with the method set of
// error. A method set is no promise about a wrap chain, so the rule
// stays quiet.
func WideOperand(w Wide) int {
	pe, ok := w.(*ParseError)
	if !ok {
		return 0
	}
	return pe.Line
}

// Concrete holds a value of a concrete error type. Go permits no
// assertion on such a value, so the fixture calls the method.
func Concrete(pe *ParseError) string {
	return pe.Error()
}

// SwitchOnAny reads a value of the empty interface. Rule G06 owns that
// shape.
func SwitchOnAny(v any) int {
	switch value := v.(type) {
	case *ParseError:
		return value.Line
	}
	return 0
}

// SwitchOnCoded reads a value of an interface that embeds error.
func SwitchOnCoded(c Coded) int {
	switch value := c.(type) {
	case *ParseError:
		return value.Line
	}
	return 0
}
