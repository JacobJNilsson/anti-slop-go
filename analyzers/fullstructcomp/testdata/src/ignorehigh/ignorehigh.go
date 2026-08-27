// Package ignorehigh holds the fixture of a high maxignore setting. A
// project that wants the report volume of the rule before the gate sets
// a high number, and the group below then reports although its fix
// needs six ignore names.
package ignorehigh

// Record holds six fields that the assertions leave out, so one
// comparison of the whole value needs six ignore names.
type Record struct {
	Name  string
	Count int
	One   string
	Two   string
	Three string
	Four  string
	Five  string
	Six   string
}

// build returns the value the fixture asserts on.
func build() Record { return Record{Name: "boot", Count: 3} }
