package antislop

import "testing"

func TestAnalyzersReturnsRegisteredRules(t *testing.T) {
	got := Analyzers()
	if got == nil {
		t.Fatal("Analyzers() returned nil; consumers range over the result")
	}
	// The spec (docs/spec/002-rules.md) defines rules G01-G11. Grow this
	// expectation with each implemented analyzer.
	if len(got) != 1 {
		t.Fatalf("Analyzers() returned %d analyzers; update this test with the rule set", len(got))
	}
}
