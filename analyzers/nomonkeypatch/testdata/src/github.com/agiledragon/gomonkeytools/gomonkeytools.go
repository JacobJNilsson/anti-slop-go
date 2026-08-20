// Package gomonkeytools has a path that starts with the letters of a
// patching library, and it is another library. The rule matches the
// path itself, or a directory under it, so this import stays clean.
package gomonkeytools

// Noop does nothing.
func Noop() {}
