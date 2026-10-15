package a

import "testing"

// The fix of rule G15 puts each field on its own line. analysistest
// formats the golden file, so the golden holds the text the fix writes
// and gofmt aligns it.

func TestLocalTable(t *testing.T) {
	tests := []item{
		{name: "two", a: 1},
		{name: "three", a: 1, b: 2}, // want `test case puts 3 keyed fields on one line; put each field on its own line`
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { _ = tt })
	}
}

func TestVarTable(t *testing.T) {
	var tests = []item{
		{name: "four", a: 1, b: 2, c: 3}, // want `test case puts 4 keyed fields on one line; put each field on its own line`
	}
	for _, tt := range tests {
		_ = tt
	}
}

func TestSharedTable(t *testing.T) {
	for _, tt := range shared {
		_ = tt
	}
}

func TestInlineRange(t *testing.T) {
	for _, tt := range []item{{name: "inline", a: 1, b: 2}} { // want `test case puts 3 keyed fields on one line`
		_ = tt
	}
}

func TestMapTable(t *testing.T) {
	tests := map[string]item{
		"first": {a: 1, b: 2, c: 3}, // want `test case puts 3 keyed fields on one line`
	}
	for name, tt := range tests {
		_, _ = name, tt
	}
}

func TestPointerCases(t *testing.T) {
	tests := []*item{
		{a: 1, b: 2, c: 3},          // want `test case puts 3 keyed fields on one line`
		&item{a: 4, b: 5, c: 6},     // want `test case puts 3 keyed fields on one line`
		(&item{a: 7, b: 8, c: 9}),   // want `test case puts 3 keyed fields on one line`
		{a: 1, b: 2, c: 3, in: inner{x: 1, y: 2, z: 3}}, // want `test case puts 4 keyed fields on one line`
	}
	for _, tt := range tests {
		_ = tt
	}
}

func TestTrailingComma(t *testing.T) {
	tests := []item{
		{a: 1, b: 2, c: 3,}, // want `test case puts 3 keyed fields on one line`
	}
	for _, tt := range tests {
		_ = tt
	}
}

func TestTwoLoopsOneTable(t *testing.T) {
	tests := []item{
		{a: 1, b: 2, c: 3}, // want `test case puts 3 keyed fields on one line`
	}
	for _, tt := range tests {
		_ = tt
	}
	for _, tt := range (tests) {
		_ = tt
	}
}

func TestParenthesizedRange(t *testing.T) {
	for _, tt := range ([]item{{a: 1, b: 2, c: 3}}) { // want `test case puts 3 keyed fields on one line`
		_ = tt
	}
}

func TestPairedNames(t *testing.T) {
	tests, limit := []item{{a: 1, b: 2, c: 3}}, 3 // want `test case puts 3 keyed fields on one line`
	for _, tt := range tests {
		_, _ = tt, limit
	}
}

func TestIndexedArray(t *testing.T) {
	tests := [...]item{
		1: {a: 1, b: 2, c: 3}, // want `test case puts 3 keyed fields on one line`
	}
	for _, tt := range tests {
		_ = tt
	}
}

func TestIndentedCase(t *testing.T) {
	t.Run("nested", func(t *testing.T) {
		for _, tt := range []item{
			{a: 1, b: 2, c: 3}, // want `test case puts 3 keyed fields on one line`
		} {
			_ = tt
		}
	})
}
