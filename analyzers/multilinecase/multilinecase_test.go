package multilinecase_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/JacobJNilsson/anti-slop-go/analyzers/multilinecase"
)

// TestAnalyzer runs the rejected and the accepted fixtures of rule G15,
// and it applies each suggested fix. Package "a" holds the table forms,
// a production file, a generated file, and a case with a comment.
// analysistest runs gofmt over the fixed text and over each golden file
// before it compares them, so the value alignment of a golden file is
// no part of the check. The golden file of the case with a comment
// equals its fixture, which states that the rule offers no fix there. Every rejected case
// carries a want comment, so a line with no want comment must stay
// clean.
func TestAnalyzer(t *testing.T) {
	analysistest.RunWithSuggestedFixes(t, analysistest.TestData(), multilinecase.Analyzer, "a")
}

// The two fixture packages of the test-packages setting. Both hold one
// table that a loop reads, and neither carries a _test.go file name.
const (
	suitePackage = "example.com/app/internal/suite"
	storePackage = "example.com/app/internal/store"
)

// The min setting moves the number of keyed fields that a report needs.
// The fixture holds a case of three fields and a case of four.
func TestNewReportsAtTheMinimum(t *testing.T) {
	configured := multilinecase.New(4, nil)
	analysistest.Run(t, analysistest.TestData(), configured, "minfour")
}

// The min flag carries that setting outside golangci-lint, and it
// writes into the analyzer instance it sits on.
func TestMinFlagConfiguresOneInstance(t *testing.T) {
	configured := multilinecase.New(multilinecase.DefaultMin, nil)
	if err := configured.Flags.Set("min", "4"); err != nil {
		t.Fatalf("the min flag rejected a number: %v", err)
	}
	analysistest.Run(t, analysistest.TestData(), configured, "minfour")

	if got := multilinecase.Analyzer.Flags.Lookup("min").Value.String(); got != "3" {
		t.Errorf("the analyzer of the module holds min %q of another instance", got)
	}
}

// A package that the setting names serves tests, so the rule reads its
// files. The store fixture beside it holds the same table and stays
// silent, because it is production code.
func TestNewReadsATestPackageAsTestCode(t *testing.T) {
	configured := multilinecase.New(multilinecase.DefaultMin, []string{suitePackage})
	analysistest.Run(t, analysistest.TestData(), configured, suitePackage, storePackage)
}

// The testpackages flag carries that setting outside golangci-lint, and
// it writes into the analyzer instance it sits on.
func TestTestPackagesFlagConfiguresOneInstance(t *testing.T) {
	configured := multilinecase.New(multilinecase.DefaultMin, nil)
	if err := configured.Flags.Set("testpackages", ".../internal/suite"); err != nil {
		t.Fatalf("the testpackages flag rejected a pattern: %v", err)
	}
	analysistest.Run(t, analysistest.TestData(), configured, suitePackage, storePackage)

	if got := multilinecase.Analyzer.Flags.Lookup("testpackages").Value.String(); got != "" {
		t.Errorf("the analyzer of the module holds the patterns %q of another instance", got)
	}
}

// The package-level analyzer carries the flags, because cmd/antislop
// registers the analyzers of the module through antislop.Analyzers().
func TestAnalyzerCarriesTheFlags(t *testing.T) {
	for _, name := range []string{"min", "testpackages"} {
		if multilinecase.Analyzer.Flags.Lookup(name) == nil {
			t.Errorf("the analyzer of the module registers no %s flag", name)
		}
	}
}
