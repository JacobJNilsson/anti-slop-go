// Package main holds the fixtures of the main and init exemption. The
// rule reads the function and not the package: main and init are the
// program itself, and every other function of this package is library
// code.
package main

import "os"

// The function that the program starts with stops the program. Nobody
// stands behind it, so it needs no justification.
func main() {
	if len(os.Args) == 1 {
		panic("the program needs one argument")
	}

	run(func() { os.Exit(2) })
}

// An init function runs before main, in any package, and 002 exempts
// it there too.
func init() {
	if os.Getenv("HOME") == "" {
		os.Exit(1)
	}
}

func run(f func()) { f() }

// A function beside main is library code. Another package cannot import
// it today, but the reader of this file still stands behind the call.
func helper(code int) {
	os.Exit(code) // want "os.Exit in library code has no justification comment"
}

func helperJustified() {
	// PANICS: the caller checked every flag before this line.
	panic("the flag table holds no entry")
}

var _ = helper
var _ = helperJustified
