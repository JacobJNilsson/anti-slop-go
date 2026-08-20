// Package ingest is the decode boundary of the fixture project. A
// boundary pattern names it, so every type switch of it stays clean.
package ingest

// Decode reads the dynamic type of a value that arrives from outside
// the program. Such a read is the work of a boundary package.
func Decode(v any) string {
	switch v.(type) {
	case int:
		return "int"
	case string:
		return "string"
	}

	return ""
}
