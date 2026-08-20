package a

import . "reflect" // want `this package imports reflect`

// A dot import puts every name of the package in the file scope, so no
// use site names reflect at all. The rule reads the objects that the
// file uses, so it reads such a use too.
func KindOf(v any) Kind {
	return ValueOf(v).Kind()
}
