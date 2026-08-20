// Package other is a package that a test of package a imports. A test
// that assigns to a function variable of this package rewires code of
// another package. The owner of that code cannot see the change, so
// the rule reports it as well.
package other

import "time"

// Send is a package-level function variable of this package. No test
// file of package other declares it, so an assignment to it in any
// test file is a report.
var Send = func(message string) error { return nil }

// Hook is a defined function type. The underlying type carries the
// signature.
type Hook func()

// After holds a defined function type.
var After Hook = func() {}

// Config offers a function field. A local value of the type is a seam
// of the design, so a test that fills it uses the design.
type Config struct {
	Now func() time.Time
}

// Clock is an interface that a package-level variable holds.
type Clock interface {
	Now() time.Time
}

// Default holds the clock of this package. An interface variable
// carries behaviour, so a test that assigns to it rewires the package
// for every other test of the binary.
var Default Clock = systemClock{}

// systemClock is the value that Default holds.
type systemClock struct{}

// Now answers with the zero time, which keeps the fixture simple.
func (systemClock) Now() time.Time { return time.Time{} }
