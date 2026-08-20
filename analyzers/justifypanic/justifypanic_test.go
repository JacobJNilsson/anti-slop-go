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
