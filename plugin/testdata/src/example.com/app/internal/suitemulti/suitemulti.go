// Package suitemulti is the fixture of rule G15 for the test-packages
// setting. The setting names the package, so the rule reads the table
// below and reports its case.
package suitemulti

type row struct{ a, b, c int }

// Rows is a table that Sum reads.
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
