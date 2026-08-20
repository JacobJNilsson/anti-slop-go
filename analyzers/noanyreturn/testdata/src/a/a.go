// Package a holds the fixtures for the noanyreturn analyzer. The code
// is deliberately bad: it exists to trigger diagnostics.
package a

import (
	"context"
	"time"

	"b"
)

// Alias is the same type as the type it names, so its use sites are
// reported.
type Alias = any

// Payload is a defined type, so its use sites are accepted.
type Payload any

// Empty is a named empty interface. The rule does not take the
// underlying type, so a result of Empty is accepted.
type Empty interface{}

func Plain() any { return nil } // want `result uses any; return a concrete type`

func Spelled() interface{} { return nil } // want `result uses interface\{\}; return a concrete type`

func AliasResult() Alias { return nil } // want `result uses Alias; return a concrete type`

func PayloadResult() Payload { return nil }

func EmptyResult() Empty { return nil }

func Named() (v any) { return nil } // want `result uses any; return a concrete type`

func Group() (first, second any) { return nil, nil } // want `result uses any; return a concrete type`

func Multi() (any, error) { return nil, nil } // want `result uses any; return a concrete type`

func Second() (int, any) { return 0, nil } // want `result uses any; return a concrete type`

func Clean() (int, error) { return 0, nil }

func NoResult(key string) {}

// The rule tests the result type itself, so a type that only holds the
// empty interface is accepted.
func SliceResult() []any { return nil }

func MapResult() map[string]any { return nil }

func PointerResult() *any { return nil }

// Generic results carry a constraint, so they are accepted.
func Generic[T any]() T { var zero T; return zero }

// Box is a generic type. An instantiation is a named type.
type Box[T any] struct{ v T }

func GiveBox() Box[any] { return Box[any]{} }

// Loader is a declared function type.
type Loader func() any // want `result uses any; return a concrete type`

// Store is an interface. Its method signatures are reported.
type Store interface {
	Load(key string) any // want `result uses any; return a concrete type`
	// CONTRACT: b.Register sets the result of a source.
	Next() any
	Count(key string) int
}

// Literal holds a function literal.
var Literal = func() any { return nil } // want `result uses any; return a concrete type`

// Locals are signatures too, wherever they sit.
func Locals() {
	inner := func() any { return nil } // want `result uses any; return a concrete type`
	// CONTRACT: b.Register sets the result of a source.
	justified := func() any { return nil }
	_, _ = inner, justified
}

// Callback holds a function type in a parameter. The inner signature is
// reported once.
func Callback(cb func() any) {} // want `result uses any; return a concrete type`

// Config holds a function type in a field.
type Config struct {
	Run func() any // want `result uses any; return a concrete type`
}

// ctxValue implements context.Context, which declares
// Value(key any) any, so the result is an external contract.
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

func (loner) Value(key any) any { return nil } // want `result uses any; return a concrete type`

// Err names a method of context.Context that returns an error.
func (loner) Err() any { return nil } // want `result uses any; return a concrete type`

// Done names a method of context.Context that returns one value, so the
// position test rejects the contract for both results.
func (loner) Done() (any, any) { return nil, nil } // want `result uses any; return a concrete type` `result uses any; return a concrete type`

// Box implements no imported interface, so its method is reported.
func (bx *Box[T]) Get() any { return nil } // want `result uses any; return a concrete type`

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
// loses the contract and reports the result; a CONTRACT comment is the
// remedy.
type cache[T any] struct{}

func (cache[T]) Deadline() (time.Time, bool) { return time.Time{}, false }

func (cache[T]) Done() <-chan struct{} { return nil }

func (cache[T]) Err() error { return nil }

func (cache[T]) Value(key T) any { return nil } // want `result uses any; return a concrete type`

var _ context.Context = cache[any]{}

// spy has the shape of the unexported interface of package b. The rule
// tests exported interfaces only, so the result is reported.
type spy struct{}

func (spy) Watch() any { return nil } // want `result uses any; return a concrete type`

// counter has the shape of the method of the constraint b.Number. A
// constraint is not a method set, so no value implements it and the
// result is reported.
type counter struct{}

func (counter) Load() any { return nil } // want `result uses any; return a concrete type`

// pager implements b.Pager, so the result at position two is an
// external contract.
type pager struct{}

func (pager) Page() (first, last int, cursor any) { return 0, 0, nil }

var _ b.Pager = pager{}

// next passes to b.Register, which the analyzer cannot see, so the
// comment carries the evidence.
//
// CONTRACT: b.Register sets the result of a source.
func next() any { return nil }

func init() { b.Register(next) }

// A method takes the comment too.
type source struct{}

// CONTRACT: b.Register sets the result of a source.
func (source) Next() any { return nil }

// The comment must sit directly above the declaration.

// CONTRACT: a blank line separates this comment from the declaration.

func Detached() any { return nil } // want `result uses any; return a concrete type`

// contract: the marker is case sensitive.
func Lowercase() any { return nil } // want `result uses any; return a concrete type`

func SameLine() any { return nil } // CONTRACT: a comment beside the code justifies nothing // want `result uses any; return a concrete type`

var trailing = 1 // CONTRACT: this comment trails code, so it justifies nothing.
func TrailingAbove() any { return nil } // want `result uses any; return a concrete type`
