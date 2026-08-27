// Package store holds the same DeepEqual call as the suite package
// beside it. No pattern of the setting names this package, so it stays
// production code and the import reports.
package store

import "reflect" // want `this package imports reflect`

// Item is the value the store holds.
type Item struct {
	Name  string
	Cells []int
}

// Equal compares two items in production code.
func Equal(got, want Item) bool {
	return reflect.DeepEqual(got, want)
}
