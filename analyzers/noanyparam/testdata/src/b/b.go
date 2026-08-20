// Package b holds the imported types that fixture package a uses. A
// type of this package is external to package a, so package a cannot
// change the signatures that b declares.
package b

// Register takes a handler. A call is not evidence the analyzer can
// read, so package a must justify such a handler with a comment.
func Register(handle func(v any)) {}

// Printer declares a variadic tail, so an implementation of Printer
// keeps that tail.
type Printer interface {
	Emit(topic string, args ...any)
	// Tag groups two names, so the empty interface sits at position two.
	Tag(topic, name string, v any)
}

// watcher is unexported, so package a cannot name it. The rule tests
// exported interfaces only.
type watcher interface {
	Watch(v any)
}

// Watch runs the unexported interface, so package b keeps a use of it.
func Watch(w watcher) { w.Watch(nil) }

// Number is a constraint: it embeds comparable as well as a method, so
// it is not a method set and no value can implement it. types.Implements
// answers true against the method set alone, so the analyzer must drop
// such an interface before it asks.
type Number interface {
	comparable
	Log(v any)
}

// Sum runs the constraint, so package b keeps a use of it.
func Sum[T Number](values []T) { }
