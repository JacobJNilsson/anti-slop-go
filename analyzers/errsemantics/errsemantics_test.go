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
	analysistest.Run(t, analysistest.TestData(), errsemantics.New(true), "equality")
}

// The flag is the configuration surface of cmd/antislop and of
// go vet -vettool. It writes into the analyzer instance it sits on, so
// two instances never share a setting.
func TestEqualityFlagConfiguresOneInstance(t *testing.T) {
	configured := errsemantics.New(false)
	if err := configured.Flags.Set("equality", "true"); err != nil {
		t.Fatalf("the equality flag rejected a value: %v", err)
	}
	analysistest.Run(t, analysistest.TestData(), configured, "equality")

	// Another instance holds its own setting. The plugin builds one
	// instance for each run for this reason: it never writes to the
	// analyzer value that every consumer of the module shares.
	if got := errsemantics.New(false).Flags.Lookup("equality").Value.String(); got != "false" {
		t.Errorf("a new analyzer already reads the setting %q of another instance", got)
	}
	if got := errsemantics.Analyzer.Flags.Lookup("equality").Value.String(); got != "false" {
		t.Errorf("the analyzer of the module reads the setting %q of another instance", got)
	}
}

// The package-level analyzer carries the flag, because cmd/antislop
// registers the analyzers of the module through antislop.Analyzers().
func TestAnalyzerCarriesTheFlag(t *testing.T) {
	if errsemantics.Analyzer.Flags.Lookup("equality") == nil {
		t.Error("the analyzer of the module registers no equality flag")
	}
}
