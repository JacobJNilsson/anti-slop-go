package nointerfacereturn_test

import (
	"go/token"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/JacobJNilsson/anti-slop-go/analyzers/nointerfacereturn"
)

// TestAnalyzer runs the rejected and the accepted fixtures of rule G09.
// Package "a" holds both. Every rejected form carries a want comment,
// so a form with no want comment must stay clean. The package holds the
// generated fixture and a test file for that reason: the generated file
// proves the exemption, and the test file proves that test code gets
// none.
func TestAnalyzer(t *testing.T) {
	results := analysistest.Run(t, analysistest.TestData(), nointerfacereturn.Analyzer, "a")
	assertReportsAtTheResultType(t, results)
}

// assertReportsAtTheResultType pins the position of the report. The
// fixture Mixed spreads its results over three lines, and the rule
// reports at the type of the result it judges. analysistest matches a
// want comment by line, so the fixture holds the position test for a
// signature that spans lines. This test pins the column: a report must
// sit at the type of the result, and not at the func keyword.
func assertReportsAtTheResultType(t *testing.T, results []*analysistest.Result) {
	t.Helper()

	found := 0
	for _, result := range results {
		for _, diag := range result.Diagnostics {
			pos := result.Pass.Fset.Position(diag.Pos)
			if !strings.Contains(diag.Message, "b.Item") {
				continue
			}
			line := sourceLine(t, result, pos)
			if !strings.HasPrefix(strings.TrimSpace(line[pos.Column-1:]), "b.Item") {
				t.Errorf("%v: the diagnostic sits at %q; want the result type", pos, line[pos.Column-1:])
			}
			found++
		}
	}
	if found == 0 {
		t.Error("no diagnostic named b.Item; the fixtures must give some")
	}
}

// sourceLine returns the line of the fixture that a position names.
func sourceLine(t *testing.T, result *analysistest.Result, pos token.Position) string {
	t.Helper()

	for _, file := range result.Pass.Files {
		tokenFile := result.Pass.Fset.File(file.FileStart)
		if tokenFile.Name() != pos.Filename {
			continue
		}
		src, err := result.Pass.ReadFile(pos.Filename)
		if err != nil {
			t.Fatalf("read %s: %v", pos.Filename, err)
		}
		lines := strings.Split(string(src), "\n")
		if pos.Line > len(lines) {
			t.Fatalf("%v: the file holds %d lines", pos, len(lines))
		}

		return lines[pos.Line-1]
	}
	t.Fatalf("%v: no file of the pass carries this position", pos)

	return ""
}
