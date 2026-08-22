// Package min3 holds the fixtures of the min setting. The rule reads
// this package with an instance that asks for three fields, so the
// checklist of two fields stays clean here.
package min3

// Item is the produced value the fixtures assert on.
type Item struct {
	Name  string
	Count int
	Label string
}

// build returns the value the fixtures assert on.
func build() Item { return Item{Name: "boot", Count: 3, Label: "run"} }
