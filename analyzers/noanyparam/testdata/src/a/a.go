// Package a holds the fixtures for the noanyparam analyzer. The code is
// deliberately bad: it exists to trigger diagnostics.
package a

import (
	"context"
	"io"
	"time"

	"b"
)

// Alias is the same type as the type it names, so its use sites are
// reported.
type Alias = any

// Payload is a defined type, so its use sites are accepted.
type Payload any

// Empty is a named empty interface. The rule does not take the
// underlying type, so a parameter of Empty is accepted.
type Empty interface{}

func Plain(v any) {} // want `parameter uses any; accept a named domain type`

func Spelled(v interface{}) {} // want `parameter uses interface\{\}; accept a named domain type`

func AliasParam(v Alias) {} // want `parameter uses Alias; accept a named domain type`

func PayloadParam(v Payload) {}

func EmptyParam(v Empty) {}

func Group(first, second any) {} // want `parameter uses any; accept a named domain type`

func Mixed(name string, v any, count int) {} // want `parameter uses any; accept a named domain type`

func Clean(name string, count int) {}

// The rule tests the parameter type itself, so a type that only holds
// the empty interface is accepted.
func SliceParam(v []any) {}

func MapParam(v map[string]any) {}

func PointerParam(v *any) {}

// Generic parameters carry a constraint, so they are accepted.
func Generic[T any](v T) {}

// Box is a generic type. An instantiation is a named type.
type Box[T any] struct{ v T }

func TakeBox(boxed Box[any]) {}

// Cause mirrors the upstream error-wrapping helper.
func Cause(cause any) error { return nil }

// A group keeps the report, because only one of its names carries the
// error-wrapping contract.
func CauseGroup(cause, value any) error { return nil } // want `parameter uses any; accept a named domain type`

// Errorf is an fmt-style helper: a format parameter controls the tail.
func Errorf(format string, args ...any) error { return nil }

// Debugf has no parameter named format, but its own name ends in "f".
func Debugf(msg string, args ...any) {}

// The name test ignores case and accepts a prefix or a suffix.
func Report(FORMAT string, args ...any) {}

func Log(msgFormat string, args ...any) {}

func Render(formatString string, args ...any) {}

// A name that only holds "format" inside itself is not a format
// parameter.
func Notify(informationText string, args ...any) {} // want `parameter uses \.\.\.any; accept a named domain type`

func Rewrite(reformatted string, args ...any) {} // want `parameter uses \.\.\.any; accept a named domain type`

// The exemption covers the variadic tail only.
func Wrapf(format string, v any) {} // want `parameter uses any; accept a named domain type`

// Emit has no format evidence, so the tail is reported.
func Emit(topic string, args ...any) {} // want `parameter uses \.\.\.any; accept a named domain type`

// Broadcast has no string parameter before the tail.
func Broadcast(args ...any) {} // want `parameter uses \.\.\.any; accept a named domain type`

// Levelf ends in "f" but takes no string parameter.
func Levelf(level int, args ...any) {} // want `parameter uses \.\.\.any; accept a named domain type`

// Countf takes a defined string type, which is not the predeclared
// string, so the tail is reported.
type Name string

func Countf(n Name, args ...any) {} // want `parameter uses \.\.\.any; accept a named domain type`

// f has one letter, so the name test does not exempt it.
func f(msg string, args ...any) {} // want `parameter uses \.\.\.any; accept a named domain type`

// Logf is a declared function type, and the type name ends in "f".
type Logf func(string, ...any)

// Sink is a declared function type with no format evidence.
type Sink func(topic string, args ...any) // want `parameter uses \.\.\.any; accept a named domain type`

// Handler is a declared function type.
type Handler func(v any) // want `parameter uses any; accept a named domain type`

// Raw declares its parameter without a name.
type Raw func(any) // want `parameter uses any; accept a named domain type`

// Store is an interface. Its method signatures are reported.
type Store interface {
	Save(v any) error // want `parameter uses any; accept a named domain type`
	Warnf(msg string, args ...any)
	// CONTRACT: b.Register sets the parameter of a handler.
	Handle(v any) error
	Load(key string) error
}

// Literal holds a function literal, which has no name of its own.
var Literal = func(v any) {} // want `parameter uses any; accept a named domain type`

var LiteralTail = func(msg string, args ...any) {} // want `parameter uses \.\.\.any; accept a named domain type`

// Locals are signatures too, wherever they sit.
func Locals() {
	inner := func(v any) {} // want `parameter uses any; accept a named domain type`
	// CONTRACT: b.Register sets the parameter of a handler.
	justified := func(v any) {}
	_, _ = inner, justified
}

// Callback holds a function type in a parameter. The inner signature is
// reported once.
func Callback(cb func(v any)) {} // want `parameter uses any; accept a named domain type`

// Unnamed holds a function type with no parameter name.
func Unnamed(func(v any)) {} // want `parameter uses any; accept a named domain type`

// Config holds a function type in a field.
type Config struct {
	Run func(v any) // want `parameter uses any; accept a named domain type`
}

// ctxValue implements context.Context, which declares
// Value(key any) any, so the parameter is an external contract.
type ctxValue struct{}

func (ctxValue) Deadline() (time.Time, bool) { return time.Time{}, false }

func (ctxValue) Done() <-chan struct{} { return nil }

func (ctxValue) Err() error { return nil }

func (ctxValue) Value(key any) any { return nil }

var _ context.Context = ctxValue{}

// mixed declares Value with a value receiver and the rest with pointer
// receivers, so only *mixed implements context.Context.
type mixed struct{}

func (mixed) Value(key any) any { return nil }

func (*mixed) Deadline() (time.Time, bool) { return time.Time{}, false }

func (*mixed) Done() <-chan struct{} { return nil }

func (*mixed) Err() error { return nil }

var _ context.Context = &mixed{}

// loner has the shape of the contract methods but implements no
// imported interface.
type loner struct{}

func (loner) Value(key any) any { return nil } // want `parameter uses any; accept a named domain type`

// Err names a method of context.Context that takes no parameter, so the
// position test rejects the contract.
func (loner) Err(v any) error { return nil } // want `parameter uses any; accept a named domain type`

// Write names a method of io.Writer, which takes a byte slice.
func (loner) Write(p any) (int, error) { return 0, nil } // want `parameter uses any; accept a named domain type`

var _ io.Writer

// Box implements no imported interface, so its method is reported.
func (bx *Box[T]) Set(v any) {} // want `parameter uses any; accept a named domain type`

// genericCtx is a generic type that implements context.Context. The
// receiver of its method is an instantiation, so the contract holds.
type genericCtx[T any] struct{ v T }

func (genericCtx[T]) Deadline() (time.Time, bool) { return time.Time{}, false }

func (genericCtx[T]) Done() <-chan struct{} { return nil }

func (genericCtx[T]) Err() error { return nil }

func (genericCtx[T]) Value(key any) any { return nil }

var _ context.Context = genericCtx[int]{}

// cache implements context.Context for the instantiation cache[any]
// only, because its type parameter takes part in the match. The rule
// loses the contract and reports the parameter; a CONTRACT comment is
// the remedy.
type cache[T any] struct{}

func (cache[T]) Deadline() (time.Time, bool) { return time.Time{}, false }

func (cache[T]) Done() <-chan struct{} { return nil }

func (cache[T]) Err() error { return nil }

func (cache[T]) Value(key any) T { var zero T; return zero } // want `parameter uses any; accept a named domain type`

var _ context.Context = cache[any]{}

// spy has the shape of the unexported interface of package b. The rule
// tests exported interfaces only, so the parameter is reported.
type spy struct{}

func (spy) Watch(v any) {} // want `parameter uses any; accept a named domain type`

// counter has the shape of the method of the constraint b.Number. A
// constraint is not a method set, so no value implements it and the
// parameter is reported.
type counter struct{}

func (counter) Log(v any) {} // want `parameter uses any; accept a named domain type`

// printer implements b.Printer, so the variadic tail of Emit is an
// external contract.
type printer struct{}

func (printer) Emit(topic string, args ...any) {}

func (printer) Tag(topic, name string, v any) {}

var _ b.Printer = printer{}

// handle passes to b.Register, which the analyzer cannot see, so the
// comment carries the evidence.
//
// CONTRACT: b.Register sets the parameter of a handler.
func handle(v any) {}

func init() { b.Register(handle) }

// A method takes the comment too.
type store struct{}

// CONTRACT: b.Register sets the parameter of a handler.
func (store) Handle(v any) {}

// The comment must sit directly above the declaration.

// CONTRACT: a blank line separates this comment from the declaration.

func Detached(v any) {} // want `parameter uses any; accept a named domain type`

// contract: the marker is case sensitive.
func Lowercase(v any) {} // want `parameter uses any; accept a named domain type`

func SameLine(v any) {} // CONTRACT: a comment beside the code justifies nothing // want `parameter uses any; accept a named domain type`

var trailing = 1 // CONTRACT: this comment trails code, so it justifies nothing.
func TrailingAbove(v any) {} // want `parameter uses any; accept a named domain type`
