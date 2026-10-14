// Package inlinemin3 holds the fixtures that check the min gate of the
// element path of rule G12. The rule reads test files only, so
// nothing here reports.
package inlinemin3

// Item is a non-trivial struct because it holds a nested struct field.
type Item struct {
	Name  string
	Count int
	Inner Inner
}

// Inner gives Item a nested struct field.
type Inner struct {
	Label string
}

// items returns the collection that the tests compare.
func items() []Item {
	return []Item{{Name: "a"}, {Name: "b"}, {Name: "c"}}
}
