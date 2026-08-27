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

// The cost gate reads the number of cmpopts.IgnoreFields names that one
// comparison of the whole value needs. Package "gate" holds the classes
// the gate decides: the group above the setting, the group at an
// anchor, the roundtrip against a value of the same type, and the
// lookalike of that roundtrip.
func TestCostGate(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), fullstructcomp.Analyzer, "gate")
}

// The setting states the number of distinct fields a report needs. The
// min3 fixture holds one function of two fields and one of three, so
// one run reads both directions of the setting.
func TestNewTakesTheMinimum(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), fullstructcomp.New(3, fullstructcomp.DefaultMaxIgnore, nil), "min3")
}

// A setting of one reports every spot check, which 002 rejects for a
// project. The message must still read as a sentence, so the singular
// form of the count sits here.
func TestNewTakesTheMinimumOfOne(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), fullstructcomp.New(1, fullstructcomp.DefaultMaxIgnore, nil), "min1")
}

// A maxignore setting of zero reports a group whose fix carries no
// ignore name at all. The fixture holds one such group and one group
// that needs a single name.
func TestNewTakesAMaxIgnoreOfZero(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), fullstructcomp.New(fullstructcomp.DefaultMin, 0, nil), "ignore0")
}

// A high maxignore setting gives the report volume of the rule before
// the gate. The fixture holds a group whose fix needs six names, which
// the default rejects.
func TestNewTakesAHighMaxIgnore(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), fullstructcomp.New(fullstructcomp.DefaultMin, 20, nil), "ignorehigh")
}

// The default is two fields, which 002 records with the evidence for
// it. The constant is exported, because the golangci-lint plugin reads
// it for a run whose settings name no number.
func TestDefaultMin(t *testing.T) {
	if fullstructcomp.DefaultMin != 2 {
		t.Errorf("DefaultMin = %d; 002 states two fields", fullstructcomp.DefaultMin)
	}
}

// The default cost is five ignore names. 002 records the measurement:
// most groups of the corpora need five names or fewer, and every group
// that a reader judged good sits at five or below.
func TestDefaultMaxIgnore(t *testing.T) {
	if fullstructcomp.DefaultMaxIgnore != 5 {
		t.Errorf("DefaultMaxIgnore = %d; 002 states five names", fullstructcomp.DefaultMaxIgnore)
	}
}

// The flag is the configuration surface of cmd/antislop and of
// go vet -vettool. It writes into the analyzer instance it sits on, so
// two instances never share a setting.
func TestMinFlagConfiguresOneInstance(t *testing.T) {
	configured := fullstructcomp.New(fullstructcomp.DefaultMin, fullstructcomp.DefaultMaxIgnore, nil)
	if err := configured.Flags.Set("min", "3"); err != nil {
		t.Fatalf("the min flag rejected a number: %v", err)
	}
	analysistest.Run(t, analysistest.TestData(), configured, "min3")

	// Another instance holds its own setting. The plugin builds one
	// instance for each run for this reason: it never writes to the
	// analyzer value that every consumer of the module shares.
	clean := fullstructcomp.New(fullstructcomp.DefaultMin, fullstructcomp.DefaultMaxIgnore, nil)
	if got := clean.Flags.Lookup("min").Value.String(); got != "2" {
		t.Errorf("a new analyzer reads the setting %q of another instance", got)
	}
	if got := fullstructcomp.Analyzer.Flags.Lookup("min").Value.String(); got != "2" {
		t.Errorf("the analyzer of the module reads the setting %q of another instance", got)
	}
}

// The maxignore flag carries the cost gate on the same path, and it
// holds the same promise about one instance.
func TestMaxIgnoreFlagConfiguresOneInstance(t *testing.T) {
	configured := fullstructcomp.New(fullstructcomp.DefaultMin, fullstructcomp.DefaultMaxIgnore, nil)
	if err := configured.Flags.Set("maxignore", "20"); err != nil {
		t.Fatalf("the maxignore flag rejected a number: %v", err)
	}
	analysistest.Run(t, analysistest.TestData(), configured, "ignorehigh")

	if got := fullstructcomp.Analyzer.Flags.Lookup("maxignore").Value.String(); got != "5" {
		t.Errorf("the analyzer of the module reads the setting %q of another instance", got)
	}
}

// The two fixture packages of the test-packages setting. Both hold one
// helper that asserts two fields of one value.
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
	configured := fullstructcomp.New(fullstructcomp.DefaultMin, fullstructcomp.DefaultMaxIgnore, patterns)
	analysistest.Run(t, analysistest.TestData(), configured, suitePackage, storePackage)
}

// The testpackages flag carries that setting outside golangci-lint,
// and it holds the same promise about one instance.
func TestTestPackagesFlagConfiguresOneInstance(t *testing.T) {
	configured := fullstructcomp.New(fullstructcomp.DefaultMin, fullstructcomp.DefaultMaxIgnore, nil)
	if err := configured.Flags.Set("testpackages", ".../internal/suite"); err != nil {
		t.Fatalf("the testpackages flag rejected a pattern: %v", err)
	}
	analysistest.Run(t, analysistest.TestData(), configured, suitePackage, storePackage)

	if got := fullstructcomp.Analyzer.Flags.Lookup("testpackages").Value.String(); got != "" {
		t.Errorf("the analyzer of the module holds the patterns %q of another instance", got)
	}
}

// The package-level analyzer carries every flag, because cmd/antislop
// registers the analyzers of the module through antislop.Analyzers().
func TestAnalyzerCarriesTheFlags(t *testing.T) {
	for _, flag := range []string{"min", "maxignore", "testpackages"} {
		if fullstructcomp.Analyzer.Flags.Lookup(flag) == nil {
			t.Errorf("the analyzer of the module registers no %s flag", flag)
		}
	}
}
