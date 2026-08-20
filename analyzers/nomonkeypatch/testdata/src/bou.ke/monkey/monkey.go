// Package monkey fakes the runtime patching library bou.ke/monkey.
// analysistest loads no module from the network, so the fixture owns
// this import path in its own GOPATH tree. The behaviour of the real
// library changes nothing here: the rule reads the import path.
package monkey

// Patch fakes the entry point of the real library.
func Patch(target, replacement any) {}
