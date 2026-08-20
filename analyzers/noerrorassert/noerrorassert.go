// Package noerrorassert implements rule G10 of the anti-slop rule set.
// The rule rejects a type assertion and a type switch on a value of
// the predeclared error type. A caller wraps an error, and the wrapper
// answers such a test with the wrong type. errors.As and errors.Is
// walk the wrap chain, so they answer the question the code asks.
package noerrorassert

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const doc = `reject a type assertion on an error

A type assertion reads the dynamic type of one value. A caller wraps an
error with %w, and the wrapper carries another dynamic type, so the
assertion fails on the error the code looks for. errors.As and errors.Is
walk the wrap chain and give the answer the code wants.

The rule reports both forms of the assertion, "err.(T)" and
"err, ok := err.(T)", and it reports a type switch on an error. A switch
gets one diagnostic, at the guard, and never one per case.

The rule reads the static type of the operand. It reports the operand
that has the predeclared error type, and an alias of that type. It
leaves every other interface alone, which includes an interface that
embeds error, and an interface that declares Error() string under
another name. Only the error type carries the promise of a wrap chain.
A defined rename, such as "type MyError error", escapes the rule too.

An assertion to an interface is a report too, such as
"err.(interface{ Timeout() bool })". Wrapping defeats that shape the
same way. errors.As accepts a pointer to an interface variable, so the
fix holds there.

No comment stops a report. A SAFETY comment answers rule G01, and rule
G10 still reports, because the fix is errors.As and not a justification.

The rule holds no list of exempt shapes. A function that walks the wrap
chain itself gets a report too. A test file gets a report too, and the
fix costs more there: a test that asserts an exact dynamic type asks a
narrower question than errors.As answers. The author decides which
question the test asks. The rule skips generated files only.`

// Analyzer is the G10 analyzer. Consumers get it through
// antislop.Analyzers().
var Analyzer = &analysis.Analyzer{
	Name:     "noerrorassert",
	Doc:      doc,
	URL:      "https://github.com/JacobJNilsson/anti-slop-go/blob/main/docs/spec/002-rules.md",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// The three messages the rule emits. Each one names the fix that fits
// the shape it reports.
//
// Two messages carry a second clause. errors.As reads the whole chain,
// and a type switch and an interface target hold most of the code that
// reads one level on purpose. Such code needs the disable setting, and
// the clause says so. The clause is no escape that the rule reads.
const (
	assertMessage = "a wrapped error defeats this type assertion; " +
		"use errors.As with a pointer to a target variable, or errors.Is for a sentinel"
	ifaceMessage = "a wrapped error defeats this type assertion; " +
		"declare a variable of the interface type and use errors.As with a pointer to it; " +
		"code that must read exactly one level disables the rule instead"
	switchMessage = "a wrapped error defeats this type switch; " +
		"use errors.As with a pointer to a target variable, or errors.Is for a sentinel; " +
		"code that must read exactly one level disables the rule instead"
)

// errorType is the predeclared error interface. The universe holds one
// object for it, and every use of the name error names that object.
var errorType = types.Universe.Lookup("error").Type()

// CONTRACT: analysis.Analyzer.Run fixes this signature.
func run(pass *analysis.Pass) (any, error) {
	// SAFETY: inspect.Analyzer is in Requires, so the driver always
	// supplies its result, and that result is an *inspector.Inspector.
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	generated := generatedFiles(pass)

	insp.Preorder([]ast.Node{(*ast.TypeAssertExpr)(nil)}, func(n ast.Node) {
		// SAFETY: the node filter above admits this type only.
		assert := n.(*ast.TypeAssertExpr)
		if generated[pass.Fset.File(assert.Pos())] {
			return
		}
		if !isError(pass.TypesInfo.TypeOf(assert.X)) {
			return
		}
		// Report at the .( token, not at the start of the operand. The
		// operand can span several lines, and a chain such as
		// "x.(A).(T)" holds two assertions that start at the same
		// position.
		pass.Reportf(assert.Lparen, "%s", message(pass, assert))
	})
	return nil, nil
}

// message returns the diagnostic text for one assertion.
//
// The guard of a type switch holds an assertion with no target type,
// and it holds exactly one. A report on that assertion therefore gives
// one diagnostic per switch, whatever the number of cases.
func message(pass *analysis.Pass, assert *ast.TypeAssertExpr) string {
	switch {
	case assert.Type == nil:
		return switchMessage
	case namesInterface(pass.TypesInfo.TypeOf(assert.Type)):
		return ifaceMessage
	default:
		return assertMessage
	}
}

// isError reports whether a type is the predeclared error type.
//
// The test is deliberately narrow. An interface that embeds error, and
// an interface that declares Error() string under another name, both
// hold the method set of error. Neither one promises a wrap chain: only
// the error contract does, through Unwrap and the %w verb of
// fmt.Errorf. A report on such a type would name code that errors.As
// cannot always replace.
//
// types.Identical unaliases its arguments, so "type E = error" is the
// same type here, and the fixture pins that.
func isError(t types.Type) bool {
	return types.Identical(t, errorType)
}

// namesInterface reports whether the target of an assertion is an
// interface type. A type parameter names one type at each
// instantiation. It is a concrete target for the reader, so the
// concrete message fits it.
func namesInterface(t types.Type) bool {
	if _, isParam := types.Unalias(t).(*types.TypeParam); isParam {
		return false
	}
	return types.IsInterface(t)
}

// generatedFiles returns the generated files of the pass. The rule
// states the shape of hand-written code, so a report against a file
// that a program writes has no reader who can act on it.
//
// The set holds each token.File itself, not its name. A //line
// directive changes the name that token.Position reports, so a
// comparison of names can exempt the wrong file in both directions.
func generatedFiles(pass *analysis.Pass) map[*token.File]bool {
	files := make(map[*token.File]bool)
	for _, f := range pass.Files {
		if ast.IsGenerated(f) {
			files[pass.Fset.File(f.FileStart)] = true
		}
	}
	return files
}
