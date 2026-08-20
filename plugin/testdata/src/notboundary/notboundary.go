// Package notboundary is the fixture package that no pattern of the
// setting names.
package notboundary

// Kind reads the dynamic type of a value inside the program.
func Kind(v any) string {
	switch v.(type) { // want `this type switch reads the dynamic type of an any value`
	case int:
		return "int"
	}

	return ""
}
