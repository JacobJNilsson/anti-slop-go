// Package codec is the boundary package that a project allows. Every
// file of it may import reflect, in a test file and in a production
// file alike.
package codec

import "reflect"

// Fields returns the name of every field of a struct value.
func Fields(v any) []string {
	t := reflect.TypeOf(v)
	names := make([]string, 0, t.NumField())
	for i := range t.NumField() {
		names = append(names, t.Field(i).Name)
	}
	return names
}
