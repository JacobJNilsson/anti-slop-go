// Package a holds the fixtures for the signature package. The code is
// deliberately bad: it exists to trigger diagnostics from the probe
// analyzer of the test.
package a

import (
	"context"
	"time"

	"b"
)

// Alias is the same type as the type it names.
type Alias = any

// Payload is a defined type, so it is a domain type.
type Payload any

func Plain(v any) {} // want `parameter uses the empty interface`

func Spelled(v interface{}) {} // want `parameter uses the empty interface`

func AliasParam(v Alias) {} // want `parameter uses the empty interface`

func Defined(v Payload) {}

func Clean(name string) {}

// A group holds two parameters, and the probe reports the field once.
func Group(first, second any) {} // want `parameter uses the empty interface`

// An unnamed parameter is one parameter.
func Unnamed(any) {} // want `parameter uses the empty interface`

// CONTRACT: an external API sets this signature.
func Justified(v any) {}

// A method takes the comment too.
type store struct{}

// CONTRACT: an external API sets this signature.
func (store) Handle(v any) {}

// CONTRACT: a blank line breaks the link to the declaration.

func Detached(v any) {} // want `parameter uses the empty interface`

// contract: the marker is case sensitive.
func Lowercase(v any) {} // want `parameter uses the empty interface`

var trailing = 1 // CONTRACT: this comment trails code, so it justifies nothing.
func TrailingAbove(v any) {} // want `parameter uses the empty interface`

// CONTRACT: an external API sets this signature.
var Literal = func(v any) {}

var Reported = func(v any) {} // want `parameter uses the empty interface`

// Locals hold signatures too.
func Locals() {
	inner := func(v any) {} // want `parameter uses the empty interface`
	// CONTRACT: an external API sets this signature.
	justified := func(v any) {}
	_, _ = inner, justified
}

// Store is an interface. A comment above one method covers that method
// only.
type Store interface {
	Save(v any) error // want `parameter uses the empty interface`
	// CONTRACT: an external API sets this signature.
	Handle(v any) error
}

// ctxValue implements context.Context, which declares
// Value(key any) any, so the parameter is an external contract.
type ctxValue struct{}

func (ctxValue) Deadline() (time.Time, bool) { return time.Time{}, false }

func (ctxValue) Done() <-chan struct{} { return nil }

func (ctxValue) Err() error { return nil }

func (ctxValue) Value(key any) any { return nil }

var _ context.Context = ctxValue{}

// loner has the shape of the contract method but implements no
// imported interface.
type loner struct{}

func (loner) Value(key any) any { return nil } // want `parameter uses the empty interface`

// Err names a method of context.Context that takes no parameter, so the
// position test rejects the contract.
func (loner) Err(v any) error { return nil } // want `parameter uses the empty interface`

// Fetch names no method of an imported interface at all.
func (loner) Fetch(v any) error { return nil } // want `parameter uses the empty interface`

// counter has the shape of the method of the constraint b.Key. A
// constraint is not a method set, so no value implements it.
type counter struct{}

func (counter) Log(v any) {} // want `parameter uses the empty interface`

var _ = b.Sum[counter]

// A signature can start on a later line than the specification or the
// statement that holds it.

// CONTRACT: an external API sets this signature.
var handlers = []func(v any){
	func(v any) {},
}

var reported = []func(v any){ // want `parameter uses the empty interface`
	func(v any) {}, // want `parameter uses the empty interface`
}

func wire() {
	// CONTRACT: an external API sets this signature.
	call(
		func(v any) {},
	)
	call(
		func(v any) {}, // want `parameter uses the empty interface`
	)
}

func call(handle func(v any)) {} // want `parameter uses the empty interface`

// A comment above a var block covers no entry of the block.

// CONTRACT: this comment sits above the block, not above an entry.
var (
	blockOne = func(v any) {} // want `parameter uses the empty interface`
	blockTwo = func(v any) {} // want `parameter uses the empty interface`
)

// A comment above a type covers no field of the type.

// CONTRACT: this comment sits above the type, not above a field.
type Table struct {
	Run   func(v any) // want `parameter uses the empty interface`
	Retry func(v any) // want `parameter uses the empty interface`
}

// A split name list puts the field on an earlier line than its
// signature, so the rule keeps the report.
type Split struct {
	// CONTRACT: this comment sits above the name list.
	One,
	Two func(v any) // want `parameter uses the empty interface`
}

// A comment covers every signature on the line it sits above.

// CONTRACT: an external API sets this signature.
func Both(cb func(v any), v any) {}

// The comment must sit above the statement that holds the signature,
// not above an outer statement.
func nested() {
	// CONTRACT: this comment sits above the if, not above the call.
	if true {
		call(func(v any) {}) // want `parameter uses the empty interface`
	}
}
