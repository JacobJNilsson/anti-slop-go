package nomonkeypatch_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/JacobJNilsson/anti-slop-go/analyzers/nomonkeypatch"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), nomonkeypatch.Analyzer, "a")
}

// The two fixture packages of the test-packages setting. Both assign
// to one package-level function variable of package other.
const (
	suitePackage = "example.com/app/internal/suite"
	storePackage = "example.com/app/internal/store"
)

// A package that the setting names is test code, so the rule reads its
// files. The suite fixture carries a want comment at the assignment to
// production code. It carries none at the assignment to its own
// variable, because the suite owns what it declares. The store fixture beside it holds the
// same assignment and reports nothing, because it is production code.
func TestNewReadsATestPackageAsTestCode(t *testing.T) {
	patterns := []string{suitePackage}
	analysistest.Run(t, analysistest.TestData(), nomonkeypatch.New(patterns), suitePackage, storePackage)
}

// The flag is the configuration surface of cmd/antislop and of
// go vet -vettool. It writes into the analyzer instance it sits on, so
// two instances never share a setting.
func TestTestPackagesFlagConfiguresOneInstance(t *testing.T) {
	configured := nomonkeypatch.New(nil)
	if err := configured.Flags.Set("testpackages", ".../internal/suite"); err != nil {
		t.Fatalf("the testpackages flag rejected a pattern: %v", err)
	}
	analysistest.Run(t, analysistest.TestData(), configured, suitePackage, storePackage)

	// Another instance holds its own patterns. The plugin builds one
	// instance for each run for this reason: it never writes to the
	// analyzer value that every consumer of the module shares.
	if got := nomonkeypatch.Analyzer.Flags.Lookup("testpackages").Value.String(); got != "" {
		t.Errorf("the analyzer of the module holds the patterns %q of another instance", got)
	}
}

// The package-level analyzer carries the flag, because cmd/antislop
// registers the analyzers of the module through antislop.Analyzers().
func TestAnalyzerCarriesTheFlag(t *testing.T) {
	if nomonkeypatch.Analyzer.Flags.Lookup("testpackages") == nil {
		t.Error("the analyzer of the module registers no testpackages flag")
	}
}
