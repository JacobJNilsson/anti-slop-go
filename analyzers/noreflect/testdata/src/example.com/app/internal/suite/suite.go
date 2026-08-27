// Package suite holds the shared test suite of the fixture. No file of
// the package carries a name that ends in _test.go, and the package
// serves the tests of the project alone. The test-packages setting
// names it, so the DeepEqual allowance of the rule holds here.
package suite

import "reflect"

// Item is the value the suite compares.
type Item struct {
	Name  string
	Cells []int
}

// Equal compares two items. Go gives a test no other way to compare
// two values of a composite type, so this call needs no allowance of
// its own.
func Equal(got, want Item) bool {
	return reflect.DeepEqual(got, want)
}
