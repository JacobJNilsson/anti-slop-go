package noadhoctypeswitch_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/JacobJNilsson/anti-slop-go/analyzers/noadhoctypeswitch"
)

// boundary is the import path of the fixture package that a project
// marks as a decode boundary. Package "a" is the package that no
// boundary pattern names.
const boundary = "example.com/app/internal/ingest"

// TestAnalyzer runs the rejected and the accepted fixtures of rule G06
// with no boundary pattern. Package "a" holds a production file, a
// generated file, and a test file. Every rejected form carries a want
// comment, so a form with no want comment must stay clean.
func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), noadhoctypeswitch.Analyzer, "a")
}

// A boundary pattern clears the whole package, its external test
// package included.
func TestNewAllowsABoundaryPackage(t *testing.T) {
	patterns := []string{
		"example.com/app/...",
		".../internal/ingest",
		// The package above a subtree reads through the wildcards of the
		// pattern, so this one names the package itself.
		"example.com/*/internal/ingest/...",
		boundary,
	}
	for _, pattern := range patterns {
		t.Run(pattern, func(t *testing.T) {
			analysistest.Run(t, analysistest.TestData(), noadhoctypeswitch.New([]string{pattern}), boundary)
		})
	}
}

// A pattern that names another package changes nothing. The reports of
// package "a" stand, which is the test that the boundary list is
// anchored and not a substring test.
func TestNewKeepsReportsOutsideTheBoundary(t *testing.T) {
	patterns := []string{
		"example.com/app/...",
		// A subtree pattern names the packages under "a" and never "a"
		// itself.
		"a/internal/...",
		// A pattern matches the whole path, so a longer name is another
		// package.
		"ab",
	}
	for _, pattern := range patterns {
		t.Run(pattern, func(t *testing.T) {
			analysistest.Run(t, analysistest.TestData(), noadhoctypeswitch.New([]string{pattern}), "a")
		})
	}
}

// The flag is the configuration surface of cmd/antislop and of
// go vet -vettool. It writes into the analyzer instance it sits on, so
// two instances never share a setting.
func TestBoundaryFlagConfiguresOneInstance(t *testing.T) {
	configured := noadhoctypeswitch.New(nil)
	if err := configured.Flags.Set("boundary", "example.com/app/..."); err != nil {
		t.Fatalf("the boundary flag rejected a pattern: %v", err)
	}
	analysistest.Run(t, analysistest.TestData(), configured, boundary)

	// Another instance holds its own patterns. The plugin builds one
	// instance for each run for this reason: it never writes to the
	// analyzer value that every consumer of the module shares.
	if got := noadhoctypeswitch.New(nil).Flags.Lookup("boundary").Value.String(); got != "" {
		t.Errorf("a new analyzer already holds the patterns %q of another instance", got)
	}
	if got := noadhoctypeswitch.Analyzer.Flags.Lookup("boundary").Value.String(); got != "" {
		t.Errorf("the analyzer of the module holds the patterns %q of another instance", got)
	}
}

// The flag takes a comma-separated list, and it appends on every
// occurrence, so a repeated flag adds patterns.
func TestBoundaryFlagAppendsAndSplits(t *testing.T) {
	a := noadhoctypeswitch.New(nil)
	if err := a.Flags.Set("boundary", "other.com/lib,.../internal/ingest"); err != nil {
		t.Fatalf("the boundary flag rejected a list: %v", err)
	}
	if err := a.Flags.Set("boundary", "other.com/tool"); err != nil {
		t.Fatalf("the boundary flag rejected a second occurrence: %v", err)
	}

	// The pattern of the first occurrence must survive the second one.
	analysistest.Run(t, analysistest.TestData(), a, boundary)

	const want = "other.com/lib,.../internal/ingest,other.com/tool"
	if got := a.Flags.Lookup("boundary").Value.String(); got != want {
		t.Errorf("the flag reads back as %q, want %q", got, want)
	}
}

// The package-level analyzer carries the flag, because cmd/antislop
// registers the analyzers of the module through antislop.Analyzers().
func TestAnalyzerCarriesTheFlag(t *testing.T) {
	if noadhoctypeswitch.Analyzer.Flags.Lookup("boundary") == nil {
		t.Error("the analyzer of the module registers no boundary flag")
	}
}
