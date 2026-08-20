// Package prod holds production files only. No test file of it imports
// reflect, so the rule needs no look at the objects the package uses.
package prod

import "reflect" // want `this package imports reflect`

// Describe reads the type of a value while the program runs.
func Describe(v any) string {
	return reflect.TypeOf(v).String()
}
