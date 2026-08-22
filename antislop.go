// Package antislop provides go/analysis analyzers that reject
// low-evidence Go patterns. The rule catalogue lives in docs/spec.
package antislop

import (
	"golang.org/x/tools/go/analysis"

	"github.com/JacobJNilsson/anti-slop-go/analyzers/fullstructcomp"
	"github.com/JacobJNilsson/anti-slop-go/analyzers/justifypanic"
	"github.com/JacobJNilsson/anti-slop-go/analyzers/noadhoctypeswitch"
	"github.com/JacobJNilsson/anti-slop-go/analyzers/noanyparam"
	"github.com/JacobJNilsson/anti-slop-go/analyzers/noanyreturn"
	"github.com/JacobJNilsson/anti-slop-go/analyzers/noerrorassert"
	"github.com/JacobJNilsson/anti-slop-go/analyzers/nointerfacereturn"
	"github.com/JacobJNilsson/anti-slop-go/analyzers/nolaundering"
	"github.com/JacobJNilsson/anti-slop-go/analyzers/nomonkeypatch"
	"github.com/JacobJNilsson/anti-slop-go/analyzers/noreflect"
	"github.com/JacobJNilsson/anti-slop-go/analyzers/nountypedmap"
	"github.com/JacobJNilsson/anti-slop-go/analyzers/safetyassert"
)

// Analyzers returns every analyzer this module provides, the opt-in
// rules included. Consumers register the full list; rule toggles happen
// in the consumer's configuration, not here.
//
// One registry serves the three consumption paths of 003. The
// golangci-lint plugin reads a configuration file, so it applies the
// opt-in severity of 002 and drops such a rule until enable names it.
// cmd/antislop and go vet -vettool read no configuration file, so they
// run every rule. The reader turns one off with the -NAME=false flag
// that multichecker gives each analyzer.
func Analyzers() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		safetyassert.Analyzer,      // G01
		nountypedmap.Analyzer,      // G02
		noanyparam.Analyzer,        // G03
		noanyreturn.Analyzer,       // G04
		nolaundering.Analyzer,      // G05
		noadhoctypeswitch.Analyzer, // G06
		noreflect.Analyzer,         // G07
		nomonkeypatch.Analyzer,     // G08
		nointerfacereturn.Analyzer, // G09, opt-in
		noerrorassert.Analyzer,     // G10
		justifypanic.Analyzer,      // G11 (opt-in)
		fullstructcomp.Analyzer,    // G12 (opt-in)
	}
}
