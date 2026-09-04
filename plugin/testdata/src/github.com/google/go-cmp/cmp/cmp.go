// Package cmp fakes github.com/google/go-cmp/cmp for the fixtures of
// the plugin. analysistest builds a fixture from the files under
// testdata and fetches no module, so the fixture owns the import path
// that rule G14 resolves.
package cmp

// Option is the option interface of the real package.
type Option interface {
	option()
}

// Diff reports the difference between two values.
func Diff(x, y any, opts ...Option) string { return "" }
