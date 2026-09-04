// Package cmp carries the name of the go-cmp package and another
// import path. The rule resolves the import path of the call, so no
// call of this package counts.
package cmp

// Diff carries the name of the go-cmp function.
func Diff(x, y any) string { return "" }
