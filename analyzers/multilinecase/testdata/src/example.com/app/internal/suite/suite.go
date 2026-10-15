// Package suite holds a shared test table. No file of the package
// carries a name that ends in _test.go. The test-packages setting names
// it, so the rule reads the table below.
package suite

type row struct{ a, b, c int }

// Rows is a table that the loop below reads.
var Rows = []row{
	{a: 1, b: 2, c: 3}, // want `test case puts 3 keyed fields on one line`
}

// Sum reads the table.
func Sum() int {
	total := 0
	for _, r := range Rows {
		total += r.a
	}

	return total
}
