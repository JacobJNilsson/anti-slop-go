// Package a holds the fixtures of rule G12. The test files of the
// package carry the assertions, and this file carries the types they
// read. The rule reads test files only, so nothing here reports.
package a

// Item is the produced value the fixtures assert on. Its graph holds
// no unexported field that cmp meets, so a report on an Item names no
// option.
type Item struct {
	Name  string
	Count int
	Inner Inner
	Cells []Cell
	Cell  *Cell
	Stamp Stamp
}

// Inner gives the fixtures a nested path, so "Inner" and "Inner.Label"
// are two fields of one base.
type Inner struct {
	Label string
}

// Cell gives the fixtures an index step and a pointer step.
type Cell struct {
	Name  string
	Count int
}

// Stamp holds an unexported field and an Equal method. cmp reads the
// method and walks no further, as it does for time.Time, so a report on
// a value that holds a Stamp names no option.
type Stamp struct {
	ticks int
}

// Equal compares two stamps.
func (s Stamp) Equal(other Stamp) bool { return s.ticks == other.ticks }

// Secret holds an unexported field and no Equal method. cmp.Diff panics
// on such a field, so a report on a Secret names cmp.AllowUnexported.
type Secret struct {
	Name   string
	Count  int
	hidden int
}

// Nest holds one field of each container shape. The walk reads the
// pointer, the slice, and the array of a type with no unexported field,
// and it meets the unexported field through the map.
type Nest struct {
	// Equal is a field and no method, so it answers no comparison.
	Equal string
	Ptr   *Cell
	Slice []Cell
	Array [2]Cell
	// Kind is a defined type with no field at all, and Anon is a
	// structure that no declaration names.
	Kind Label
	Anon struct {
		Tag string
	}
	Map map[string]Leaf
}

// Label is a defined type that holds no field.
type Label string

// Leaf holds the unexported field that the walk of a Nest meets.
type Leaf struct {
	Name   string
	hidden int
}

// Sealed holds an unexported field, and its Equal method takes a
// pointer. cmp reads the method for a pointer to a Sealed, and it reads
// no method for a Sealed value, because a value holds the methods of
// its value receivers only.
type Sealed struct {
	hidden int
}

// Equal compares two sealed values through pointers.
func (s *Sealed) Equal(other *Sealed) bool { return s.hidden == other.hidden }

// Loose holds an unexported field, and its Equal method takes a value
// and has a pointer receiver. The method set of a Loose value holds no
// such method, so cmp reads the fields of the value.
type Loose struct {
	hidden int
}

// Equal compares a pointer to a loose value against a loose value.
func (l *Loose) Equal(other Loose) bool { return l.hidden == other.hidden }

// LooseValue holds a Loose by value. cmp calls no method for that
// field, so it meets the unexported field under it.
type LooseValue struct {
	Name  string
	Count int
	Field Loose
}

// Mismatch holds an unexported field, and its Equal method takes
// another type. cmp calls no such method.
type Mismatch struct {
	hidden int
}

// Equal carries the name of the method that cmp reads, and another
// parameter.
func (m Mismatch) Equal(ticks int) bool { return m.hidden == ticks }

// SealedValue holds a Sealed by value. cmp reads no method for that
// field, so it meets the unexported field under it.
type SealedValue struct {
	Name  string
	Count int
	Field Sealed
}

// SealedPointer holds a pointer to a Sealed. cmp reads the method of
// the pointer type and walks no further.
type SealedPointer struct {
	Name  string
	Count int
	Field *Sealed
}

// MismatchValue holds a value whose Equal method cmp cannot call.
type MismatchValue struct {
	Name  string
	Count int
	Field Mismatch
}

// Matcher carries the name of the equality family on a type of this
// project. The name of a method is no evidence of the module that
// declares it.
type Matcher struct{}

// Equal carries the name of the testify assertion.
func (Matcher) Equal(other any) bool { return other == nil }

// cell returns a value through a call. A call breaks the chain of a
// selector, so the receiver of this method is no base.
func (i Item) cell() Cell { return Cell{Name: i.Name, Count: i.Count} }

// build returns the value the fixtures assert on.
func build() Item {
	return Item{Name: "boot", Count: 3, Inner: Inner{Label: "run"}, Cell: &Cell{}}
}

// secret returns a value whose type holds an unexported field.
func secret() Secret { return Secret{Name: "boot", Count: 3, hidden: 1} }

// nest returns a value whose type graph holds an unexported field.
func nest() Nest { return Nest{Equal: "boot", Ptr: &Cell{}} }

// sealedValue returns a value that holds a Sealed by value.
func sealedValue() SealedValue { return SealedValue{Name: "boot", Count: 3} }

// sealedPointer returns a value that holds a pointer to a Sealed.
func sealedPointer() SealedPointer {
	return SealedPointer{Name: "boot", Count: 3, Field: &Sealed{}}
}

// mismatchValue returns a value whose Equal method takes another type.
func mismatchValue() MismatchValue { return MismatchValue{Name: "boot", Count: 3} }

// looseValue returns a value whose field carries a method that only a
// pointer to it holds.
func looseValue() LooseValue { return LooseValue{Name: "boot", Count: 3} }
