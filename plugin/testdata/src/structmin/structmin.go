// Package structmin holds the fixture of the fullstructcomp-min
// setting. The test builds the rule with a setting of three fields, so
// the run of two fields below stays clean and the run of three reports.
package structmin

// Item is the produced value the fixture asserts on.
type Item struct {
	Name  string
	Count int
	Label string
}
