package pathmatch_test

import (
	"testing"

	"github.com/JacobJNilsson/anti-slop-go/internal/pathmatch"
)

// TestMatch pins the pattern syntax that docs/spec/003-implementation.md
// states. A configuration entry that matches one package too many
// disables a rule where the project did not ask for it, so every clause
// of the syntax gets a case here.
func TestMatch(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		{"a path equal to the pattern", "example.com/app", "example.com/app", true},
		{"another path", "example.com/app", "example.com/other", false},

		// The pattern matches the whole path. A bare segment is
		// therefore a whole path and never a suffix.
		{"a bare segment matches the whole path", "codec", "codec", true},
		{"a bare segment is no suffix", "codec", "example.com/app/codec", false},
		{"a pattern is no prefix", "example.com", "example.com/app", false},
		{"a pattern is no infix", "app", "example.com/app/codec", false},

		// A star matches inside one segment only.
		{"a star matches one segment", "*/internal/codec", "app/internal/codec", true},
		{"a star does not cross a slash", "*/internal/codec", "example.com/app/internal/codec", false},
		{"a star matches an empty run", "codec*", "codec", true},
		{"a star matches a part of a segment", "example.com/app/codec*", "example.com/app/codecv2", true},
		{"a star inside a path", "example.com/*/codec", "example.com/app/codec", true},
		{"a star does not swallow two segments", "example.com/*/codec", "example.com/app/internal/codec", false},

		// Three dots match any run of characters, slashes included.
		{"three dots cross a slash", ".../internal/codec", "example.com/app/internal/codec", true},
		{"three dots match one segment too", ".../internal/codec", "app/internal/codec", true},
		{"three dots alone match every path", "...", "example.com/app", true},
		{"three dots need the slash of the pattern", ".../codec", "codec", false},
		{"a subtree pattern matches a package under it", "example.com/app/...", "example.com/app/internal/codec", true},
		{"a subtree pattern matches the package itself", "example.com/app/...", "example.com/app", true},
		// The package above the subtree reads through the wildcards of
		// the pattern too. A text comparison would answer false here.
		{"a subtree pattern with three dots matches the package itself", ".../internal/codec/...", "example.com/a/internal/codec", true},
		{"a subtree pattern with a star matches the package itself", "example.com/*/...", "example.com/app", true},
		{"a subtree pattern stops at the segment", "example.com/app/...", "example.com/apple", false},
		{"a subtree pattern is anchored at the start", "example.com/app/...", "other.com/example.com/app/x", false},

		// Every other character of the pattern matches itself, and the
		// expression the matcher builds must not read it.
		{"a dot is no wildcard", "example.com/app", "exampleXcom/app", false},
		{"a plus is no repetition", "example.com/a+", "example.com/aa", false},
		{"a plus matches itself", "example.com/a+", "example.com/a+", true},
		{"a bracket matches itself", "example.com/a[b", "example.com/a[b", true},

		{"an empty pattern matches no package", "", "example.com/app", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pathmatch.Match(tt.pattern, tt.path); got != tt.want {
				t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
			}
		})
	}
}

// TestAny drives the list form that a setting holds. A rule asks one
// question of the whole list.
func TestAny(t *testing.T) {
	patterns := []string{"example.com/app/internal/codec", ".../internal/wire"}

	tests := []struct {
		name     string
		patterns []string
		path     string
		want     bool
	}{
		{"no pattern matches no package", nil, "example.com/app", false},
		{"the first pattern matches", patterns, "example.com/app/internal/codec", true},
		{"a later pattern matches", patterns, "other.com/lib/internal/wire", true},
		{"no pattern of the list matches", patterns, "example.com/app/service", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pathmatch.Any(tt.patterns, tt.path); got != tt.want {
				t.Errorf("Any(%v, %q) = %v, want %v", tt.patterns, tt.path, got, tt.want)
			}
		})
	}
}
