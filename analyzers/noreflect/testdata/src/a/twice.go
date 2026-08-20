package a

import (
	"reflect"   // want `this package imports reflect`
	r "reflect" // want `this package imports reflect`
)

// A file may import one package twice under two names. The import is
// the decision, so each specification carries its own report.
func SameType(v any) bool {
	return reflect.TypeOf(v) == r.TypeOf(v)
}
