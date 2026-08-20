// This file is the second hand-written file of the fixture package. It
// exists to pin the per-file lookup of the comment index. A rule reads
// the comments of the file that holds the assertion, and of no other
// file. An index that answers from one file for the whole package
// accepts the marker of the wrong file. The fixtures of a single-file
// package cannot see that error.
//
// Both directions need a pair, because the file order of the pass is
// not fixed:
//
//   - line 22 below holds an assertion with no justification. The
//     marker of a.go ends on line 21, one line above it.
//   - line 24 of a.go holds an assertion with no justification. The
//     marker on line 23 below ends one line above it.
//
// Both assertions must be reported. Keep the four lines aligned.

package a

func CrossFileFromB(x any) {

	a := x.(T) // want "no SAFETY justification"
	// SAFETY: this marker belongs to b.go alone. It ends on line 23.
	_ = a
}
