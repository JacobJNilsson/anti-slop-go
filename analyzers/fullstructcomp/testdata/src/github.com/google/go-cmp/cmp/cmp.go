// Package cmp fakes github.com/google/go-cmp/cmp for the fixtures of
// rule G12. analysistest fetches no module, so the fixture owns the
// import path that the rule resolves.
package cmp

// Option is the type of an option of Diff.
type Option interface{}

// Diff returns the difference of two values.
func Diff(x, y any, opts ...Option) string { return "" }
