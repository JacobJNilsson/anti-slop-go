// Package b holds the imported types that fixture package a uses. A
// type of this package is external to package a, so package a cannot
// change the signatures that b declares.
package b

// Register takes a source. A call is not evidence the analyzer can
// read, so package a must justify such a source with a comment.
func Register(next func() any) {}

// Pager groups two results, so the empty interface sits at position
// two in an implementation.
type Pager interface {
	Page() (first, last int, cursor any)
}

// watcher is unexported, so package a cannot name it. The rule tests
// exported interfaces only.
type watcher interface {
	Watch() any
}

// Watch runs the unexported interface, so package b keeps a use of it.
func Watch(w watcher) any { return w.Watch() }

// Number is a constraint: it embeds comparable as well as a method, so
// it is not a method set and no value can implement it. types.Implements
// answers true against the method set alone, so the analyzer must drop
// such an interface before it asks.
type Number interface {
	comparable
	Load() any
}

// Sum runs the constraint, so package b keeps a use of it.
func Sum[T Number](values []T) {}
