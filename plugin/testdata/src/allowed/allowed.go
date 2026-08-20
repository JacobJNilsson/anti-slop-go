// Package allowed is the fixture package that the reflect-allow
// setting names. Every file of it may import reflect.
package allowed

import "reflect"

// Kind reads the kind of a value at run time.
func Kind(v any) reflect.Kind {
	return reflect.ValueOf(v).Kind()
}
