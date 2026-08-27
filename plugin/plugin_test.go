package antislopplugin

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"

	antislop "github.com/JacobJNilsson/anti-slop-go"
	"github.com/JacobJNilsson/anti-slop-go/analyzers/errsemantics"
	"github.com/JacobJNilsson/anti-slop-go/analyzers/fullstructcomp"
	"github.com/JacobJNilsson/anti-slop-go/analyzers/noadhoctypeswitch"
	"github.com/JacobJNilsson/anti-slop-go/analyzers/noreflect"
)

// registryNames returns the name of every rule the module provides.
func registryNames(t *testing.T) []string {
	t.Helper()

	names := analyzerNames(antislop.Analyzers())
	if len(names) == 0 {
		t.Fatal("antislop.Analyzers() is empty; the plugin has nothing to build")
	}

	return names
}

// defaultNames returns the rules a run gets with no setting: every rule
// of the registry, without the opt-in rules.
func defaultNames(t *testing.T) []string {
	t.Helper()

	names := registryNames(t)
	if len(optInRules) == 0 {
		t.Fatal("optInRules is empty; 002 gives rules G09, G11, and G13 an opt-in severity")
	}
	out := make([]string, 0, len(names))
	for _, n := range names {
		if !optInRules[n] {
			out = append(out, n)
		}
	}
	if len(out) == len(names) {
		t.Fatal("no rule of the registry is opt-in; optInRules names a rule the registry does not hold")
	}

	return out
}

// defaultRule returns the name of a rule that runs with no setting.
// enable rejects such a name, and disable takes it.
func defaultRule(t *testing.T) string {
	t.Helper()

	return defaultNames(t)[0]
}

// optInNames returns every opt-in rule of the shipped set, in registry
// order, so a test exercises all of them and not only the first.
func optInNames(t *testing.T) []string {
	t.Helper()

	var names []string
	for _, name := range registryNames(t) {
		if optInRules[name] {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		t.Fatal("the module ships no opt-in rule; the enable setting selects nothing")
	}

	return names
}

func analyzerNames(as []*analysis.Analyzer) []string {
	names := make([]string, 0, len(as))
	for _, a := range as {
		names = append(names, a.Name)
	}

	return names
}

// build runs the whole path a golangci-lint run takes: decode the
// settings, then ask for the analyzers.
//
// CONTRACT: the helper passes conf to New, whose signature
// register.NewPlugin fixes.
func build(t *testing.T, conf any) ([]*analysis.Analyzer, error) {
	t.Helper()

	p, err := New(conf)
	if err != nil {
		t.Fatalf("New(%v) returned an unexpected error: %v", conf, err)
	}

	return p.BuildAnalyzers()
}

// The import of this package must be enough to register the plugin,
// because the golangci-lint builder writes a blank import and nothing
// else.
func TestInitRegistersThePlugin(t *testing.T) {
	newPlugin, err := register.GetPlugin(name)
	if err != nil {
		t.Fatalf("register.GetPlugin(%q) failed: %v", name, err)
	}
	if newPlugin == nil {
		t.Fatalf("register.GetPlugin(%q) returned no constructor", name)
	}

	p, err := newPlugin(nil)
	if err != nil {
		t.Fatalf("the registered constructor failed on empty settings: %v", err)
	}
	if _, ok := p.(*plugin); !ok {
		t.Fatalf("the registered constructor returned %T; want the plugin of this package", p)
	}
}

func TestNewDecodesSettings(t *testing.T) {
	known := registryNames(t)

	conf := map[string]any{
		"enable":  []any{known[0]},
		"disable": []any{known[0]},
	}

	p, err := New(conf)
	if err != nil {
		t.Fatalf("New returned an unexpected error: %v", err)
	}

	got, ok := p.(*plugin)
	if !ok {
		t.Fatalf("New returned %T; want the plugin of this package", p)
	}
	if !slices.Equal(got.settings.Enable, []string{known[0]}) {
		t.Errorf("enable = %v; want %v", got.settings.Enable, []string{known[0]})
	}
	if !slices.Equal(got.settings.Disable, []string{known[0]}) {
		t.Errorf("disable = %v; want %v", got.settings.Disable, []string{known[0]})
	}
}

// golangci-lint passes nil when the configuration has no settings
// block. That is the common case and it must give the default rule set,
// which holds every rule that is not opt-in.
func TestNewAcceptsNoSettings(t *testing.T) {
	got, err := build(t, nil)
	if err != nil {
		t.Fatalf("BuildAnalyzers returned an unexpected error: %v", err)
	}
	if !slices.Equal(analyzerNames(got), defaultNames(t)) {
		t.Errorf("analyzers = %v; want the default rule set %v", analyzerNames(got), defaultNames(t))
	}
}

func TestNewRejectsWrongType(t *testing.T) {
	_, err := New(map[string]any{"disable": "safetyassert"})
	if err == nil {
		t.Fatal("New accepted a string where the setting takes a list; want an error")
	}
}

// The decoder rejects an unknown field, so a key of a rule that does
// not exist yet fails the run. This test records that contract.
//
// The promise of the plugin is the failure itself. The message of that
// error belongs to encoding/json, through the register package of
// golangci-lint, and no sentinel and no type name it. The test
// therefore asserts that the call failed, which rule G13 asks for.
func TestNewRejectsUnknownKey(t *testing.T) {
	if _, err := New(map[string]any{"panic-allow": []any{"example.com/x"}}); err == nil {
		t.Fatal("New accepted an unknown settings key; want an error")
	}
}

func TestBuildAnalyzersDefaultsToTheDefaultRuleSet(t *testing.T) {
	got, err := build(t, map[string]any{})
	if err != nil {
		t.Fatalf("BuildAnalyzers returned an unexpected error: %v", err)
	}
	if !slices.Equal(analyzerNames(got), defaultNames(t)) {
		t.Errorf("analyzers = %v; want the default rule set %v", analyzerNames(got), defaultNames(t))
	}
}

func TestBuildAnalyzersDropsDisabledRules(t *testing.T) {
	byDefault := defaultNames(t)
	dropped := byDefault[0]

	got, err := build(t, map[string]any{"disable": []any{dropped}})
	if err != nil {
		t.Fatalf("BuildAnalyzers returned an unexpected error: %v", err)
	}

	names := analyzerNames(got)
	if slices.Contains(names, dropped) {
		t.Errorf("analyzers = %v; %q is disabled and must not appear", names, dropped)
	}
	if len(names) != len(byDefault)-1 {
		t.Errorf("analyzers = %v; want the other %d rules", names, len(byDefault)-1)
	}
}

// The opt-in severity of 002 reaches the golangci-lint path here: the
// registry holds rules G09, G11, and G13, and a run without the enable
// setting must not apply them. This test runs the real opt-in set, and
// not the synthetic one of TestSelectAnalyzersOptIn. It checks every
// opt-in rule, one at a time and all together, so a rule that joins
// the set later is covered with no test change.
func TestBuildAnalyzersAppliesTheOptInSeverity(t *testing.T) {
	optIns := optInNames(t)

	byDefault, err := build(t, map[string]any{})
	if err != nil {
		t.Fatalf("BuildAnalyzers returned an unexpected error: %v", err)
	}
	for _, optIn := range optIns {
		if names := analyzerNames(byDefault); slices.Contains(names, optIn) {
			t.Errorf("analyzers = %v; %q is opt-in and enable does not name it", names, optIn)
		}
	}

	for _, optIn := range optIns {
		one, err := build(t, map[string]any{"enable": []any{optIn}})
		if err != nil {
			t.Fatalf("BuildAnalyzers returned an unexpected error: %v", err)
		}
		names := analyzerNames(one)
		if !slices.Contains(names, optIn) {
			t.Errorf("analyzers = %v; enable names %q, so the run must apply it", names, optIn)
		}
		for _, other := range optIns {
			if other != optIn && slices.Contains(names, other) {
				t.Errorf("analyzers = %v; enable named only %q, so %q must stay off", names, optIn, other)
			}
		}
	}

	all := make([]any, 0, len(optIns))
	for _, optIn := range optIns {
		all = append(all, optIn)
	}
	enabled, err := build(t, map[string]any{"enable": all})
	if err != nil {
		t.Fatalf("BuildAnalyzers returned an unexpected error: %v", err)
	}
	if names := analyzerNames(enabled); !slices.Equal(names, registryNames(t)) {
		t.Errorf("analyzers = %v; enable names every opt-in rule, so the run applies every rule %v", names, registryNames(t))
	}
}

// Enable selects the opt-in rules. A rule that already runs is not a
// choice, so naming it is a mistake and the run stops. This keeps the
// setting from looking as if it did something.
func TestBuildAnalyzersRejectsADefaultOnRuleInEnable(t *testing.T) {
	onByDefault := defaultRule(t)

	got, err := build(t, map[string]any{"enable": []any{onByDefault}})
	if err == nil {
		t.Fatalf("BuildAnalyzers accepted a default-on rule in enable and returned %v; want an error", analyzerNames(got))
	}

	message := err.Error()
	if !strings.Contains(message, "enable") {
		t.Errorf("the error does not name the setting: %v", err)
	}
	if !strings.Contains(message, onByDefault) {
		t.Errorf("the error does not name the rule: %v", err)
	}
	if !strings.Contains(message, "on by default") {
		t.Errorf("the error does not state the reason: %v", err)
	}
}

// selectAnalyzers takes the opt-in set as a parameter, so this test
// supplies one of its own and reads the filter alone. It names a rule
// that ships on by default, so the three cases hold whatever the
// shipped opt-in set holds.
// TestBuildAnalyzersAppliesTheOptInSeverity runs the real set.
func TestSelectAnalyzersOptIn(t *testing.T) {
	all := antislop.Analyzers()
	known := registryNames(t)
	chosen := defaultNames(t)[0]
	optIn := map[string]bool{chosen: true}

	tests := []struct {
		name     string
		settings Settings
		want     []string
	}{
		{
			name:     "an opt-in rule stays off when enable omits it",
			settings: Settings{},
			want:     without(known, chosen),
		},
		{
			name:     "enable turns the opt-in rule on",
			settings: Settings{Enable: []string{chosen}},
			want:     known,
		},
		{
			name:     "disable wins over enable",
			settings: Settings{Enable: []string{chosen}, Disable: []string{chosen}},
			want:     without(known, chosen),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := selectAnalyzers(all, optIn, tt.settings)
			if err != nil {
				t.Fatalf("selectAnalyzers returned an unexpected error: %v", err)
			}
			if !slices.Equal(analyzerNames(got), tt.want) {
				t.Errorf("analyzers = %v; want %v", analyzerNames(got), tt.want)
			}
		})
	}
}

// without returns names without the one named.
func without(names []string, drop string) []string {
	out := make([]string, 0, len(names))
	for _, n := range names {
		if n != drop {
			out = append(out, n)
		}
	}

	return out
}

// A typo must stop the run. golangci-lint prints the error from
// BuildAnalyzers and exits.
func TestBuildAnalyzersRejectsUnknownNames(t *testing.T) {
	known := registryNames(t)

	// Both settings validate their names, so both must reject a typo.
	for _, setting := range []string{"enable", "disable"} {
		t.Run(setting, func(t *testing.T) {
			got, err := build(t, map[string]any{setting: []any{"nosuchrule"}})
			if err == nil {
				t.Fatalf("BuildAnalyzers accepted an unknown rule name and returned %v; want an error", analyzerNames(got))
			}

			message := err.Error()
			if !strings.Contains(message, setting) {
				t.Errorf("the error does not name the setting %q: %v", setting, err)
			}
			if !strings.Contains(message, "nosuchrule") {
				t.Errorf("the error does not name the bad rule: %v", err)
			}
			// The message must help the reader find the right name. The
			// list is sorted, so the reader reads it as a list and not
			// as a registration order.
			sorted := slices.Sorted(slices.Values(known))
			if !strings.Contains(message, strings.Join(sorted, ", ")) {
				t.Errorf("the error does not list the known rules in order (%v): %v", sorted, err)
			}
		})
	}
}

// The analyzers read type information, and the syntax load mode does
// not supply it.
func TestGetLoadMode(t *testing.T) {
	p, err := New(nil)
	if err != nil {
		t.Fatalf("New returned an unexpected error: %v", err)
	}
	if got := p.GetLoadMode(); got != register.LoadModeTypesInfo {
		t.Errorf("GetLoadMode() = %q; want %q", got, register.LoadModeTypesInfo)
	}
}

// The reflect-allow setting is the configuration surface of rule G07 on
// the golangci-lint path. golangci-lint never sets analyzer flags, so
// the plugin builds the analyzer through its constructor.
func TestNewDecodesReflectAllow(t *testing.T) {
	patterns := []string{"example.com/app/internal/codec", ".../internal/wire"}

	p, err := New(map[string]any{"reflect-allow": []any{patterns[0], patterns[1]}})
	if err != nil {
		t.Fatalf("New returned an unexpected error: %v", err)
	}

	got, ok := p.(*plugin)
	if !ok {
		t.Fatalf("New returned %T; want the plugin of this package", p)
	}
	if !slices.Equal(got.settings.ReflectAllow, patterns) {
		t.Errorf("reflect-allow = %v; want %v", got.settings.ReflectAllow, patterns)
	}
}

// The patterns must reach the analyzer, so this test runs the analyzer
// that BuildAnalyzers returned. The allowed fixture carries no want
// comment and the other one does, so one run tests both directions.
func TestBuildAnalyzersGivesNoreflectItsPatterns(t *testing.T) {
	built, err := build(t, map[string]any{"reflect-allow": []any{"allowed"}})
	if err != nil {
		t.Fatalf("BuildAnalyzers returned an unexpected error: %v", err)
	}

	configured := byName(t, built, noreflect.Analyzer.Name)
	analysistest.Run(t, analysistest.TestData(), configured, "allowed", "notallowed")
}

// The boundary-packages setting is the configuration surface of rule
// G06 on the golangci-lint path. It takes the same path patterns.
func TestNewDecodesBoundaryPackages(t *testing.T) {
	patterns := []string{"example.com/app/internal/ingest", ".../api/..."}

	p, err := New(map[string]any{"boundary-packages": []any{patterns[0], patterns[1]}})
	if err != nil {
		t.Fatalf("New returned an unexpected error: %v", err)
	}

	got, ok := p.(*plugin)
	if !ok {
		t.Fatalf("New returned %T; want the plugin of this package", p)
	}
	if !slices.Equal(got.settings.BoundaryPackages, patterns) {
		t.Errorf("boundary-packages = %v; want %v", got.settings.BoundaryPackages, patterns)
	}
}

// The patterns must reach the analyzer of rule G06 too. The boundary
// fixture carries no expectation comment and the other one does, so one
// run tests both directions.
func TestBuildAnalyzersGivesNoadhoctypeswitchItsPatterns(t *testing.T) {
	built, err := build(t, map[string]any{"boundary-packages": []any{"boundary"}})
	if err != nil {
		t.Fatalf("BuildAnalyzers returned an unexpected error: %v", err)
	}

	configured := byName(t, built, noadhoctypeswitch.Analyzer.Name)
	analysistest.Run(t, analysistest.TestData(), configured, "boundary", "notboundary")
}

// The errsemantics-equality setting is the configuration surface of
// rule G13 on the golangci-lint path. It takes a boolean and no
// pattern, and the decoder reads it like the other keys.
func TestNewDecodesErrsemanticsEquality(t *testing.T) {
	p, err := New(map[string]any{"errsemantics-equality": true})
	if err != nil {
		t.Fatalf("New returned an unexpected error: %v", err)
	}

	got, ok := p.(*plugin)
	if !ok {
		t.Fatalf("New returned %T; want the plugin of this package", p)
	}
	if !got.settings.Equality {
		t.Error("errsemantics-equality = false; want true")
	}
}

// The setting must reach the analyzer, so this test runs the analyzer
// that BuildAnalyzers returned. Rule G13 is opt-in, so enable names it.
func TestBuildAnalyzersGivesErrsemanticsItsSetting(t *testing.T) {
	built, err := build(t, map[string]any{
		"errsemantics-equality": true,
		"enable":                []any{errsemantics.Analyzer.Name},
	})
	if err != nil {
		t.Fatalf("BuildAnalyzers returned an unexpected error: %v", err)
	}

	configured := byName(t, built, errsemantics.Analyzer.Name)
	analysistest.Run(t, analysistest.TestData(), configured, "errtext")
}

// Two golangci-lint runs can hold different settings, and the analyzer
// values of the module are shared. The plugin must therefore build its
// own instance of every configurable rule, and never write to the
// shared one.
func TestBuildAnalyzersLeavesTheSharedAnalyzersAlone(t *testing.T) {
	built, err := build(t, map[string]any{
		"reflect-allow":            []any{"allowed"},
		"boundary-packages":        []any{"boundary"},
		"fullstructcomp-min":       3,
		"fullstructcomp-maxignore": 20,
		"errsemantics-equality":    true,
		"enable": []any{
			fullstructcomp.Analyzer.Name,
			errsemantics.Analyzer.Name,
		},
	})
	if err != nil {
		t.Fatalf("BuildAnalyzers returned an unexpected error: %v", err)
	}

	// Each rule states the value its shared analyzer reads with no
	// setting. A pattern list is empty there, a number is the default of
	// its rule, and a boolean is false.
	configurable := []struct {
		shared *analysis.Analyzer
		flag   string
		clean  string
	}{
		{noreflect.Analyzer, "allow", ""},
		{noadhoctypeswitch.Analyzer, "boundary", ""},
		{fullstructcomp.Analyzer, "min", strconv.Itoa(fullstructcomp.DefaultMin)},
		{fullstructcomp.Analyzer, "maxignore", strconv.Itoa(fullstructcomp.DefaultMaxIgnore)},
		{errsemantics.Analyzer, "equality", "false"},
	}
	for _, rule := range configurable {
		t.Run(rule.shared.Name+"."+rule.flag, func(t *testing.T) {
			if configured := byName(t, built, rule.shared.Name); configured == rule.shared {
				t.Error("BuildAnalyzers returned the shared analyzer; the next run would inherit these settings")
			}
			if got := rule.shared.Flags.Lookup(rule.flag).Value.String(); got != rule.clean {
				t.Errorf("the shared analyzer holds the setting %q of one run", got)
			}
		})
	}
}

// byName returns the analyzer of one rule.
func byName(t *testing.T, as []*analysis.Analyzer, name string) *analysis.Analyzer {
	t.Helper()

	for _, a := range as {
		if a.Name == name {
			return a
		}
	}
	t.Fatalf("the built rule set holds no rule named %q", name)

	return nil
}

// The fullstructcomp-min setting is the configuration surface of rule
// G12 on the golangci-lint path. golangci-lint sets no analyzer flag,
// so the plugin builds the analyzer through its constructor.
func TestNewDecodesFullStructCompMin(t *testing.T) {
	p, err := New(map[string]any{"fullstructcomp-min": 3})
	if err != nil {
		t.Fatalf("New returned an unexpected error: %v", err)
	}

	got, ok := p.(*plugin)
	if !ok {
		t.Fatalf("New returned %T; want the plugin of this package", p)
	}
	if got.settings.FullStructCompMin == nil {
		t.Fatal("fullstructcomp-min = none; want the number the settings hold")
	}
	if *got.settings.FullStructCompMin != 3 {
		t.Errorf("fullstructcomp-min = %d; want 3", *got.settings.FullStructCompMin)
	}
}

// A configuration that names no number gets the default of the rule.
// The field is a pointer for this reason: an integer field would read
// an absent key as zero, and zero is a setting of its own.
func TestBuildAnalyzersDefaultsTheMinimum(t *testing.T) {
	p, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned an unexpected error: %v", err)
	}

	got, ok := p.(*plugin)
	if !ok {
		t.Fatalf("New returned %T; want the plugin of this package", p)
	}
	if min := got.minFields(); min != fullstructcomp.DefaultMin {
		t.Errorf("min = %d; want the default %d of the rule", min, fullstructcomp.DefaultMin)
	}
}

// The number must reach the analyzer, so this test runs the analyzer
// that BuildAnalyzers returned. The fixture holds a run of two fields
// and a run of three, so one run tests both directions of the setting.
func TestBuildAnalyzersGivesFullstructcompItsMinimum(t *testing.T) {
	built, err := build(t, map[string]any{
		"fullstructcomp-min": 3,
		"enable":             []any{fullstructcomp.Analyzer.Name},
	})
	if err != nil {
		t.Fatalf("BuildAnalyzers returned an unexpected error: %v", err)
	}

	configured := byName(t, built, fullstructcomp.Analyzer.Name)
	analysistest.Run(t, analysistest.TestData(), configured, "structmin")
}

// The fullstructcomp-maxignore setting carries the cost gate of rule
// G12 on the golangci-lint path.
func TestNewDecodesFullStructCompMaxIgnore(t *testing.T) {
	p, err := New(map[string]any{"fullstructcomp-maxignore": 20})
	if err != nil {
		t.Fatalf("New returned an unexpected error: %v", err)
	}

	got, ok := p.(*plugin)
	if !ok {
		t.Fatalf("New returned %T; want the plugin of this package", p)
	}
	if got.settings.FullStructCompMaxIgnore == nil {
		t.Fatal("fullstructcomp-maxignore = none; want the number the settings hold")
	}
	if *got.settings.FullStructCompMaxIgnore != 20 {
		t.Errorf("fullstructcomp-maxignore = %d; want 20", *got.settings.FullStructCompMaxIgnore)
	}
}

// A configuration that names no cost gets the default of the rule. The
// field is a pointer for the same reason the minimum is one: zero is a
// setting of its own, and it reports a group whose fix carries no
// ignore name.
func TestBuildAnalyzersDefaultsTheMaxIgnore(t *testing.T) {
	p, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned an unexpected error: %v", err)
	}

	got, ok := p.(*plugin)
	if !ok {
		t.Fatalf("New returned %T; want the plugin of this package", p)
	}
	if names := got.maxIgnoreNames(); names != fullstructcomp.DefaultMaxIgnore {
		t.Errorf("maxignore = %d; want the default %d of the rule", names, fullstructcomp.DefaultMaxIgnore)
	}
}

// The number must reach the analyzer, so this test runs the analyzer
// that BuildAnalyzers returned. The fixture holds a group whose fix
// needs six ignore names, which the default of the rule rejects.
func TestBuildAnalyzersGivesFullstructcompItsMaxIgnore(t *testing.T) {
	built, err := build(t, map[string]any{
		"fullstructcomp-maxignore": 20,
		"enable":                   []any{fullstructcomp.Analyzer.Name},
	})
	if err != nil {
		t.Fatalf("BuildAnalyzers returned an unexpected error: %v", err)
	}

	configured := byName(t, built, fullstructcomp.Analyzer.Name)
	analysistest.Run(t, analysistest.TestData(), configured, "structignore")
}
