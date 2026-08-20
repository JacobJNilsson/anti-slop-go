// Package notallowed is the fixture package that no pattern of the
// setting names.
package notallowed

import "reflect" // want `this package imports reflect`

// Kind reads the kind of a value at run time.
func Kind(v any) reflect.Kind {
	return reflect.ValueOf(v).Kind()
}
