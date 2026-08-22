// Package min1 holds the fixture of the singular form of the message.
// 002 states that a setting of one reports every spot check, and that
// no project wants it. The message must read as a sentence there.
package min1

// Item is the produced value the fixture asserts on.
type Item struct {
	Name  string
	Count int
}

// build returns the value the fixture asserts on.
func build() Item { return Item{Name: "boot", Count: 3} }
