// Package antislopplugin registers the anti-slop analyzers as a
// golangci-lint module plugin. A consumer builds a custom golangci-lint
// binary that imports this package; the import runs init, and init
// registers the plugin.
//
// The directory is "plugin", because the import path is the contract a
// consumer writes in .custom-gcl.yml. The package name is not, because
// the standard library has a package "plugin". golangci-lint writes a
// blank import, so no code ever writes this name.
//
// No other package of this module imports this one. The standalone
// cmd/antislop binary and a program that calls antislop.Analyzers()
// therefore never compile the golangci-lint register dependency.
package antislopplugin

import (
	"fmt"
	"slices"
	"strings"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"

	antislop "github.com/JacobJNilsson/anti-slop-go"
	"github.com/JacobJNilsson/anti-slop-go/analyzers/errsemantics"
	"github.com/JacobJNilsson/anti-slop-go/analyzers/fullstructcomp"
	"github.com/JacobJNilsson/anti-slop-go/analyzers/justifypanic"
	"github.com/JacobJNilsson/anti-slop-go/analyzers/noadhoctypeswitch"
	"github.com/JacobJNilsson/anti-slop-go/analyzers/nointerfacereturn"
	"github.com/JacobJNilsson/anti-slop-go/analyzers/noreflect"
)

// name identifies the plugin. golangci-lint needs the same word in the
// linters.settings.custom key and in this registration. A mismatch
// stops the run with "plugin \"antislop\" not found".
const name = "antislop"

func init() { register.Plugin(name, New) }

// optInRules names the rules that a project turns on deliberately.
// Specification 002 gives some rules an opt-in severity, and such a
// rule stays off until the enable setting names it.
//
// This set is where the opt-in severity takes effect. The registry of
// the module holds every rule, because cmd/antislop and go vet read no
// configuration file. 003 records that split.
var optInRules = map[string]bool{
	nointerfacereturn.Analyzer.Name: true, // G09
	justifypanic.Analyzer.Name:      true, // G11
	fullstructcomp.Analyzer.Name:    true, // G12
	errsemantics.Analyzer.Name:      true, // G13
}

// Settings is the configuration surface of the plugin. golangci-lint
// decodes the linters.settings.custom.antislop.settings block into
// this structure with encoding/json, so the fields carry json tags.
// The decoder rejects an unknown field, which makes every documented
// key a promise: a key appears here only when a rule reads it, and a
// key that ships stays.
type Settings struct {
	// BoundaryPackages names the package path patterns that decode
	// input. Rule G06 accepts a type switch on an any value there. 003
	// states the pattern syntax. An empty list names no boundary, which
	// is the default of the rule.
	BoundaryPackages []string `json:"boundary-packages"`

	// ReflectAllow names the package path patterns that may import
	// reflect. Rule G07 reads it. 003 states the pattern syntax. An
	// empty list allows no package, which is the default of the rule.
	ReflectAllow []string `json:"reflect-allow"`

	// FullStructCompMin states the number of distinct fields of one
	// value that a report of rule G12 needs. The key takes an integer.
	// An absent key gives fullstructcomp.DefaultMin, which is why the
	// field is a pointer: the decoder writes zero for an absent key of
	// an integer field, and zero is a setting of its own.
	FullStructCompMin *int `json:"fullstructcomp-min"`

	// Equality turns on the equality forms of rule G13. Such a form
	// compares the message of an error against a string. A package
	// writes that form about its own messages, and the Go wiki page
	// TestComments accepts it. The rule therefore leaves the form alone
	// until this setting is true.
	Equality bool `json:"errsemantics-equality"`

	// Enable names the opt-in rules to run. A rule that is on by
	// default is not a choice, so a name outside optInRules is an
	// error and the run stops.
	Enable []string `json:"enable"`

	// Disable names the rules to drop from the default set. A name in
	// both Enable and Disable stays disabled. A project that disables
	// every rule gets a linter that reports nothing, which is a legal
	// configuration.
	Disable []string `json:"disable"`
}

// plugin holds the decoded settings for one golangci-lint run.
type plugin struct {
	settings Settings
}

// New builds the plugin from the raw settings that golangci-lint read
// from the configuration file. It returns an error for a malformed
// settings block, and golangci-lint stops the run.
//
// CONTRACT: register.NewPlugin fixes this whole signature. The
// parameter type is therefore any, and the result is the
// register.LinterPlugin interface, although this function builds one
// concrete plugin.
func New(conf any) (register.LinterPlugin, error) {
	settings, err := register.DecodeSettings[Settings](conf)
	if err != nil {
		return nil, err
	}

	return &plugin{settings: settings}, nil
}

// BuildAnalyzers returns the analyzers this run must apply.
func (p *plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return selectAnalyzers(p.configured(), optInRules, p.settings)
}

// configured returns the rule set of this run.
//
// A rule that reads a setting gets a new instance, and the rule always
// gets one, whether the settings name it or not. One path is easier to
// read than two, and the instance costs one allocation for each run.
//
// The package-level analyzer values of the module are shared by every
// consumer, and two golangci-lint runs can hold different settings, so
// the plugin never writes to those values. A rule that reads no setting
// keeps its shared value, because nothing writes to it.
//
// The copy of the slice is defence. Analyzers() builds a new slice for
// each call today, and a test of the registry pins that. This function
// writes to the slice, so it takes a copy and depends on no promise of
// another package.
func (p *plugin) configured() []*analysis.Analyzer {
	all := slices.Clone(antislop.Analyzers())
	for i, a := range all {
		switch a.Name {
		case noadhoctypeswitch.Analyzer.Name:
			all[i] = noadhoctypeswitch.New(p.settings.BoundaryPackages)
		case noreflect.Analyzer.Name:
			all[i] = noreflect.New(p.settings.ReflectAllow)
		case fullstructcomp.Analyzer.Name:
			all[i] = fullstructcomp.New(p.minFields())
		case errsemantics.Analyzer.Name:
			all[i] = errsemantics.New(p.settings.Equality)
		}
	}

	return all
}

// minFields returns the setting of rule G12. A configuration that names
// no number gets the default of the rule.
func (p *plugin) minFields() int {
	if p.settings.FullStructCompMin == nil {
		return fullstructcomp.DefaultMin
	}

	return *p.settings.FullStructCompMin
}

// GetLoadMode tells golangci-lint to give the analyzers full type
// information. Every rule reads analysis.Pass.TypesInfo, and the
// syntax load mode leaves that field empty.
func (*plugin) GetLoadMode() string { return register.LoadModeTypesInfo }

// selectAnalyzers applies the settings to the rule set: it drops a
// disabled rule, and it drops an opt-in rule that enable does not
// name. golangci-lint prints an error from here and stops the run, so
// a name that does nothing is an error and not a silent omission.
//
// It takes the opt-in set as a parameter, so a test can supply one.
func selectAnalyzers(all []*analysis.Analyzer, optIn map[string]bool, settings Settings) ([]*analysis.Analyzer, error) {
	known := make([]string, 0, len(all))
	for _, a := range all {
		known = append(known, a.Name)
	}
	slices.Sort(known)

	if err := validateNames(known, "enable", settings.Enable); err != nil {
		return nil, err
	}
	for _, n := range settings.Enable {
		if !optIn[n] {
			return nil, fmt.Errorf("enable: rule %q is on by default; enable selects opt-in rules only", n)
		}
	}
	if err := validateNames(known, "disable", settings.Disable); err != nil {
		return nil, err
	}

	out := make([]*analysis.Analyzer, 0, len(all))
	for _, a := range all {
		if slices.Contains(settings.Disable, a.Name) {
			continue
		}
		if optIn[a.Name] && !slices.Contains(settings.Enable, a.Name) {
			continue
		}
		out = append(out, a)
	}

	return out, nil
}

// validateNames rejects a name in the setting that is not a registered
// rule. known must be sorted, because the message lists it.
func validateNames(known []string, setting string, names []string) error {
	for _, n := range names {
		if !slices.Contains(known, n) {
			return fmt.Errorf("%s: unknown rule %q; the rules are: %s",
				setting, n, strings.Join(known, ", "))
		}
	}

	return nil
}
