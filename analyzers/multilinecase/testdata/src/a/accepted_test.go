package a

import "testing"

func TestTwoFields(t *testing.T) {
	tests := []item{
		{name: "two", a: 1},
	}
	for _, tt := range tests {
		_ = tt
	}
}

func TestMultiLineCase(t *testing.T) {
	tests := []item{
		{name: "multi", a: 1, b: 2,
			c: 3},
	}
	for _, tt := range tests {
		_ = tt
	}
}

func TestPositionalCase(t *testing.T) {
	tests := []item{
		{"positional", 1, 2, 3, inner{}},
	}
	for _, tt := range tests {
		_ = tt
	}
}

func TestMixedCase(t *testing.T) {
	tests := []item{
		{"mixed", 1, 2, 3, inner{x: 1, y: 2, z: 3}},
	}
	for _, tt := range tests {
		_ = tt
	}
}

func TestTableNeverRanged(t *testing.T) {
	rows := []item{{name: "setup", a: 1, b: 2, c: 3}}
	if take(rows) != 1 {
		t.Fatal("want one row")
	}
}

func TestRangeNamesNoDeclaration(t *testing.T) {
	h := holder{rows: []item{{name: "held", a: 1, b: 2, c: 3}}}
	grid := [][]item{{{name: "grid", a: 1, b: 2, c: 3}}}
	for _, tt := range h.rows {
		_ = tt
	}
	for _, tt := range load() {
		_ = tt
	}
	for _, tt := range grid[0] {
		_ = tt
	}
	for _, tt := range grid[0][0:1] {
		_ = tt
	}
}

func TestNestedLiteralInsideAMultiLineCase(t *testing.T) {
	tests := []item{
		{
			name: "nested",
			in:   inner{x: 1, y: 2, z: 3},
		},
	}
	for _, tt := range tests {
		_ = tt
	}
}

func TestElementsThatAreNoStructs(t *testing.T) {
	maps := []map[string]int{{"a": 1, "b": 2, "c": 3}}
	ints := []int{1, 2, 3}
	for _, m := range maps {
		_ = m
	}
	for _, n := range ints {
		_ = n
	}
}

func TestLaterAssignmentIsNoTable(t *testing.T) {
	rows := load()
	for _, tt := range rows {
		_ = tt
	}
	rows = []item{{name: "later", a: 1, b: 2, c: 3}}
	_ = rows
}

func TestCallResultNamesNoTable(t *testing.T) {
	rows, n := load(), 3
	for _, tt := range rows {
		_, _ = tt, n
	}
}

func TestRangeOverAnInteger(t *testing.T) {
	for i := range 3 {
		_ = i
	}
}

func TestRangeOverAParameter(t *testing.T) {
	check := func(rows []item) {
		for _, tt := range rows {
			_ = tt
		}
	}
	check(nil)
}

func TestRangeOverAConstant(t *testing.T) {
	const n = 3
	for i := range n {
		_ = i
	}
}

func lookup() ([]item, bool) { return nil, false }

func TestMultiValueCallNamesNoTable(t *testing.T) {
	rows, ok := lookup()
	for _, tt := range rows {
		_, _ = tt, ok
	}
}
