// Package store holds the same table as the suite package. It is
// production code, and the rule stays silent.
package store

type row struct{ a, b, c int }

// Rows is a table that the loop below reads.
var Rows = []row{
	{a: 1, b: 2, c: 3},
}

// Sum reads the table.
func Sum() int {
	total := 0
	for _, r := range Rows {
		total += r.a
	}

	return total
}
