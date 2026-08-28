// Package a holds the fixtures for the nointerfacereturn analyzer. The
// code is deliberately bad: it exists to trigger diagnostics.
package a

import (
	"errors"

	"b"
)

// Storage is the interface that most fixtures return.
type Storage interface {
	Load(key string) string
}

// Alias names the same interface, so a result of Alias is a result of
// Storage.
type Alias = Storage

// Empty is a defined interface with no method. It is no alias of any,
// so this rule judges it and rule G04 leaves it alone.
type Empty interface{}

// CloseError embeds error. It holds the method set of error, and it is
// another type, so the exemption of error does not reach it.
type CloseError interface {
	error
	Path() string
}

type fileStore struct{}

func (fileStore) Load(key string) string { return key }

type memStore struct{}

func (memStore) Load(key string) string { return key }

type item struct{}

func (item) Name() string { return "item" }

type closeError struct{}

func (closeError) Error() string { return "closed" }

func (closeError) Path() string { return "/" }

// One concrete type through every path, in an exported function.

func NewStore() Storage { return &fileStore{} } // want `result uses Storage, and every return builds \*fileStore; return the concrete type`

// An unexported function carries the same proof.

func newStore() Storage { return fileStore{} } // want `result uses Storage, and every return builds fileStore; return the concrete type`

// An alias is the same type.

func NewAlias() Alias { return &fileStore{} } // want `result uses Alias, and every return builds \*fileStore; return the concrete type`

// A defined interface with no method is a domain type, and the rule
// judges it.

func NewEmpty() Empty { return &fileStore{} } // want `result uses Empty, and every return builds \*fileStore; return the concrete type`

// An interface that embeds error is another type than error.

func Closed() CloseError { return closeError{} } // want `result uses CloseError, and every return builds closeError; return the concrete type`

// A local variable carries the proof as well as a composite literal.

func BuildStore() Storage { // want `result uses Storage, and every return builds \*fileStore; return the concrete type`
	s := &fileStore{}

	return s
}

// The constructor shape: one concrete type, and nil beside an error.
// The caller sees one concrete type or nil, so the proof holds.

func OpenStore(name string) (Storage, error) { // want `result uses Storage, and every return builds \*fileStore or nil; return the concrete type`
	if name == "" {
		return nil, errors.New("open: no name")
	}

	return &fileStore{}, nil
}

// nil alone names no concrete type.
func NoStore() Storage { return nil }

// Two concrete types are honest interface use.
func PickStore(mem bool) Storage {
	if mem {
		return memStore{}
	}

	return &fileStore{}
}

// The evidence sits with the caller of this function, not here.
func Passthrough(s Storage) Storage { return s }

// The result of the call is already the interface.
func Reopen(name string) (Storage, error) { return OpenStore(name) }

func openFile(name string) (*fileStore, error) { return &fileStore{}, nil }

// A call with several results carries the proof in its tuple.

func OpenAgain(name string) (Storage, error) { // want `result uses Storage, and every return builds \*fileStore; return the concrete type`
	return openFile(name)
}

func splitFile(name string) (int, *fileStore) { return len(name), &fileStore{} }

// The tuple carries a type for each result, so the rule reads the one
// it judges.

func SplitStore(name string) (int, Storage) { // want `result uses Storage, and every return builds \*fileStore; return the concrete type`
	return splitFile(name)
}

// A conversion states the widening, so the static type of the operand
// is the interface and the rule reads no concrete type.
func Widened() Storage { return Storage(&fileStore{}) }

// A body with no return statement proves nothing.
func Unimplemented() Storage { panic("not implemented") }

// A declaration with no body, such as an assembly stub, proves nothing.
func External() Storage

// A naked return reads the result variable, whose type is the
// interface, so the body proves no concrete type.
func Named() (s Storage) {
	s = &fileStore{}

	return
}

// The same function with the value in the return statement proves it.

func NamedExplicit() (s Storage) { return &fileStore{} } // want `result uses Storage, and every return builds \*fileStore; return the concrete type`

// Go groups names, so this signature holds two results. Both results
// give one message at one position, and the reader needs it once.

func Group() (first, second Storage) { return &fileStore{}, &fileStore{} } // want `result uses Storage, and every return builds \*fileStore; return the concrete type`

// The results of a group can build two types. The two messages differ,
// so the reader gets both.

func Split() (first, second Storage) { return &fileStore{}, memStore{} } // want `result uses Storage, and every return builds \*fileStore; return the concrete type` `result uses Storage, and every return builds memStore; return the concrete type`

// The rule judges each result position on its own. Position one holds
// two concrete types and stays clean; position two holds one.

func Mixed(mem bool) (
	Storage,
	b.Item, // want `result uses b.Item, and every return builds item; return the concrete type`
) {
	if mem {
		return memStore{}, item{}
	}

	return &fileStore{}, item{}
}

// A function literal takes its signature from the call or the variable
// that holds it, so the rule reads a function declaration only.
var Build = func() Storage { return &fileStore{} }

// The return of a nested literal belongs to that literal. The rule
// reads the returns of this body only, so the proof holds here.

func Outer() Storage { // want `result uses Storage, and every return builds \*fileStore; return the concrete type`
	inner := func() Storage { return memStore{} }
	_ = inner

	return &fileStore{}
}

// Holder is a generic interface.
type Holder[T any] interface {
	Get() T
}

type holder[T any] struct{ v T }

func (h holder[T]) Get() T { return h.v }

// A result type that mentions a type parameter is out of scope.
func Boxed[T any](v T) Holder[T] { return holder[T]{v} }

// Every other result of a generic function follows the rule.

func Keep[T any](v T) Storage { return &fileStore{} } // want `result uses Storage, and every return builds \*fileStore; return the concrete type`

// The error result is exempt. Go returns errors through the interface.
func Save() error { return closeError{} }

// The empty interface belongs to rule G04.
func Anything() any { return &fileStore{} }

func Spelled() interface{} { return &fileStore{} }

// A result that is no interface is no business of this rule.
func Count() int { return 1 }

// A function with no result is no business of this rule either.
func Reset() {}

// builder implements b.Factory, which declares Build with the same
// result, so the signature is an external contract.
type builder struct{}

func (builder) Build() b.Item { return item{} }

func (builder) Kind() string { return "builder" }

var _ b.Factory = builder{}

// Load names no method of an imported interface, so the contract test
// finds no signature and the result reports.
func (builder) Load(key string) Storage { return &fileStore{} } // want `result uses Storage, and every return builds \*fileStore; return the concrete type`

// loner has the shape of one method of b.Factory and implements no
// imported interface, because it declares no other method of one.
type loner struct{}

func (loner) Build() b.Item { return item{} } // want `result uses b.Item, and every return builds item; return the concrete type`

// twin names the method of b.Factory that declares one result. The
// position test reads the length of the declared results, so the
// contract fails at the second result. twin implements no imported
// interface either.
type twin struct{}

func (twin) Build() (b.Item, b.Item) { return item{}, item{} } // want `result uses b.Item, and every return builds item; return the concrete type` `result uses b.Item, and every return builds item; return the concrete type`

type sized struct{}

func (sized) Size() int { return 1 }

// splitter implements b.Splitter, which declares both results, so both
// results are an external contract. The two results hold two different
// interfaces, so the rule must read the declared result at the position
// it judges.
type splitter struct{}

func (splitter) Split() (b.Item, b.Thing) { return item{}, sized{} }

var _ b.Splitter = splitter{}

// Sink is an interface of this package, and sink implements it. The
// method cannot narrow its result: sink would stop satisfying Sink, and
// the package would stop compiling.
type Sink interface {
	Next() Storage
	Kind() string
}

type sink struct{}

func (sink) Next() Storage { return &fileStore{} }

func (sink) Kind() string { return "sink" }

var _ Sink = sink{}

// The compiler answers the same way for an unexported interface of this
// package, so the scan reads one.
type stream interface {
	Head() Storage
	Tail() Storage
}

type list struct{}

func (list) Head() Storage { return &fileStore{} }

func (list) Tail() Storage { return memStore{} }

var _ stream = list{}

// halfSink declares the method of Sink and implements no interface, so
// it can narrow the result alone.
type halfSink struct{}

func (halfSink) Next() Storage { return &fileStore{} } // want `result uses Storage, and every return builds \*fileStore; return the concrete type`

// newItem passes to b.Register, which the analyzer cannot see, so the
// comment carries the evidence.
//
// CONTRACT: b.Register fixes the result of a build function.
func newItem() b.Item { return item{} }

func init() { b.Register(newItem) }

// The comment must sit directly above the declaration.

// CONTRACT: a blank line separates this comment from the declaration.

func Detached() b.Item { return item{} } // want `result uses b.Item, and every return builds item; return the concrete type`
