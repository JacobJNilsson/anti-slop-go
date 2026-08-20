package a

// A generator such as goyacc points the reader back at its input with
// a //line directive. This file carries no generated header, so the
// rule reads it. The message must name the line that the directive
// gives, because the driver prints that same line in the header of
// the diagnostic.
//line input.go:100

func LinedWiden(u User) User {
	v := any(u)
	return v.(User) // want `this assertion takes back a value that v widens from User to any at line 102; remove the widening`
}
