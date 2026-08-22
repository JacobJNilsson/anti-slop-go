// Package b holds a package-level value. The root of a chain that
// starts with the name of a package is no variable, so the rule reads
// no base there.
package b

// Item is the type of the package-level value.
type Item struct {
	Name  string
	Count int
}

// Global is the value a test of another package reads.
var Global Item
