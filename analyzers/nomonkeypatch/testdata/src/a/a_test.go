package a

import (
	"testing"
	"time"

	_ "bou.ke/monkey" // want `imports the runtime patching library "bou.ke/monkey"`

	"github.com/agiledragon/gomonkey/v2" // want `imports the runtime patching library "github.com/agiledragon/gomonkey/v2"`

	"github.com/agiledragon/gomonkeytools"

	"other"
)

// stubClock offers a method value, which a test may bind to a name of
// its own.
type stubClock struct{}

// Now answers with a fixed time.
func (stubClock) Now() time.Time { return time.Time{} }

// TestPatchesTheCodeUnderTest rewires the variables that a non-test
// file of the package declares.
func TestPatchesTheCodeUnderTest(t *testing.T) {
	send = func(message string) error { return nil } // want `assigns to the package-level variable send; inject the dependency through a parameter or a field`
	Now = time.Now                                   // want `variable Now;`
	after = func() {}                                // want `variable after;`
	// Parentheses around the name change nothing.
	(send) = func(message string) error { return nil } // want `variable send;`
}

// TestPatchesAnotherPackage rewires the variables of an imported
// package.
func TestPatchesAnotherPackage(t *testing.T) {
	other.Send = func(message string) error { return nil } // want `variable other.Send;`
	other.After = func() {}                                // want `variable other.After;`
	// One statement can hold two targets, and each one gets its own
	// report.
	Now, other.Send = time.Now, func(message string) error { return nil } // want `variable Now;` `variable other.Send;`
}

// addressOf answers with the pointer that it takes.
func addressOf(clock *func() time.Time) *func() time.Time { return clock }

// stubStore is the value a test puts in an interface variable.
type stubStore struct{}

// Get answers with the empty string.
func (stubStore) Get(key string) string { return "" }

// TestPatchesAnInterfaceVariable rewires a package-level variable that
// holds an interface. The variable carries behaviour, exactly as a
// function variable does.
func TestPatchesAnInterfaceVariable(t *testing.T) {
	store = stubStore{}         // want `variable store;`
	other.Default = stubClock{} // want `variable other.Default;`
}

// TestKeepsTheInterfaceSeam uses an interface where the design offers
// it.
func TestKeepsTheInterfaceSeam(t *testing.T) {
	var local Store = stubStore{}
	local = stubStore{}
	_ = local

	server := Server{}
	server.Store = stubStore{}
}

// TestPatchesAContainer reaches the behaviour through a package-level
// container. The container is production state, so the entry it holds
// is production behaviour.
func TestPatchesAContainer(t *testing.T) {
	Options.Now = time.Now       // want `variable Options.Now;`
	Options.Inner.Now = time.Now // want `variable Options.Inner.Now;`
	Registry["boot"] = func() {} // want `variable Registry\["boot"\];`
	Chain[0] = func() {}         // want `variable Chain\[0\];`
	*Fallback = time.Now         // want `variable \*Fallback;`
	// The field holds no behaviour, so the rule leaves it alone.
	Options.Timeout = 1
}

// TestKeepsALocalContainer builds its own containers. The test owns
// every one of them.
func TestKeepsALocalContainer(t *testing.T) {
	config := Config{}
	config.Now = time.Now
	config.Inner.Now = time.Now

	registry := map[string]func(){}
	registry["boot"] = func() {}

	chain := []func(){nil}
	chain[0] = func() {}

	fallback := &config.Now
	*fallback = time.Now

	// The rule follows no call. A target under a call reaches no
	// variable that the rule can name, so the walk stops there.
	*addressOf(&config.Now) = time.Now

	// A test file declares this container, so the test files own it.
	TestRegistry["boot"] = func() {}
}

// TestPatchesInARangeClause assigns through the range clause, which
// is an assignment with another shape.
func TestPatchesInARangeClause(t *testing.T) {
	for _, Now = range []func() time.Time{time.Now} { // want `variable Now;`
	}
	// The clause declares its own variables, so the package-level name
	// stays untouched.
	for _, send := range []func(message string) error{nil} {
		_ = send
	}
	// A clause with one variable and no value keeps the rule quiet,
	// because an integer holds no behaviour.
	for count = range 3 {
	}
}

// TestRestoresWithCleanup puts the value back at the end of the test.
// The restore does not undo the design, so the report stands.
func TestRestoresWithCleanup(t *testing.T) {
	original := send
	t.Cleanup(func() {
		send = original // want `variable send;`
	})
}

// TestBindsAMethodValue shows that the rule reads the target of the
// assignment and not the value.
func TestBindsAMethodValue(t *testing.T) {
	var clock stubClock
	handler := clock.Now
	_ = handler
	Now = clock.Now // want `variable Now;`
}

// TestKeepsTheDesign uses the seams that the design offers.
func TestKeepsTheDesign(t *testing.T) {
	local := func(message string) error { return nil }
	local = func(message string) error { return nil }
	_ = local

	// A short declaration makes a new variable of the block. It hides
	// the package-level variable of the same name and changes nothing
	// in the package.
	send := func(message string) error { return nil }
	send = func(message string) error { return nil }
	_ = send

	server := Server{}
	server.Log = func(message string) {}

	config := other.Config{}
	config.Now = time.Now

	count = 1
	count += 2

	TestSink = func(message string) {}
	resetSink()

	sinks := map[string]func(){}
	sinks["reset"] = func() {}

	_ = func() {}

	gomonkeytools.Noop()
	gomonkey.ApplyFunc(nil, nil)
}

// patchedNow reaches a symbol of another package through the linker.
//
//go:linkname patchedNow time.Now // want `uses a //go:linkname directive; inject the dependency through a parameter or a field`
func patchedNow() time.Time

// go:linkname spacedIsNoDirective time.Now
//
//go:linknamex notADirective time.Now
var _ = patchedNow
