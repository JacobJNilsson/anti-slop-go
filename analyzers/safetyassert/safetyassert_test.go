package safetyassert_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/JacobJNilsson/anti-slop-go/analyzers/safetyassert"
)

// TestAnalyzer runs the rejected and the accepted fixtures of rule G01.
// Package "a" holds both; every rejected form carries a want comment.
func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), safetyassert.Analyzer, "a")
}

// TestSkipsGeneratedFiles pins the exemption for generated code. The
// package holds one file with the canonical header and one file with a
// lax header. Each file holds an unjustified assertion and no want
// comment, so any diagnostic fails the test.
func TestSkipsGeneratedFiles(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), safetyassert.Analyzer, "gen")
}
