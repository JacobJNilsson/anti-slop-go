// Package a holds the fixtures of rule G15. This production file holds
// a table that a loop reads, and the rule reads test files only.
package a

// inner is a struct that a case holds as one field value.
type inner struct{ x, y, z int }

// item is the type of every table case in the fixtures.
type item struct {
	name    string
	a, b, c int
	in      inner
}

// holder gives a selector expression for a range statement.
type holder struct{ rows []item }

var prodRows = []item{{name: "p", a: 1, b: 2, c: 3}}

func sumProd() int {
	total := 0
	for _, r := range prodRows {
		total += r.a
	}

	return total
}

func load() []item { return nil }

func take(rows []item) int { return len(rows) }
