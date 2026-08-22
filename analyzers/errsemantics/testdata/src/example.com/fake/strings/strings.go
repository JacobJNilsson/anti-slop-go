// Package strings fakes the standard library package of that name. The
// rule resolves the object of a call to its import path, so a call of
// this Contains is no finding. A rule that read the name of the
// package would report it.
package strings

// Contains carries the signature of strings.Contains.
func Contains(s, substr string) bool { return false }
