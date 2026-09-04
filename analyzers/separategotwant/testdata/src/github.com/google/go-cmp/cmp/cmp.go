// Package cmp fakes github.com/google/go-cmp/cmp for the fixtures of
// rule G14. analysistest builds a fixture from the files under testdata
// and fetches no module, so the fixture owns the import path that the
// rule resolves. The signatures match the real package at the positions
// the rule reads: Diff takes the two values first, then the options.
package cmp

// Option is the option interface of the real package.
type Option interface {
	option()
}

// Diff reports the difference between two values. The rule reads this
// function of the package, and it judges every argument of it.
func Diff(x, y any, opts ...Option) string { return "" }

// Equal answers the comparison with a boolean. The rule reads Diff
// alone, so no call of this function counts.
func Equal(x, y any, opts ...Option) bool { return true }
