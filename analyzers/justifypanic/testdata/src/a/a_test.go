package a

import (
	"os"
	"testing"
)

// A test file gets the whole exemption. A panic there stops one test
// binary, and the author of the test reads the stack trace at once. No
// call of this file carries a want comment, so a diagnostic here fails
// the suite.

func TestParse(t *testing.T) {
	defer func() { _ = recover() }()
	panic("the fixture holds no config")
}

func mustParse(t *testing.T, s string) Config {
	t.Helper()
	c, err := parse(s)
	if err != nil {
		panic(err)
	}

	return c
}

func TestMustParse(t *testing.T) {
	if got := mustParse(t, "name"); got.Name != "name" {
		t.Errorf("mustParse gave %q; want %q", got.Name, "name")
	}
}

func TestMainExits(t *testing.T) {
	if t == nil {
		os.Exit(1)
	}
}
