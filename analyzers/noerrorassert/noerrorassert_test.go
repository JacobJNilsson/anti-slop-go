package noerrorassert_test

import (
	"fmt"
	"go/token"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/JacobJNilsson/anti-slop-go/analyzers/noerrorassert"
)

// TestAnalyzer runs the rejected and the accepted fixtures of rule G10.
// Package "a" holds both. Every rejected form carries a want comment,
// so a file with no want comment must stay clean. The package holds
// the generated fixture and a test file for that reason: the generated
// file proves the exemption, and the test file proves that test code
// gets none.
func TestAnalyzer(t *testing.T) {
	results := analysistest.Run(t, analysistest.TestData(), noerrorassert.Analyzer, "a")
	assertChainedReportsAtTwoColumns(t, results)
}

// assertChainedReportsAtTwoColumns pins the position of the report.
// The fixture Chained holds two assertions on one line, and the rule
// reports at the .( token of each one. A report at the start of the
// operand would put both diagnostics at one position, and the reader
// could not tell them apart. analysistest matches a want comment by
// line alone, so only a column test holds that difference.
func assertChainedReportsAtTwoColumns(t *testing.T, results []*analysistest.Result) {
	t.Helper()
	pairs := 0
	for _, result := range results {
		byLine := make(map[string][]token.Position)
		for _, diag := range result.Diagnostics {
			pos := result.Pass.Fset.Position(diag.Pos)
			key := fmt.Sprintf("%s:%d", pos.Filename, pos.Line)
			byLine[key] = append(byLine[key], pos)
		}
		for key, positions := range byLine {
			if len(positions) < 2 {
				continue
			}
			pairs++
			if positions[0].Column == positions[1].Column {
				t.Errorf("%s: two diagnostics share column %d; the rule reports at the .( token of each assertion",
					key, positions[0].Column)
			}
		}
	}
	if pairs == 0 {
		t.Error("no line carried two diagnostics; the chained fixture must give two")
	}
}
