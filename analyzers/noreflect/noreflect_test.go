package noreflect_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/JacobJNilsson/anti-slop-go/analyzers/noreflect"
)

// allowed is the import path of the fixture package that a project
// allows. Package "a" is the package that no pattern allows.
const allowed = "example.com/app/internal/codec"

// TestAnalyzer runs the rejected and the accepted fixtures of rule G07
// with no allow pattern. Package "a" holds a production file, a
// generated file, and test files. Package "prod" holds production files
// only, and no test file of it imports reflect. Every rejected form
// carries a want comment, so a file with no want comment must stay
// clean.
func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), noreflect.Analyzer, "a", "prod")
}

// The allowlist is the only escape the rule offers, so a matching
// pattern must clear the whole package, test files included.
func TestNewAllowsAMatchingPackage(t *testing.T) {
	patterns := []string{
		"example.com/app/...",
		".../internal/codec",
		// The package above a subtree reads through the wildcards of
		// the pattern, so this one names the package itself.
		"example.com/*/internal/codec/...",
		allowed,
	}
	for _, pattern := range patterns {
		t.Run(pattern, func(t *testing.T) {
			analysistest.Run(t, analysistest.TestData(), noreflect.New([]string{pattern}, nil), allowed)
		})
	}
}

// A pattern that names another package changes nothing. The reports of
// package "a" stand, which is the test that the allowlist is anchored
// and not a substring test.
func TestNewKeepsReportsOutsideTheAllowlist(t *testing.T) {
	patterns := []string{
		"example.com/app/...",
		// A subtree pattern allows the packages under "a" and never "a"
		// itself.
		"a/internal/...",
		// A pattern matches the whole path, so a longer name is another
		// package.
		"ab",
	}
	for _, pattern := range patterns {
		t.Run(pattern, func(t *testing.T) {
			analysistest.Run(t, analysistest.TestData(), noreflect.New([]string{pattern}, nil), "a")
		})
	}
}

// The two fixture packages of the test-packages setting. Both hold one
// DeepEqual call, and the suite package holds a second file that
// reaches beyond DeepEqual.
const (
	suitePackage = "example.com/app/internal/suite"
	storePackage = "example.com/app/internal/store"
)

// A package that the setting names serves tests, so the DeepEqual
// allowance holds for its files. The suite fixture carries a want
// comment at the use beyond DeepEqual and nowhere else. The store
// fixture beside it holds the same DeepEqual call, and its import
// reports.
func TestNewReadsATestPackageAsTestCode(t *testing.T) {
	patterns := []string{suitePackage}
	analysistest.Run(t, analysistest.TestData(), noreflect.New(nil, patterns), suitePackage, storePackage)
}

// The testpackages flag is the configuration surface of that setting
// outside golangci-lint, and it holds the same promise about one
// instance.
func TestTestPackagesFlagConfiguresOneInstance(t *testing.T) {
	configured := noreflect.New(nil, nil)
	if err := configured.Flags.Set("testpackages", ".../internal/suite"); err != nil {
		t.Fatalf("the testpackages flag rejected a pattern: %v", err)
	}
	analysistest.Run(t, analysistest.TestData(), configured, suitePackage, storePackage)

	if got := noreflect.Analyzer.Flags.Lookup("testpackages").Value.String(); got != "" {
		t.Errorf("the analyzer of the module holds the patterns %q of another instance", got)
	}
}

// The flag is the configuration surface of cmd/antislop and of
// go vet -vettool. It writes into the analyzer instance it sits on, so
// two instances never share a setting.
func TestAllowFlagConfiguresOneInstance(t *testing.T) {
	configured := noreflect.New(nil, nil)
	if err := configured.Flags.Set("allow", "example.com/app/..."); err != nil {
		t.Fatalf("the allow flag rejected a pattern: %v", err)
	}
	analysistest.Run(t, analysistest.TestData(), configured, allowed)

	// Another instance holds its own patterns. The plugin builds one
	// instance for each run for this reason: it never writes to the
	// analyzer value that every consumer of the module shares.
	if got := noreflect.New(nil, nil).Flags.Lookup("allow").Value.String(); got != "" {
		t.Errorf("a new analyzer already holds the patterns %q of another instance", got)
	}
	if got := noreflect.Analyzer.Flags.Lookup("allow").Value.String(); got != "" {
		t.Errorf("the analyzer of the module holds the patterns %q of another instance", got)
	}
}

// The flag takes a comma-separated list, and it appends on every
// occurrence, so a repeated flag adds patterns.
func TestAllowFlagAppendsAndSplits(t *testing.T) {
	a := noreflect.New(nil, nil)
	if err := a.Flags.Set("allow", "other.com/lib,.../internal/codec"); err != nil {
		t.Fatalf("the allow flag rejected a list: %v", err)
	}
	if err := a.Flags.Set("allow", "other.com/tool"); err != nil {
		t.Fatalf("the allow flag rejected a second occurrence: %v", err)
	}

	// The pattern of the first occurrence must survive the second one.
	analysistest.Run(t, analysistest.TestData(), a, allowed)

	const want = "other.com/lib,.../internal/codec,other.com/tool"
	if got := a.Flags.Lookup("allow").Value.String(); got != want {
		t.Errorf("the flag reads back as %q, want %q", got, want)
	}
}

// The package-level analyzer carries both flags, because cmd/antislop
// registers the analyzers of the module through antislop.Analyzers().
func TestAnalyzerCarriesTheFlags(t *testing.T) {
	for _, flag := range []string{"allow", "testpackages"} {
		if noreflect.Analyzer.Flags.Lookup(flag) == nil {
			t.Errorf("the analyzer of the module registers no %s flag", flag)
		}
	}
}
