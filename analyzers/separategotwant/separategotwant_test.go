package separategotwant_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/JacobJNilsson/anti-slop-go/analyzers/separategotwant"
)

// TestAnalyzer runs the rejected and the accepted fixtures of rule G14.
// Package "a" holds a production file, a generated test file, a file of
// lookalike packages, and the test files that carry the assertions.
// Every rejected form carries a want comment, so a line with no want
// comment must stay clean.
func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), separategotwant.Analyzer, "a")
}

// The two fixture packages of the test-packages setting. Both hold one
// helper that asserts on the result of a call that takes the testing
// value.
const (
	suitePackage = "example.com/app/internal/suite"
	storePackage = "example.com/app/internal/store"
)

// A package that the setting names serves tests, so the rule reads its
// files. The rule reads no such package today, and the setting
// therefore adds reports. The suite fixture carries the want comment.
// The store fixture beside it holds the same helper and stays silent,
// because it is production code.
func TestNewReadsATestPackageAsTestCode(t *testing.T) {
	configured := separategotwant.New([]string{suitePackage})
	analysistest.Run(t, analysistest.TestData(), configured, suitePackage, storePackage)
}

// The testpackages flag carries that setting outside golangci-lint, and
// it writes into the analyzer instance it sits on. Two instances never
// share a setting, which is why the plugin builds one instance for each
// run.
func TestTestPackagesFlagConfiguresOneInstance(t *testing.T) {
	configured := separategotwant.New(nil)
	if err := configured.Flags.Set("testpackages", ".../internal/suite"); err != nil {
		t.Fatalf("the testpackages flag rejected a pattern: %v", err)
	}
	analysistest.Run(t, analysistest.TestData(), configured, suitePackage, storePackage)

	if got := separategotwant.Analyzer.Flags.Lookup("testpackages").Value.String(); got != "" {
		t.Errorf("the analyzer of the module holds the patterns %q of another instance", got)
	}
}

// The package-level analyzer carries the flag, because cmd/antislop
// registers the analyzers of the module through antislop.Analyzers().
func TestAnalyzerCarriesTheFlag(t *testing.T) {
	if separategotwant.Analyzer.Flags.Lookup("testpackages") == nil {
		t.Error("the analyzer of the module registers no testpackages flag")
	}
}
