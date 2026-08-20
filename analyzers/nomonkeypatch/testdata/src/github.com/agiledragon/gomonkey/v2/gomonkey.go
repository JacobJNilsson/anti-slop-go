// Package gomonkey fakes the runtime patching library
// github.com/agiledragon/gomonkey. The fixture owns the version
// directory of the path, which the rule must match as well.
package gomonkey

// ApplyFunc fakes the entry point of the real library.
func ApplyFunc(target, double any) {}
