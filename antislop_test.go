package antislop

import "testing"

func TestAnalyzersReturnsRegisteredRules(t *testing.T) {
	got := Analyzers()
	if got == nil {
		t.Fatal("Analyzers() returned nil; consumers range over the result")
	}
	// The spec (docs/spec/002-rules.md) defines rules G01-G11. Grow this
	// expectation with each implemented analyzer.
	if len(got) != 9 {
		t.Fatalf("Analyzers() returned %d analyzers; update this test with the rule set", len(got))
	}
}

// Every caller gets its own slice. A consumer edits the result to put a
// configured analyzer in the place of a shared one, which the
// golangci-lint plugin does for each run. A shared backing array would
// carry that edit into the next caller.
func TestAnalyzersReturnsAFreshSlice(t *testing.T) {
	first := Analyzers()
	second := Analyzers()
	if len(first) == 0 || len(second) == 0 {
		t.Fatal("Analyzers() returned an empty rule set")
	}

	first[0] = nil
	if second[0] == nil {
		t.Error("Analyzers() shares its backing array between calls; one caller can edit the set of another")
	}
}
