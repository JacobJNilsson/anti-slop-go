package antislopplugin

import (
	"slices"
	"strings"
	"testing"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"

	antislop "github.com/JacobJNilsson/anti-slop-go"
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
// block. That is the common case and it must give the default rule set.
func TestNewAcceptsNoSettings(t *testing.T) {
	got, err := build(t, nil)
	if err != nil {
		t.Fatalf("BuildAnalyzers returned an unexpected error: %v", err)
	}
	if !slices.Equal(analyzerNames(got), registryNames(t)) {
		t.Errorf("analyzers = %v; want the full rule set %v", analyzerNames(got), registryNames(t))
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
func TestNewRejectsUnknownKey(t *testing.T) {
	_, err := New(map[string]any{"boundary-packages": []any{"example.com/x"}})
	if err == nil {
		t.Fatal("New accepted an unknown settings key; want an error")
	}
	if !strings.Contains(err.Error(), "boundary-packages") {
		t.Errorf("the error does not name the bad key: %v", err)
	}
}

func TestBuildAnalyzersDefaultsToTheFullRuleSet(t *testing.T) {
	got, err := build(t, map[string]any{})
	if err != nil {
		t.Fatalf("BuildAnalyzers returned an unexpected error: %v", err)
	}
	if !slices.Equal(analyzerNames(got), registryNames(t)) {
		t.Errorf("analyzers = %v; want the full rule set %v", analyzerNames(got), registryNames(t))
	}
}

func TestBuildAnalyzersDropsDisabledRules(t *testing.T) {
	known := registryNames(t)
	dropped := known[0]

	got, err := build(t, map[string]any{"disable": []any{dropped}})
	if err != nil {
		t.Fatalf("BuildAnalyzers returned an unexpected error: %v", err)
	}

	names := analyzerNames(got)
	if slices.Contains(names, dropped) {
		t.Errorf("analyzers = %v; %q is disabled and must not appear", names, dropped)
	}
	if len(names) != len(known)-1 {
		t.Errorf("analyzers = %v; want the other %d rules", names, len(known)-1)
	}
}

// Enable selects the opt-in rules. A rule that already runs is not a
// choice, so naming it is a mistake and the run stops. This keeps the
// setting from looking as if it did something.
func TestBuildAnalyzersRejectsADefaultOnRuleInEnable(t *testing.T) {
	known := registryNames(t)

	got, err := build(t, map[string]any{"enable": []any{known[0]}})
	if err == nil {
		t.Fatalf("BuildAnalyzers accepted a default-on rule in enable and returned %v; want an error", analyzerNames(got))
	}

	message := err.Error()
	if !strings.Contains(message, "enable") {
		t.Errorf("the error does not name the setting: %v", err)
	}
	if !strings.Contains(message, known[0]) {
		t.Errorf("the error does not name the rule: %v", err)
	}
	if !strings.Contains(message, "on by default") {
		t.Errorf("the error does not state the reason: %v", err)
	}
}

// No rule is opt-in today, so the filter needs a set of its own to
// exercise. selectAnalyzers is a pure function for that reason.
func TestSelectAnalyzersOptIn(t *testing.T) {
	all := antislop.Analyzers()
	known := registryNames(t)
	chosen := known[0]
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
