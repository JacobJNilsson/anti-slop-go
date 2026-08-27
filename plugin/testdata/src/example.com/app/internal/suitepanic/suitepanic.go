// Package suitepanic is the fixture of rule G11 for the test-packages
// setting. The package serves tests and holds no file whose name ends
// in _test.go. The setting names it, so the panic below needs no
// justification and this file carries no want comment.
package suitepanic

// Setup builds the fixture of a test. A bad name is a fault of the
// test, and the test binary stops.
func Setup(name string) string {
	if name == "" {
		panic("the suite needs a name")
	}

	return name
}
