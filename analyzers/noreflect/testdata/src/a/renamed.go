package a

import r "reflect" // want `this package imports reflect`

// A renamed import hides the name of the package at the use site. The
// rule reads the object that the compiler resolved, so the name of the
// import changes no answer.
func TypeName(v any) string {
	return r.TypeOf(v).String()
}
