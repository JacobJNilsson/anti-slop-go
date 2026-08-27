package errsemantics_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/JacobJNilsson/anti-slop-go/analyzers/errsemantics"
)

// TestAnalyzer runs the rejected and the accepted fixtures of rule G13
// with the default setting. Package "a" holds a production file, a
// generated test file, and test files. Every rejected form carries a
// want comment, so a line with no want comment must stay clean. The
// equality forms sit there with no want comment, because the default
// setting leaves them alone.
func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), errsemantics.Analyzer, "a")
}

// The equality setting adds the equality forms and replaces nothing.
// Package "equality" holds the same shapes as the equality fixture of
// package "a", and the default forms beside them.
func TestNewReportsTheEqualityFormsWhenTheSettingIsOn(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), errsemantics.New(true, nil), "equality")
}

// The flag is the configuration surface of cmd/antislop and of
// go vet -vettool. It writes into the analyzer instance it sits on, so
// two instances never share a setting.
func TestEqualityFlagConfiguresOneInstance(t *testing.T) {
	configured := errsemantics.New(false, nil)
	if err := configured.Flags.Set("equality", "true"); err != nil {
		t.Fatalf("the equality flag rejected a value: %v", err)
	}
	analysistest.Run(t, analysistest.TestData(), configured, "equality")

	// Another instance holds its own setting. The plugin builds one
	// instance for each run for this reason: it never writes to the
	// analyzer value that every consumer of the module shares.
	if got := errsemantics.New(false, nil).Flags.Lookup("equality").Value.String(); got != "false" {
		t.Errorf("a new analyzer already reads the setting %q of another instance", got)
	}
	if got := errsemantics.Analyzer.Flags.Lookup("equality").Value.String(); got != "false" {
		t.Errorf("the analyzer of the module reads the setting %q of another instance", got)
	}
}

// The two fixture packages of the test-packages setting. Both hold one
// helper that reads the text of an error.
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
	patterns := []string{suitePackage}
	configured := errsemantics.New(false, patterns)
	analysistest.Run(t, analysistest.TestData(), configured, suitePackage, storePackage)
}

// The testpackages flag carries that setting outside golangci-lint,
// and it holds the same promise about one instance.
func TestTestPackagesFlagConfiguresOneInstance(t *testing.T) {
	configured := errsemantics.New(false, nil)
	if err := configured.Flags.Set("testpackages", ".../internal/suite"); err != nil {
		t.Fatalf("the testpackages flag rejected a pattern: %v", err)
	}
	analysistest.Run(t, analysistest.TestData(), configured, suitePackage, storePackage)

	if got := errsemantics.Analyzer.Flags.Lookup("testpackages").Value.String(); got != "" {
		t.Errorf("the analyzer of the module holds the patterns %q of another instance", got)
	}
}

// The package-level analyzer carries both flags, because cmd/antislop
// registers the analyzers of the module through antislop.Analyzers().
func TestAnalyzerCarriesTheFlags(t *testing.T) {
	for _, flag := range []string{"equality", "testpackages"} {
		if errsemantics.Analyzer.Flags.Lookup(flag) == nil {
			t.Errorf("the analyzer of the module registers no %s flag", flag)
		}
	}
}
