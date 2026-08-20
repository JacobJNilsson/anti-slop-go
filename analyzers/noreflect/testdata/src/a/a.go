package a

import "reflect" // want `this package imports reflect`

// Describe reads the type of a value while the program runs. The
// compiler knew that type at the call site.
func Describe(v any) string {
	return reflect.TypeOf(v).String()
}
