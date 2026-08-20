// Package a holds the production code of the fixture. The test files
// of the package try to rewire the code below. Rule G08 reports the
// attempt in the test file, never at the declaration.
package a

import (
	"time"
	// The //go:linkname directive at the end of this file needs the
	// unsafe import. The compiler rejects the directive without it.
	_ "unsafe"
)

// Now holds the clock of the package. It is a package-level function
// variable, and a non-test file declares it.
var Now = time.Now

// send is unexported. A test file of package a can name it, so the
// rule reads the unexported variable as well.
var send = func(message string) error { return nil }

// Hook is a defined function type. The underlying type carries the
// signature, so a variable of the type holds a function.
type Hook func()

// after holds a defined function type.
var after Hook = func() {}

// count holds no function, so an assignment to it is out of scope.
var count int

// Store is an interface that a package-level variable holds.
type Store interface {
	Get(key string) string
}

// store holds the store of the package. The variable carries
// behaviour, so a test that assigns to it rewires the package.
var store Store

// Server offers a function field and an interface field. A local
// server is a seam of the design, and a test that fills it uses the
// design.
type Server struct {
	Log   func(message string)
	Store Store
}

// Config holds behaviour in its fields. A local value of the type is
// a seam of the design. A package-level value of the type is
// production state.
type Config struct {
	Now     func() time.Time
	Inner   Inner
	Timeout int
}

// Inner sits one level down, so a target can hold two selectors.
type Inner struct {
	Now func() time.Time
}

// Options is a package-level container of behaviour.
var Options = Config{}

// Registry maps a name to behaviour, at package level.
var Registry = map[string]func(){}

// Chain holds behaviour in a package-level list.
var Chain = []func(){nil}

// Fallback points at the clock of the package.
var Fallback = &Now

// Reset rewires the variables of the package from production code.
// The rule reads test files only, so these assignments stand.
func Reset() {
	send = func(message string) error { return nil }
	Now = time.Now
	after = func() {}
	count = 0
}

// hidden is the symbol that the directive below exports under another
// name. The directive sits in a production file. Such a file is
// systems programming that the rule cannot judge, so the rule leaves
// the directive alone.
//
//go:linkname hidden a.exported
func hidden() {}
