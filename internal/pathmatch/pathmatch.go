// Package pathmatch matches the import path of a package against the
// path patterns of a configuration setting. One implementation serves
// every setting that names packages, so two settings cannot answer the
// same pattern differently.
//
// docs/spec/003-implementation.md states the syntax, and this package
// is the implementation of it:
//
//   - A pattern matches the whole path. A pattern is therefore no
//     prefix, no infix, and no suffix of a longer path.
//   - "*" matches any run of characters that holds no slash. The run
//     may be empty.
//   - "..." matches any run of characters, slashes included. The run
//     may be empty.
//   - A pattern that ends in "/..." matches the package above that
//     suffix as well, which is the rule of the go command:
//     "example.com/app/..." matches "example.com/app".
//   - Every other character of a pattern matches itself. A dot of a
//     domain name is a dot, and no wildcard.
package pathmatch

import (
	"regexp"
	"strings"
)

// Any reports whether path matches one of the patterns. An empty list
// matches no package, so a setting that names no pattern allows
// nothing.
func Any(patterns []string, path string) bool {
	for _, pattern := range patterns {
		if Match(pattern, path) {
			return true
		}
	}
	return false
}

// Match reports whether path matches one pattern.
func Match(pattern, path string) bool {
	// The go command reads "example.com/app/..." as the package and
	// every package under it. The expression below covers the packages
	// under it, and this test covers the package itself.
	//
	// The package above the subtree reads as a pattern again, and never
	// as text. The part before the suffix can hold wildcards, and
	// "example.com/*/..." must match "example.com/app".
	if above, isSubtree := strings.CutSuffix(pattern, "/..."); isSubtree && Match(above, path) {
		return true
	}
	return expr(pattern).MatchString(path)
}

// expr compiles one pattern into an anchored expression. The anchors
// are \A and \z, which no character of a path can move: a pattern
// matches the whole path.
//
// The literal runs go through regexp.QuoteMeta, so a character with a
// meaning in an expression, such as the dot of a domain name, matches
// itself only.
func expr(pattern string) *regexp.Regexp {
	var out strings.Builder
	out.WriteString(`\A`)
	for i := 0; i < len(pattern); {
		switch {
		case strings.HasPrefix(pattern[i:], "..."):
			out.WriteString(`.*`)
			i += len("...")
		case pattern[i] == '*':
			out.WriteString(`[^/]*`)
			i++
		default:
			start := i
			for i < len(pattern) && pattern[i] != '*' && !strings.HasPrefix(pattern[i:], "...") {
				i++
			}
			out.WriteString(regexp.QuoteMeta(pattern[start:i]))
		}
	}
	out.WriteString(`\z`)
	// The builder writes an anchor, a quoted literal, and two fixed
	// wildcards only, so the expression always compiles.
	return regexp.MustCompile(out.String())
}
