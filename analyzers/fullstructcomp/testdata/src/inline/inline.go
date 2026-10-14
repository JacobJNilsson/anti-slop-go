// Package inline holds the fixtures of the collection paths of rule
// G12. The inline path reads a composite literal of a slice or array
// of non-trivial structs. The element path reads indexed accesses on
// a collection variable. The rule reads test files only, so nothing
// here reports.
package inline

import "time"

// Item is a non-trivial struct: it holds a nested struct field. A
// composite literal of []Item carries nested braces, and the failure
// output of require.Equal is hard to read.
type Item struct {
	Name  string
	Count int
	Inner Inner
}

// Inner gives Item a nested struct field.
type Inner struct {
	Label string
}

// Flat is a trivial struct: two scalar fields and no nesting. A
// composite literal of []Flat is short and readable.
type Flat struct {
	Name  string
	Count int
}

// Wide is a non-trivial struct: five exported scalar fields, no
// nesting. The field count alone crosses the threshold.
type Wide struct {
	Name     string
	Count    int
	State    string
	Kind     string
	Priority int
}

// Tiny is a trivial struct: one exported field.
type Tiny struct {
	Name string
}

// Secret holds an unexported field beside two exported ones. It is
// trivial by itself (2 exported fields, no nesting), so it does not
// trigger the inline path alone.
type Secret struct {
	Name   string
	Count  int
	hidden int
}

// WithSecret is a non-trivial struct that nests a Secret. A
// comparison of a []WithSecret needs cmp.AllowUnexported because the
// walk meets the unexported field of Secret.
type WithSecret struct {
	Label  string
	Detail Secret
}

// Stamped holds three exported fields, and one of them is a time.Time.
// The type has an Equal method, so cmp reads it as a value and the
// field is not a nested struct. Stamped is trivial.
type Stamped struct {
	Name  string
	Count int
	At    time.Time
}

// StampedWide holds four exported fields, so the size condition makes
// it non-trivial although no field is a nested struct.
type StampedWide struct {
	Name  string
	Count int
	Kind  string
	At    time.Time
}

// Hidden holds a struct in an unexported field. Only exported fields
// count, so Hidden is trivial.
type Hidden struct {
	Name  string
	Count int
	inner Inner
}

// Items is a named slice type of non-trivial structs.
type Items []Item

// items returns the collection the fixtures assert on.
func items() []Item {
	return []Item{
		{Name: "a", Count: 1, Inner: Inner{Label: "x"}},
		{Name: "b", Count: 2, Inner: Inner{Label: "y"}},
	}
}

// wideItems returns a collection of wide structs.
func wideItems() []Wide {
	return []Wide{
		{Name: "a", Count: 1, State: "new", Kind: "order", Priority: 1},
	}
}

// secretItems returns a collection that holds an unexported field.
func secretItems() []WithSecret {
	return []WithSecret{
		{Label: "a", Detail: Secret{Name: "x", Count: 1}},
	}
}

// arrayItems returns an array of items.
func arrayItems() [2]Item {
	return [2]Item{
		{Name: "a", Count: 1, Inner: Inner{Label: "x"}},
		{Name: "b", Count: 2, Inner: Inner{Label: "y"}},
	}
}
