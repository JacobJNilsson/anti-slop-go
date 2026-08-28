package a

type T struct{ N int }

func (T) M() {}

type A interface{ M() }

var store any

func consume(T) {}

func identity(v any) any { return v }

// The cross-file fixtures pin the per-file lookup: a marker in one file
// must never justify an assertion in another file. The lines of a.go
// and b.go align on purpose. Keep them aligned, or the pair proves
// nothing.

// SAFETY: this marker belongs to a.go alone. It ends on line 21, one
// line above the assertion of b.go, and it justifies nothing there.

func CrossFileFromA(x any) {
	a := x.(T) // want "no justification comment"
	_ = a
}

func Bare(x any) {
	a := x.(T) // want "type assertion has no justification comment; state the checked invariant in a comment directly above it, or use the comma-ok form"
	_ = a
}

func CommaOK(x any) {
	a, ok := x.(T)
	_, _ = a, ok
}

func CommaOKInIf(x any) {
	if a, ok := x.(T); ok {
		consume(a)
	}
}

func CommaOKParenthesized(x any) {
	a, ok := (x.(T))
	_, _ = a, ok
}

func CommaOKVarSpec(x any) {
	var a, ok = x.(T)
	_, _ = a, ok
}

func TypeSwitch(x any) {
	switch v := x.(type) {
	case T:
		consume(v)
	}
}

func BlankAssign(x any) {
	_ = x.(T) // want "no justification comment"
}

func CallArgument(x any) {
	consume(x.(T)) // want "no justification comment"
}

func VarSpec(x any) {
	var a = x.(T) // want "no justification comment"
	_ = a
}

func TypedVarSpec(x any) {
	var a T = x.(T) // want "no justification comment"
	_ = a
}

func SafetyAboveTheStatement(x any) {
	// SAFETY: only T values reach this function.
	a := x.(T)
	_ = a
}

func SafetyAboveTheEnclosingIf(x any) {
	// SAFETY: only T values reach this function.
	if a := x.(T); a.N > 0 {
		consume(a)
	}
}

func SafetyAboveTheCaseClause(x any, kind int) {
	switch kind {
	// SAFETY: the caller pairs kind 1 with a T value.
	case 1:
		a := x.(T)
		_ = a
	}
}

func SafetyAboveAMultiStatementCaseClause(x any, kind int) {
	switch kind {
	// SAFETY: the caller pairs kind 1 with a T value.
	case 1:
		first := x.(T)
		later := x.(T) // want "no justification comment"
		consume(first)
		consume(later)
	}
}

func SafetyAboveAMultiStatementCommClause(x any, ch chan int) {
	select {
	// SAFETY: the caller fills ch after it stores a T value.
	case <-ch:
		first := x.(T)
		later := x.(T) // want "no justification comment"
		consume(first)
		consume(later)
	}
}

func ChainedAssertions(x any) {
	a := x.(A).(T) // want "no justification comment" "no justification comment"
	_ = a
}

func SafetyAboveAChain(x any) {
	// SAFETY: only T values reach this function.
	a := x.(A).(T)
	_ = a
}

func SafetyInAMultiLineGroup(x any) {
	// SAFETY: the loader checks the payload before it stores the value.
	// Only T values reach this function.
	a := x.(T)
	_ = a
}

func SafetyInABlockComment(x any) {
	/* SAFETY : the marker accepts a space before the colon. */
	a := x.(T)
	_ = a
}

func SafetyOnTheSecondLineOfAGroup(x any) {
	// The loader checks the payload before it stores the value.
	// SAFETY: only T values reach this function.
	a := x.(T)
	_ = a
}

func SafetyInABlockWithAStarGutter(x any) {
	/*
	 * SAFETY: only T values reach this function.
	 */
	a := x.(T)
	_ = a
}

func PlainCommentAboveTheAssertion(x any) {
	// Only T values reach this function. The text carries no marker
	// word, and the rule accepts it.
	a := x.(T)
	_ = a
}

func PlainCommentInAListItem(x any) {
	// - only T values reach this function.
	a := x.(T)
	_ = a
}

func BlankLineBelowTheComment(x any) {
	// SAFETY: a blank line breaks the link to the statement.

	a := x.(T) // want "no justification comment"
	_ = a
}

func SafetyTrailingThePreviousStatement(x any) {
	consume(T{}) // SAFETY: this comment trails the statement beside it.
	a := x.(T)   // want "no justification comment"
	_ = a
}

func SafetyOnTheSameLine(x any) {
	a := x.(T) // SAFETY: a trailing comment is not above the assertion. // want "no justification comment"
	_ = a
}

func ParenthesizedOperand(x any) {
	a := (x).(T) // want "no justification comment"
	_ = a
}

func ParenthesizedAssertion(x any) {
	a := (x.(T)) // want "no justification comment"
	_ = a
}

func SafetyAboveAParenthesizedOperand(x any) {
	// SAFETY: only T values reach this function.
	a := (x).(T)
	_ = a
}

func MultiLineOperand(x any) {
	a := identity(
		x,
	).(T) // want "no justification comment"
	_ = a
}

func SafetyAboveTheAssertionOfAMultiLineOperand(x any) {
	a := identity(
		x,
		// SAFETY: identity returns the value it receives.
	).(T)
	_ = a
}

func SafetyInsideAFuncLiteral(x any) {
	// SAFETY: this comment justifies the go statement, not the body.
	go func() {
		a := x.(T) // want "no justification comment"
		_ = a
	}()
}

var packageLevelBare = store.(T) // want "no justification comment"

// SAFETY: init seeds store with a T value.
var packageLevelJustified = store.(T)

// SAFETY: this comment sits above the block, not above a spec of it.
var (
	blockScopedBare = store.(T) // want "no justification comment"

	// SAFETY: init seeds store with a T value.
	blockScopedJustified = store.(T)
)
