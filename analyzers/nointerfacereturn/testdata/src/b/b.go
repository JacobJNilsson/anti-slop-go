// Package b holds the imported types that fixture package a uses. A
// type of this package is external to package a, so package a cannot
// change the signatures that b declares.
package b

// Item is the interface that a build function returns.
type Item interface {
	Name() string
}

// Factory declares Build with an interface result, so an implementation
// cannot change that result. It declares a second method as well, so a
// type with Build alone does not implement it.
type Factory interface {
	Build() Item
	Kind() string
}

// Thing is a second interface result, so a method of Splitter carries
// two of them.
type Thing interface {
	Size() int
}

// Splitter declares two interface results in one method. An
// implementation cannot narrow either one, so the rule must read the
// declared result at the position it judges.
type Splitter interface {
	Split() (Item, Thing)
}

// Register takes a build function. A call is not evidence the analyzer
// can read, so package a must justify such a function with a comment.
func Register(build func() Item) {}
