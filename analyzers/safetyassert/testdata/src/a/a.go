package a

type T struct{ N int }

func (T) M() {}

type A interface{ M() }

var store any

func consume(T) {}

func identity(v any) any { return v }

func Bare(x any) {
	a := x.(T) // want "type assertion has no SAFETY justification; state the checked invariant in a SAFETY: comment directly above it, or use the comma-ok form"
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
	_ = x.(T) // want "no SAFETY justification"
}

func CallArgument(x any) {
	consume(x.(T)) // want "no SAFETY justification"
}

func VarSpec(x any) {
	var a = x.(T) // want "no SAFETY justification"
	_ = a
}

func TypedVarSpec(x any) {
	var a T = x.(T) // want "no SAFETY justification"
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
		later := x.(T) // want "no SAFETY justification"
		consume(first)
		consume(later)
	}
}

func SafetyAboveAMultiStatementCommClause(x any, ch chan int) {
	select {
	// SAFETY: the caller fills ch after it stores a T value.
	case <-ch:
		first := x.(T)
		later := x.(T) // want "no SAFETY justification"
		consume(first)
		consume(later)
	}
}

func ChainedAssertions(x any) {
	a := x.(A).(T) // want "no SAFETY justification" "no SAFETY justification"
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

func MarkerWithoutAColon(x any) {
	// SAFETY needs a colon, so this text is not a justification.
	a := x.(T) // want "no SAFETY justification"
	_ = a
}

func BlankLineBelowTheComment(x any) {
	// SAFETY: a blank line breaks the link to the statement.

	a := x.(T) // want "no SAFETY justification"
	_ = a
}

func SafetyTrailingThePreviousStatement(x any) {
	consume(T{}) // SAFETY: this comment trails the statement beside it.
	a := x.(T)   // want "no SAFETY justification"
	_ = a
}

func SafetyOnTheSameLine(x any) {
	a := x.(T) // SAFETY: a trailing comment is not above the assertion. // want "no SAFETY justification"
	_ = a
}

func ParenthesizedOperand(x any) {
	a := (x).(T) // want "no SAFETY justification"
	_ = a
}

func ParenthesizedAssertion(x any) {
	a := (x.(T)) // want "no SAFETY justification"
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
	).(T) // want "no SAFETY justification"
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
		a := x.(T) // want "no SAFETY justification"
		_ = a
	}()
}

var packageLevelBare = store.(T) // want "no SAFETY justification"

// SAFETY: init seeds store with a T value.
var packageLevelJustified = store.(T)

// SAFETY: this comment sits above the block, not above a spec of it.
var (
	blockScopedBare = store.(T) // want "no SAFETY justification"

	// SAFETY: init seeds store with a T value.
	blockScopedJustified = store.(T)
)
