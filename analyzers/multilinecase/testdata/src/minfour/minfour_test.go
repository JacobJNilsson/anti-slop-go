package minfour

import "testing"

func TestMin(t *testing.T) {
	tests := []row{
		{a: 1, b: 2, c: 3},
		{a: 1, b: 2, c: 3, d: 4}, // want `test case puts 4 keyed fields on one line`
	}
	for _, tt := range tests {
		_ = tt
	}
}
