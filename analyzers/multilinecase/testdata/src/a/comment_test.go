package a

// A case with a comment between its braces gets the report and no fix,
// because the edit could move the comment to the wrong field.
var commented = []item{
	{name: "line", a: 1, /* why */ b: 2}, // want `test case puts 3 keyed fields on one line`
}

func readCommented() {
	for _, tt := range commented {
		_ = tt
	}
}
