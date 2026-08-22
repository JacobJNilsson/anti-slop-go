package a

import "testing"

// logger carries the name of a failure method and no relation to the
// testing package.
type logger struct{}

// Errorf carries the name that the rule reads on a testing value.
func (logger) Errorf(format string, args ...any) {}

// The standard library shape: an if statement that compares, and a call
// that fails the test.
func TestStdlibShape(t *testing.T) {
	got := build()
	if got.Name != "boot" { // want `assertions name 2 fields of got one at a time`
		t.Errorf("Name = %q", got.Name)
	}
	if got.Count != 3 {
		t.Fatalf("Count = %d", got.Count)
	}
}

// A condition that names two fields of one base is one compare already,
// so it counts for nothing.
func TestCombinedCondition(t *testing.T) {
	got := build()
	want := build()
	if got.Name != want.Name || got.Count != want.Count {
		t.Errorf("got %v, want %v", got, want)
	}
}

// A comparison with no failure call is no assertion site. The rule
// reads the body for a call that fails the test.
func TestNoFailureCall(t *testing.T) {
	got := build()
	total := 0
	if got.Name != "boot" {
		total++
	}
	if got.Count != 3 {
		t.Helper()
		total++
	}
	if total != 2 {
		t.Errorf("total = %d", total)
	}
}

// A statement that is no call, and a call that names no method, both
// stand in the body of the compare. The rule reads past them to the
// call that fails the test.
func TestNonCallStatement(t *testing.T) {
	got := build()
	done := make(chan int, 1)
	done <- 1
	if got.Name != "boot" {
		<-done
		build()
		t.Errorf("Name = %q", got.Name)
	}
}

// A comparison that is no equality test states a range, and the rule
// reads none. One field is left, so the function stays clean.
func TestOrderingCondition(t *testing.T) {
	got := build()
	if got.Count < 3 {
		t.Errorf("Count = %d", got.Count)
	}
	if got.Name != "boot" {
		t.Errorf("Name = %q", got.Name)
	}
}

// A method of the project can carry the name of the equality family.
// The rule reads the module that declares the type of the receiver. A
// type of this project names this module, and an interface written in
// place names none, so neither call counts.
func TestProjectEqualMethod(t *testing.T) {
	got := build()
	var declared Matcher
	var written interface{ Equal(other any) bool } = declared
	if declared.Equal(got.Name) || written.Equal(got.Count) {
		t.Error("the matcher accepts no field of the value")
	}
}

// The failure method must sit on a value of the testing package. A type
// of the project carries no such meaning, whatever the name of its
// method.
func TestNonTestingReceiver(t *testing.T) {
	got := build()
	var log logger
	if got.Name != "boot" {
		log.Errorf("Name = %q", got.Name)
	}
	if got.Count != 3 {
		log.Errorf("Count = %d", got.Count)
	}
	if got.Inner.Label != "run" {
		t.Errorf("Label = %q", got.Inner.Label)
	}
}
