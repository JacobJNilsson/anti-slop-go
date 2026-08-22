package antislop

import (
	"testing"

	"github.com/JacobJNilsson/anti-slop-go/analyzers/nointerfacereturn"
)

func TestAnalyzersReturnsRegisteredRules(t *testing.T) {
	got := Analyzers()
	if got == nil {
		t.Fatal("Analyzers() returned nil; consumers range over the result")
	}
	// The spec (docs/spec/002-rules.md) defines the rules G01-G13. Grow
	// this expectation with each implemented analyzer.
	if len(got) != 13 {
		t.Fatalf("Analyzers() returned %d analyzers; update this test with the rule set", len(got))
	}
}

// The registry holds every rule, an opt-in rule included. One registry
// serves the three consumption paths. The golangci-lint plugin applies
// the opt-in severity of 002, because it reads a configuration file.
// The standalone paths read none, and multichecker gives them the
// -nointerfacereturn=false flag instead. 003 records the decision.
func TestAnalyzersHoldsTheOptInRules(t *testing.T) {
	for _, a := range Analyzers() {
		if a == nointerfacereturn.Analyzer {
			return
		}
	}
	t.Error("Analyzers() omits nointerfacereturn; the standalone paths would lose the rule")
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
