// Package ignore0 holds the fixtures of a maxignore setting of zero.
// The rule then reports a group whose fix carries no ignore name at
// all, which means the assertions name every field of the value.
package ignore0

// Pair holds the two fields that the assertions name, so one comparison
// of the whole value needs no ignore name.
type Pair struct {
	Name  string
	Count int
}

// Trio holds one field more than the assertions name, so the fix needs
// one ignore name and a setting of zero rejects it.
type Trio struct {
	Name  string
	Count int
	State string
}

// pair returns the value the first fixture asserts on.
func pair() Pair { return Pair{Name: "boot", Count: 3} }

// trio returns the value the second fixture asserts on.
func trio() Trio { return Trio{Name: "boot", Count: 3, State: "new"} }
