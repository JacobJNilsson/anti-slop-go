// Package suitereflect is the fixture of rule G07 for the
// test-packages setting. The setting names the package, so the
// DeepEqual allowance of the rule holds. No line of this file carries
// an expectation.
package suitereflect

import "reflect"

// Equal compares two values of a composite type, which Go gives a test
// no other way to do.
func Equal(got, want []string) bool {
	return reflect.DeepEqual(got, want)
}
