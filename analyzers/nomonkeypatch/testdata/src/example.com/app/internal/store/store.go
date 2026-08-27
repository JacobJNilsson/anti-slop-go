// Package store holds the same assignment as the suite package beside
// it. No pattern of the setting names this package, so it stays
// production code and the rule reads no line of it.
package store

import "other"

// Install rewires production code from production code. The rule reads
// test files only, so this assignment stands.
func Install() {
	other.Send = func(message string) error { return nil }
}
