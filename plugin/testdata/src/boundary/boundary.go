// Package boundary is the fixture package that the boundary-packages
// setting names. Every type switch of it decodes input.
package boundary

// Kind reads the dynamic type of a value that arrives from outside the
// program.
func Kind(v any) string {
	switch v.(type) {
	case int:
		return "int"
	}

	return ""
}
