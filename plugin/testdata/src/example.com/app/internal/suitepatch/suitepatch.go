// Package suitepatch is the fixture of rule G08 for the test-packages
// setting. The setting names the package, so the rule reads this file
// as test code and reports the assignment below.
package suitepatch

import "example.com/app/internal/prod"

// Patch rewires production code. The owner of that code cannot see the
// change.
func Patch() {
	prod.Send = func(message string) error { return nil } // want `test assigns to the package-level variable prod.Send`
}
