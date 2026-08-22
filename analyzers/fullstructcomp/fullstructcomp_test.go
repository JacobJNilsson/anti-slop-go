package fullstructcomp_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/JacobJNilsson/anti-slop-go/analyzers/fullstructcomp"
)

// TestAnalyzer runs the rejected and the accepted fixtures of rule G12
// at the default setting. Package "a" holds a production file, a
// generated test file, test files of the package, and an external test
// package. Every rejected form carries a want comment, so a function
// with no want comment must stay clean.
func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), fullstructcomp.Analyzer, "a")
}

// The setting states the number of distinct fields a report needs. The
// min3 fixture holds one function of two fields and one of three, so
// one run reads both directions of the setting.
func TestNewTakesTheMinimum(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), fullstructcomp.New(3), "min3")
}

// A setting of one reports every spot check, which 002 rejects for a
// project. The message must still read as a sentence, so the singular
// form of the count sits here.
func TestNewTakesTheMinimumOfOne(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), fullstructcomp.New(1), "min1")
}

// The default is two fields, which 002 records with the evidence for
// it. The constant is exported, because the golangci-lint plugin reads
// it for a run whose settings name no number.
func TestDefaultMin(t *testing.T) {
	if fullstructcomp.DefaultMin != 2 {
		t.Errorf("DefaultMin = %d; 002 states two fields", fullstructcomp.DefaultMin)
	}
}

// The flag is the configuration surface of cmd/antislop and of
// go vet -vettool. It writes into the analyzer instance it sits on, so
// two instances never share a setting.
func TestMinFlagConfiguresOneInstance(t *testing.T) {
	configured := fullstructcomp.New(fullstructcomp.DefaultMin)
	if err := configured.Flags.Set("min", "3"); err != nil {
		t.Fatalf("the min flag rejected a number: %v", err)
	}
	analysistest.Run(t, analysistest.TestData(), configured, "min3")

	// Another instance holds its own setting. The plugin builds one
	// instance for each run for this reason: it never writes to the
	// analyzer value that every consumer of the module shares.
	if got := fullstructcomp.New(fullstructcomp.DefaultMin).Flags.Lookup("min").Value.String(); got != "2" {
		t.Errorf("a new analyzer reads the setting %q of another instance", got)
	}
	if got := fullstructcomp.Analyzer.Flags.Lookup("min").Value.String(); got != "2" {
		t.Errorf("the analyzer of the module reads the setting %q of another instance", got)
	}
}

// The package-level analyzer carries the flag, because cmd/antislop
// registers the analyzers of the module through antislop.Analyzers().
func TestAnalyzerCarriesTheFlag(t *testing.T) {
	if fullstructcomp.Analyzer.Flags.Lookup("min") == nil {
		t.Error("the analyzer of the module registers no min flag")
	}
}
