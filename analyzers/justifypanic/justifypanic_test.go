package justifypanic_test

import (
	"os"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/JacobJNilsson/anti-slop-go/analyzers/justifypanic"
)

// TestAnalyzer runs the rejected and the accepted fixtures of rule G11.
// Package "a" holds both, and every rejected form carries a want
// comment, so a call with no want comment must stay clean. The package
// holds the generated fixture and a test file for that reason: each one
// proves an exemption by carrying no want comment at all.
func TestAnalyzer(t *testing.T) {
	results := analysistest.Run(t, analysistest.TestData(), justifypanic.Analyzer, "a")
	assertReportsAtTheCall(t, results)
}

// TestExemptsMainAndInit runs the main package. The exemption reads the
// function and not the package, so main and init stay clean while the
// helper beside them carries a want comment.
func TestExemptsMainAndInit(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), justifypanic.Analyzer, "m")
}

// The two fixture packages of the test-packages setting. Both hold the
// same two shapes: a panic and an os.Exit call in library code.
const (
	suitePackage = "example.com/app/internal/suite"
	storePackage = "example.com/app/internal/store"
)

// A package that the setting names serves tests, so a call that stops
// the process needs no justification there. The suite fixture carries
// no expectation. The store fixture beside it carries one for each
// call, so one run reads both directions of the setting.
func TestNewExemptsATestPackage(t *testing.T) {
	patterns := []string{suitePackage}
	analysistest.Run(t, analysistest.TestData(), justifypanic.New(patterns), suitePackage, storePackage)
}

// The flag is the configuration surface of cmd/antislop and of
// go vet -vettool. It writes into the analyzer instance it sits on, so
// two instances never share a setting.
func TestTestPackagesFlagConfiguresOneInstance(t *testing.T) {
	configured := justifypanic.New(nil)
	if err := configured.Flags.Set("testpackages", ".../internal/suite,other.com/lib"); err != nil {
		t.Fatalf("the testpackages flag rejected a list: %v", err)
	}
	if err := configured.Flags.Set("testpackages", "other.com/tool"); err != nil {
		t.Fatalf("the testpackages flag rejected a second occurrence: %v", err)
	}
	analysistest.Run(t, analysistest.TestData(), configured, suitePackage, storePackage)

	const want = ".../internal/suite,other.com/lib,other.com/tool"
	if got := configured.Flags.Lookup("testpackages").Value.String(); got != want {
		t.Errorf("the flag reads back as %q, want %q", got, want)
	}
	// Another instance holds its own patterns. The plugin builds one
	// instance for each run for this reason: it never writes to the
	// analyzer value that every consumer of the module shares.
	if got := justifypanic.Analyzer.Flags.Lookup("testpackages").Value.String(); got != "" {
		t.Errorf("the analyzer of the module holds the patterns %q of another instance", got)
	}
}

// The package-level analyzer carries the flag, because cmd/antislop
// registers the analyzers of the module through antislop.Analyzers().
func TestAnalyzerCarriesTheFlag(t *testing.T) {
	if justifypanic.Analyzer.Flags.Lookup("testpackages") == nil {
		t.Error("the analyzer of the module registers no testpackages flag")
	}
}

// assertReportsAtTheCall pins the position of the report. The fixture
// DeferredExit holds "defer os.Exit(code)", where the statement starts
// six columns before the call. analysistest matches a want comment by
// line alone, so only a column test holds that difference.
func assertReportsAtTheCall(t *testing.T, results []*analysistest.Result) {
	t.Helper()

	const deferred = "defer os.Exit(code)"
	checked := 0
	for _, result := range results {
		for _, diag := range result.Diagnostics {
			pos := result.Pass.Fset.Position(diag.Pos)
			line := sourceLine(t, pos.Filename, pos.Line)
			if !strings.Contains(line, deferred) {
				continue
			}
			checked++
			want := strings.Index(line, "os.Exit") + 1
			if pos.Column != want {
				t.Errorf("the report sits at column %d; want %d, the column of the call", pos.Column, want)
			}
		}
	}
	// analysistest analyses package "a" twice, once with its test files
	// and once without, so the same diagnostic arrives more than once.
	if checked == 0 {
		t.Error("the deferred fixture gave no diagnostic; the column test proved nothing")
	}
}

// sourceLine returns one line of a fixture file, counted from 1.
func sourceLine(t *testing.T, name string, line int) string {
	t.Helper()

	src, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("cannot read the fixture %s: %v", name, err)
	}
	lines := strings.Split(string(src), "\n")
	if line > len(lines) {
		t.Fatalf("%s holds %d lines; the diagnostic names line %d", name, len(lines), line)
	}

	return lines[line-1]
}
