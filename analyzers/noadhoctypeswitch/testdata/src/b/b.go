// Package b holds the imported types that fixture package a uses. A
// type of this package is external to package a, so package a cannot
// change the signatures that b declares.
package b

// Decoder declares a method with an empty interface parameter. An
// implementation of Decoder keeps that parameter, so the interface is
// the evidence that rule G03 reads and rule G06 reads with it.
type Decoder interface {
	Decode(value any) error
	Format() string
}

// Tagger declares the empty interface at position two, so the position
// of a parameter decides whether the contract covers it.
type Tagger interface {
	Tag(name string, value any)
	Format() string
}

// Register takes a handler. A call is not evidence the analyzer can
// read, so package a must justify such a handler with a comment.
func Register(handle func(v any)) {}
