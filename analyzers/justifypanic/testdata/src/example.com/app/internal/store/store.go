// Package store holds the same two shapes as the suite package beside
// it. No pattern of the setting names this package, so it stays
// production code and both calls report.
package store

import "os"

// Item is the value the store holds.
type Item struct{ Name string }

// Build returns an item. A bad name stops the process of the caller.
// Rule G11 asks the author to state that decision.
func Build(name string) Item {
	if name == "" {
		panic("the store needs a name") // want "panic in library code has no justification comment"
	}

	return Item{Name: name}
}

// Halt ends the process of the caller.
func Halt(code int) {
	os.Exit(code) // want "os.Exit in library code has no justification comment"
}
