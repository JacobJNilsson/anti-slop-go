// Package onefield holds the types of the fixtures of the second
// report of rule G12.
package onefield

// Refusal has two exported fields.
type Refusal struct {
	StatusCode int
	Body       string
}

// Single has one exported field and one unexported field.
type Single struct {
	Name string
	note string
}

// Triple has three exported fields.
type Triple struct {
	A, B, C int
}

// Inner is a nested value.
type Inner struct {
	Code int
	Text string
}

// Nested holds a value with fields of its own.
type Nested struct {
	Body  Inner
	Extra string
}

// Hidden has one exported field and one unexported field.
type Hidden struct {
	Code int
	note string
}

// Equaler declares the Equal method that cmp calls.
type Equaler struct {
	Code int
	Note string
}

// Equal compares the code only.
func (e Equaler) Equal(o Equaler) bool { return e.Code == o.Code }

// Refuse returns a value to compare.
func Refuse() *Refusal { return &Refusal{} }


// TwinA and TwinB have the same fields and are two types.
type TwinA struct{ X, Y int }

// TwinB has the fields of TwinA.
type TwinB struct{ X, Y int }

// Base is embedded to promote its field.
type Base struct{ Code int }

// Derived promotes the field Code of Base.
type Derived struct {
	Base
	Text string
}
