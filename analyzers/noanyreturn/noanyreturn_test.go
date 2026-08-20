package noanyreturn_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/JacobJNilsson/anti-slop-go/analyzers/noanyreturn"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), noanyreturn.Analyzer, "a")
}
