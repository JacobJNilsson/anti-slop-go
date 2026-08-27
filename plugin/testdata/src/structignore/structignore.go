// Package structignore holds the fixture of the
// fullstructcomp-maxignore setting. The test builds the rule with a
// high number, so the run below reports although one comparison of the
// whole value needs six cmpopts.IgnoreFields names. The default of the
// rule is five, and it would stop the report.
package structignore

// Item is the produced value the fixture asserts on. Six of its fields
// stay outside the assertions.
type Item struct {
	Name  string
	Count int
	One   string
	Two   string
	Three string
	Four  string
	Five  string
	Six   string
}
