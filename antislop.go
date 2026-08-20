// Package antislop provides go/analysis analyzers that reject
// low-evidence Go patterns. The rule catalogue lives in docs/spec.
package antislop

import (
	"golang.org/x/tools/go/analysis"

	"github.com/JacobJNilsson/anti-slop-go/analyzers/nountypedmap"
	"github.com/JacobJNilsson/anti-slop-go/analyzers/safetyassert"
)

// Analyzers returns every analyzer this module provides.
// Consumers register the full list; rule toggles happen in the
// consumer's configuration, not here.
func Analyzers() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		safetyassert.Analyzer, // G01
		nountypedmap.Analyzer, // G02
	}
}
