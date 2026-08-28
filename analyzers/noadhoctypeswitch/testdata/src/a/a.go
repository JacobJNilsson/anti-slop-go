// Package a holds the fixtures for the noadhoctypeswitch analyzer. The
// code is deliberately bad: it exists to trigger diagnostics.
//
// Rule G03 reports many of the signatures below. analysistest reads the
// expectation comments of one analyzer, so those reports leave no mark
// here.
package a

import (
	"io"

	"b"
)

// Alias names the empty interface, so a value of it is an any value.
type Alias = any

// Payload is a defined type. It is a domain type of this package, and a
// switch on one of its values reads a type the package owns.
type Payload any

// parseError gives the error fixtures a concrete error type to name.
type parseError struct{}

func (parseError) Error() string { return "parse" }

// decode returns an any value, so a call of it is an operand of the
// empty interface type.
func decode() any { return nil }

// Plain switches on a parameter that carries no evidence. Rule G03
// reports the signature as well: one cause gives two findings, and the
// signature is the one to fix.
func Plain(v any) {
	switch v.(type) { // want `this type switch reads the dynamic type of an any value`
	case int:
	case string:
	}
}

// The binding form asks the same question, so it gets the same report.

func Binding(v any) int {
	switch value := v.(type) { // want `this type switch reads the dynamic type of an any value`
	case int:
		return value
	case string:
		return len(value)
	}

	return 0
}

// One switch gets one report, whatever the number of its cases. The
// full message sits here, and the other fixtures name its first clause.

func ManyCases(v any) string {
	switch v.(type) { // want `^this type switch reads the dynamic type of an any value; branch on a domain value, or name a boundary package with boundary-packages \(-noadhoctypeswitch\.boundary\)$`
	case int:
		return "int"
	case string, bool:
		return "text or flag"
	case nil:
		return "nil"
	default:
		return "other"
	}
}

// An init statement changes no question the guard asks.

func WithInit(v any) int {
	switch base := 1; value := v.(type) { // want `this type switch reads the dynamic type of an any value`
	case int:
		return value + base
	}

	return 0
}

// Parentheses hold the same operand.

func Parenthesized(v any) {
	switch (v).(type) { // want `this type switch reads the dynamic type of an any value`
	case int:
	}
}

// chain gives the fixture below an operand that spans two lines.
type chain struct{}

func (chain) Next() any { return nil }

// The diagnostic sits at the .( token, so it names the line of the
// guard and not the line where the operand starts.
func MultiLineOperand(c chain) {
	switch c.
		Next().(type) { // want `this type switch reads the dynamic type of an any value`
	case int:
	}
}

// An alias is the same type as the type it names.

func AliasOperand(v Alias) {
	switch v.(type) { // want `this type switch reads the dynamic type of an any value`
	case int:
	}
}

// A local variable holds no evidence either. Rule G05 reports the
// widening that fills it, and rule G06 reports the switch that reads it
// back.
func LocalVariable(n int) {
	var v any = n
	switch v.(type) { // want `this type switch reads the dynamic type of an any value`
	case int:
	}
}

// A conversion to the empty interface gives an operand of that type.
func Conversion(n int) {
	switch any(n).(type) { // want `this type switch reads the dynamic type of an any value`
	case int:
	}
}

// A call that returns an any value is no parameter, so no signature
// admitted the value and no contract covers it.
func CallResult() {
	switch decode().(type) { // want `this type switch reads the dynamic type of an any value`
	case int:
	}
}

// Go reads a value of a parameterized type through an interface only.
// This switch is therefore the one way to branch on the instantiation,
// and the widening is no choice of the author. Rule G05 leaves the same
// widening alone.
func Instantiation[T any](v T) string {
	switch any(v).(type) {
	case int:
		return "int"
	}

	return ""
}

// The walk reads the type parameter through a pointer as well.
func PointerInstantiation[T any](v *T) string {
	switch any(v).(type) {
	case *int:
		return "int"
	}

	return ""
}

// A conversion of a value that names no type parameter keeps its
// report, in a generic function with the rest.
func GenericConversion[T any](v T, n int) string {
	_ = v
	switch any(n).(type) { // want `this type switch reads the dynamic type of an any value`
	case int:
		return "int"
	}

	return ""
}

// wrap is a call and no conversion, so the operand of the switch below
// carries no type that the walk can read.
func wrap(v any) any { return v }

func WrappedCall(v any) string {
	switch wrap(v).(type) { // want `this type switch reads the dynamic type of an any value`
	case int:
		return "int"
	}

	return ""
}

// recover returns an any value, because the language sets that
// signature. The program owns the value that it panicked with, so the
// rule reports the switch.
func Recovered() {
	defer func() {
		switch recover().(type) { // want `this type switch reads the dynamic type of an any value`
		case error:
		}
	}()
}

// A named result is no parameter of the signature.
func NamedResult() (v any) {
	switch v.(type) { // want `this type switch reads the dynamic type of an any value`
	case int:
	}

	return v
}

// A field of a struct is no parameter. The switch reads a value that no
// signature admitted.
type box struct{ value any }

func (bx box) Kind() string {
	switch bx.value.(type) { // want `this type switch reads the dynamic type of an any value`
	case int:
		return "int"
	}

	return ""
}

// Each switch gets its own report.

func Nested(outer, inner any) {
	switch outer.(type) { // want `this type switch reads the dynamic type of an any value`
	case int:
		switch inner.(type) { // want `this type switch reads the dynamic type of an any value`
		case string:
		}
	}
}

// A single type assertion is another shape. Rule G01 asks for the
// justification of it and rule G05 reads the widening before it. This
// rule reads the guard of a switch only.
func SingleAssertion(v any) int {
	n, isNumber := v.(int)
	if !isNumber {
		return 0
	}

	return n
}

// A defined type is a domain type, so the switch reads a type of this
// package and stays clean.
func PayloadOperand(p Payload) {
	switch p.(type) {
	case int:
	}
}

// An interface with a method is no empty interface, so the operand
// carries the evidence of that method set.
func NarrowInterface(r io.Reader) {
	switch r.(type) {
	case io.ReadCloser:
	}
}

// The static type of an error value is no empty interface, so this rule
// reads nothing here. Rule G10 owns the shape and reports it.
func ErrorOperand(err error) {
	switch err.(type) {
	case parseError:
	}
}

// An alias of the error type is the error type, so it stays out here
// too.
type ErrAlias = error

func ErrorAliasOperand(err ErrAlias) {
	switch err.(type) {
	case parseError:
	}
}

// The name "cause" exempts a parameter under rule G03. It names no API
// that sets the signature, so it is no evidence about the value and the
// switch keeps its report.

func Cause(cause any) {
	switch cause.(type) { // want `this type switch reads the dynamic type of an any value`
	case int:
	}
}

// handle passes to b.Register, which the analyzer cannot see, so the
// comment carries the evidence. The switch consumes the contract that
// the comment states.
//
// CONTRACT: b.Register sets the parameter of a handler.
func handle(v any) {
	switch v.(type) {
	case int:
	}
}

// The evidence reads the operand through its parentheses, so the
// contract of the parameter holds here too.
//
// CONTRACT: b.Register sets the parameter of a handler.
func parenthesizedHandle(v any) {
	switch (v).(type) {
	case int:
	}
}

// The evidence reads the parameter itself and no copy of it. A new
// variable carries no contract, and an assignment to the parameter
// keeps the one the comment states.
//
// CONTRACT: b.Register sets the parameter of a handler.
func copiedHandle(v any) {
	w := v
	switch w.(type) { // want `this type switch reads the dynamic type of an any value`
	case int:
	}
	v = wrap(v)
	switch v.(type) {
	case int:
	}
}

func init() {
	b.Register(handle)
	b.Register(parenthesizedHandle)
	b.Register(copiedHandle)
}

// The justification belongs to the signature that admitted the any. A
// comment above the switch justifies nothing, because the reader of the
// signature never sees it.

func CommentOnSwitch(v any) {
	// CONTRACT: this comment sits above the statement and not above the
	// signature that admitted the value.

	switch v.(type) { // want `this type switch reads the dynamic type of an any value`
	case int:
	}
}

// jsonDecoder implements b.Decoder, so the external interface sets the
// parameter of Decode and the switch consumes that contract.
type jsonDecoder struct{}

func (jsonDecoder) Format() string { return "json" }

func (jsonDecoder) Decode(value any) error {
	switch value.(type) {
	case int:
	}

	return nil
}

// Log names no method of an interface that package a imports, so the
// receiver carries no evidence for this parameter.
func (jsonDecoder) Log(v any) {
	switch v.(type) { // want `this type switch reads the dynamic type of an any value`
	case int:
	}
}

// Tag names a method of b.Tagger, which declares the empty interface at
// position two. The parameter here sits at position one, so the
// contract of that name covers it not.
func (jsonDecoder) Tag(value any) {
	switch value.(type) { // want `this type switch reads the dynamic type of an any value`
	case int:
	}
}

var _ b.Decoder = jsonDecoder{}

// tagger implements b.Tagger, and the empty interface of Tag sits at
// the position the interface declares.
type tagger struct{}

func (tagger) Format() string { return "tag" }

func (tagger) Tag(name string, value any) {
	switch value.(type) {
	case int:
	}
}

var _ b.Tagger = tagger{}

// loner declares the method of b.Decoder and implements no interface of
// b, because it declares no Format method. The parameter therefore
// carries no evidence.
type loner struct{}

func (loner) Decode(value any) error {
	switch value.(type) { // want `this type switch reads the dynamic type of an any value`
	case int:
	}

	return nil
}

// wide names the method of b.Decoder and takes two parameters, so the
// position of its empty interface sits outside the signature that the
// interface declares. The rule reads that position and reports.
type wide struct{}

func (wide) Decode(name string, value any) error {
	switch value.(type) { // want `this type switch reads the dynamic type of an any value`
	case int:
	}

	return nil
}

// A function literal carries a signature of its own, and that signature
// decides for its own parameter.
func Closures() {
	inner := func(v any) {
		switch v.(type) { // want `this type switch reads the dynamic type of an any value`
		case int:
		}
	}
	// CONTRACT: b.Register sets the parameter of a handler.
	justified := func(v any) {
		switch v.(type) {
		case int:
		}
	}
	inner(1)
	justified(1)
}

// The signature that declares the parameter decides, wherever the
// switch sits. A literal that reads a contracted parameter of the
// function around it therefore stays clean.
//
// CONTRACT: b.Register sets the parameter of a handler.
func capture(v any) func() {
	return func() {
		switch v.(type) {
		case int:
		}
	}
}

// The same shape without the comment keeps its report. Rule G03 reports
// the signature, which is the line the author fixes.

func Capture(v any) func() {
	return func() {
		switch v.(type) { // want `this type switch reads the dynamic type of an any value`
		case int:
		}
	}
}
