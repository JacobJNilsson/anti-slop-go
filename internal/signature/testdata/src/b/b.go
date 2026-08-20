// Package b holds the imported types that fixture package a uses.
package b

// Key is a constraint: it embeds comparable as well as a method, so it
// is not a method set and no value can implement it. types.Implements
// answers true against the method set alone, so the shared test must
// drop such an interface before it asks.
type Key interface {
	comparable
	Log(v any)
}

// Sum runs the constraint, so package b keeps a use of it.
func Sum[T Key](values []T) {}
