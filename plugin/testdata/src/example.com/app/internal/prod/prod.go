// Package prod is production code that the fixture of rule G08
// rewires. No pattern of the setting names this package.
package prod

// Send holds the transport of the package, in a package-level function
// variable.
var Send = func(message string) error { return nil }
