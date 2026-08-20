// Package safetyassert implements rule G01 of the anti-slop rule set:
// a single-result type assertion must carry a SAFETY comment that states
// the invariant which makes the panic unreachable.
package safetyassert

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/JacobJNilsson/anti-slop-go/internal/signature"
)

const doc = `require a SAFETY comment for panicking type assertions

A single-result type assertion v.(T) panics when the value is not a T.
The author must state the invariant that makes the panic unreachable, in
a comment that matches SAFETY: and sits directly above the assertion or
above the statement that contains it. The comma-ok form is checked code
and needs no comment. The rule skips generated files.`

// Analyzer reports single-result type assertions that carry no SAFETY
// justification. It implements rule G01; see docs/spec/002-rules.md.
var Analyzer = &analysis.Analyzer{
	Name:     "safetyassert",
	Doc:      doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// message is the single diagnostic this rule emits. It names the problem
// and both fixes.
const message = "type assertion has no SAFETY justification; " +
	"state the checked invariant in a SAFETY: comment directly above it, " +
	"or use the comma-ok form"

// safetyMarker is the marker word of rule G01. Package signature holds
// the contract that the three markers of
// docs/spec/003-implementation.md share.
const safetyMarker = "SAFETY"

// CONTRACT: analysis.Analyzer.Run fixes this signature.
func run(pass *analysis.Pass) (any, error) {
	// SAFETY: inspect.Analyzer is in Requires, so the driver puts its
	// result type in ResultOf before it calls this function.
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	justifications := signature.NewJustifications(pass, safetyMarker)

	insp.WithStack([]ast.Node{(*ast.TypeAssertExpr)(nil)}, func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push {
			return true
		}
		// SAFETY: the node filter above admits this type only.
		assert := n.(*ast.TypeAssertExpr)
		if assert.Type == nil {
			return true // x.(type) heads a type switch; rule G06 covers it.
		}
		// The comma-ok form records a tuple. A missing entry gives a nil
		// type, and the assertion below then reports false.
		if _, isTuple := pass.TypesInfo.Types[assert].Type.(*types.Tuple); isTuple {
			return true
		}
		if justifications.Generated(assert.Pos()) {
			return true
		}
		if justifications.MarkedAbove(assert.Pos(), candidateLines(pass.Fset, assert, stack)) {
			return true
		}
		// Report at the .( token, not at the start of the operand: in a
		// chain x.(A).(T) both assertions start at x, and one position
		// then carries two identical diagnostics.
		pass.Reportf(assert.Lparen, "%s", message)
		return true
	})
	return nil, nil
}

// candidateLines returns the lines a SAFETY comment may end directly
// above: the line of the assertion, the line of its .( token for a
// multi-line operand, and the lines of the statements that contain it.
func candidateLines(fset *token.FileSet, assert *ast.TypeAssertExpr, stack []ast.Node) []int {
	lines := []int{signature.LineOf(fset, assert.Pos()), signature.LineOf(fset, assert.Lparen)}
	for _, stmt := range enclosingStmts(stack) {
		lines = append(lines, signature.LineOf(fset, stmt.Pos()))
	}
	return lines
}

// enclosingStmts returns the innermost statement that holds the node and
// the outermost statement below the enclosing block. The second one
// accepts a comment above an `if` whose init statement holds the
// assertion. A package-level declaration has no enclosing statement, so
// the result is empty.
func enclosingStmts(stack []ast.Node) []ast.Stmt {
	var inner, outer ast.Stmt
	for i := len(stack) - 1; i >= 0; i-- {
		if _, isBlock := stack[i].(*ast.BlockStmt); isBlock {
			break
		}
		stmt, isStmt := stack[i].(ast.Stmt)
		if !isStmt {
			continue
		}
		// A case or a communication clause holds a statement list, not a
		// block, so the block test above does not stop the walk. The
		// clause line is the line above the first statement of the list
		// only. A later statement in the same clause needs its own
		// comment.
		if body, isClause := clauseBody(stmt); isClause {
			if len(body) == 0 || body[0] != outer {
				break
			}
		}
		if inner == nil {
			inner = stmt
		}
		outer = stmt
	}
	var stmts []ast.Stmt
	if inner != nil {
		stmts = append(stmts, inner)
	}
	if outer != nil && outer != inner {
		stmts = append(stmts, outer)
	}
	return stmts
}

// clauseBody returns the statement list of a case clause or of a
// communication clause, and reports whether the statement is one.
func clauseBody(stmt ast.Stmt) ([]ast.Stmt, bool) {
	switch clause := stmt.(type) {
	case *ast.CaseClause:
		return clause.Body, true
	case *ast.CommClause:
		return clause.Body, true
	}
	return nil, false
}
