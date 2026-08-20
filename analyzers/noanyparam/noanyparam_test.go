package noanyparam_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/JacobJNilsson/anti-slop-go/analyzers/noanyparam"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), noanyparam.Analyzer, "a")
}
