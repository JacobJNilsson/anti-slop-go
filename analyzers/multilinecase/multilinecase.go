// Package multilinecase implements rule G15 of the anti-slop rule set.
// The rule rejects a test case that puts several keyed fields on one
// line. A case that holds each field on its own line reads from top to
// bottom, and a change to one value marks one line in a diff.
package multilinecase

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"

	"github.com/JacobJNilsson/anti-slop-go/internal/pathmatch"
	"github.com/JacobJNilsson/anti-slop-go/internal/signature"
)

const doc = `put each field of a test case on its own line

A table-driven test lists its cases in one literal. A case that holds
several fields on one line hides the want value in a row of text. The
reader counts commas to find a field, and a change to one value marks
the whole case in a diff.

A table is a composite literal of a slice, an array, or a map that a
range statement reads. The rule finds the literal in the range
expression, or in the initializer of the variable that the range
expression names. The variable comes from a := statement or a var
declaration. A later assignment is not read.

A case is an element of a slice or array table, or a value of a map
table, that is a composite literal of a struct type. The rule reports a
case with at least min keyed fields when its braces sit on one line.
The report carries a fix that puts each field on its own line. A case
with a comment between its braces gets no fix. gofmt aligns the values
after the fix.

The rule reads test files only, and it skips generated files. A test
file is a file whose name ends in _test.go. Every file of a package that
the testpackages flag names is a test file as well. The golangci-lint
plugin takes the same patterns in the test-packages setting.`

// DefaultMin is the number of keyed fields a report needs. A case of two
// fields, such as {name: "empty", want: 0}, reads well on one line.
const DefaultMin = 3

// Analyzer is the G15 analyzer. The rule is opt-in, so the
// golangci-lint plugin runs it only when the enable setting names it.
// Consumers get it through antislop.Analyzers(), and cmd/antislop
// registers its flags.
var Analyzer = New(DefaultMin, nil)

// New returns an analyzer that reports a case at min keyed fields and
// reads the packages the testPackages patterns name as test code. A
// programmatic consumer, such as the golangci-lint plugin, builds one
// instance for each configuration, because two runs can hold different
// settings and the package-level value is shared.
//
// The instance carries its own flags, which write into the
// configuration of that instance. The flags are the configuration
// surface of cmd/antislop and of go vet -vettool, which read no
// settings file.
func New(min int, testPackages []string) *analysis.Analyzer {
	cfg := &config{min: min, testPackages: testPackages}
	a := &analysis.Analyzer{
		Name: "multilinecase",
		Doc:  doc,
		URL:  "https://github.com/JacobJNilsson/anti-slop-go/blob/main/docs/spec/002-rules.md",
		Run:  cfg.run,
	}
	a.Flags.IntVar(&cfg.min, "min", min,
		"the number of keyed fields of one test case that a report needs")
	a.Flags.Var(&cfg.testPackages, "testpackages",
		"package path patterns whose files count as test files, separated by commas; a repeated flag adds patterns")

	return a
}

// config holds the settings of one analyzer instance.
type config struct {
	min          int
	testPackages pathmatch.List
}

// CONTRACT: analysis.Analyzer.Run fixes this signature.
func (c *config) run(pass *analysis.Pass) (any, error) {
	generated := signature.GeneratedFiles(pass)
	testFile := signature.TestFiles(pass, c.testPackages)

	var files []*ast.File
	for _, file := range pass.Files {
		if testFile(file.FileStart) && !generated(file.FileStart) {
			files = append(files, file)
		}
	}

	r := &reader{
		pass:     pass,
		min:      c.min,
		inits:    initializers(pass, files),
		reported: make(map[token.Pos]bool),
	}
	for _, file := range files {
		ast.Inspect(file, func(n ast.Node) bool {
			if stmt, isRange := n.(*ast.RangeStmt); isRange {
				r.readTable(file, stmt.X)
			}

			return true
		})
	}

	return nil, nil
}

// initializers maps each variable that a := statement or a var
// declaration of the files declares to its initializer. A name pairs
// with a value only where both lists have the same length, so a call
// that returns several values pairs with nothing.
func initializers(pass *analysis.Pass, files []*ast.File) map[*types.Var]ast.Expr {
	inits := make(map[*types.Var]ast.Expr)
	bind := func(names []*ast.Ident, values []ast.Expr) {
		if len(names) != len(values) {
			return
		}
		for i, name := range names {
			// Defs holds no entry for a name that := only assigns.
			if v, isVar := pass.TypesInfo.Defs[name].(*types.Var); isVar {
				inits[v] = values[i]
			}
		}
	}
	for _, file := range files {
		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.AssignStmt:
				if node.Tok == token.DEFINE {
					bind(identNames(node.Lhs), node.Rhs)
				}
			case *ast.ValueSpec:
				bind(node.Names, node.Values)
			}

			return true
		})
	}

	return inits
}

// identNames returns the names of a left-hand side. A := statement
// holds identifiers only, and the blank identifier has no variable.
func identNames(lhs []ast.Expr) []*ast.Ident {
	names := make([]*ast.Ident, 0, len(lhs))
	for _, expr := range lhs {
		// SAFETY: the parser accepts identifiers only on the left of :=.
		names = append(names, expr.(*ast.Ident))
	}

	return names
}

// reader holds the state of one run over one package.
type reader struct {
	pass  *analysis.Pass
	min   int
	inits map[*types.Var]ast.Expr
	// reported keeps one report for each case. Two loops can read one
	// table, and both readings reach the same case.
	reported map[token.Pos]bool
}

// readTable reads the cases of the table that one range expression
// names. An expression that names no table reads nothing. The compiler
// accepts a range over a composite literal of a slice, an array, or a
// map only, so the literal needs no type check.
func (r *reader) readTable(file *ast.File, expr ast.Expr) {
	expr = ast.Unparen(expr)
	if id, isIdent := expr.(*ast.Ident); isIdent {
		v, _ := r.pass.TypesInfo.Uses[id].(*types.Var)
		expr = r.inits[v]
	}
	table, isLit := ast.Unparen(expr).(*ast.CompositeLit)
	if !isLit {
		return
	}
	for _, elt := range table.Elts {
		// A map table and an indexed array table key each case.
		if kv, isKV := elt.(*ast.KeyValueExpr); isKV {
			elt = kv.Value
		}
		r.readCase(file, elt)
	}
}

// readCase reports one case that puts its keyed fields on one line.
func (r *reader) readCase(file *ast.File, expr ast.Expr) {
	expr = ast.Unparen(expr)
	if unary, isUnary := expr.(*ast.UnaryExpr); isUnary && unary.Op == token.AND {
		expr = ast.Unparen(unary.X)
	}
	lit, isLit := expr.(*ast.CompositeLit)
	if !isLit || !r.isStruct(lit) || r.reported[lit.Lbrace] {
		return
	}
	if len(lit.Elts) < max(r.min, 1) || !allKeyed(lit.Elts) {
		return
	}
	fset := r.pass.Fset
	if fset.Position(lit.Lbrace).Line != fset.Position(lit.Rbrace).Line {
		return
	}

	r.reported[lit.Lbrace] = true
	diag := analysis.Diagnostic{
		Pos:     lit.Lbrace,
		Message: message(len(lit.Elts)),
	}
	if fix, ok := r.splitFix(file, lit); ok {
		diag.SuggestedFixes = []analysis.SuggestedFix{fix}
	}
	r.pass.Report(diag)
}

// message names the problem and the fix.
func message(fields int) string {
	return fmt.Sprintf("test case puts %d keyed fields on one line; put each field on its own line", fields)
}

// isStruct reports whether the literal builds a struct. An elided
// literal in a table of pointers has the pointer type, so the check
// reads through one pointer.
func (r *reader) isStruct(lit *ast.CompositeLit) bool {
	t := r.pass.TypesInfo.TypeOf(lit)
	if pointer, isPointer := t.Underlying().(*types.Pointer); isPointer {
		t = pointer.Elem()
	}
	_, isStruct := t.Underlying().(*types.Struct)

	return isStruct
}

// allKeyed reports whether every element of a literal is a key and a
// value.
func allKeyed(elts []ast.Expr) bool {
	for _, elt := range elts {
		if _, isKV := elt.(*ast.KeyValueExpr); !isKV {
			return false
		}
	}

	return true
}

// splitFix builds the fix that puts each field of a case on its own
// line. The edits touch the gaps between the tokens only, so gofmt sets
// the indentation and aligns the values. A gap after a field also
// removes a trailing comma that was there. The fix is unavailable
// where a comment sits between the braces, because the edit could move
// such a comment to the wrong field.
func (r *reader) splitFix(file *ast.File, lit *ast.CompositeLit) (analysis.SuggestedFix, bool) {
	for _, group := range file.Comments {
		if group.Pos() > lit.Lbrace && group.Pos() < lit.Rbrace {
			return analysis.SuggestedFix{}, false
		}
	}

	edits := []analysis.TextEdit{{
		Pos:     lit.Lbrace + 1,
		End:     lit.Elts[0].Pos(),
		NewText: []byte("\n"),
	}}
	for i, elt := range lit.Elts {
		end := lit.Rbrace
		if i+1 < len(lit.Elts) {
			end = lit.Elts[i+1].Pos()
		}
		edits = append(edits, analysis.TextEdit{
			Pos:     elt.End(),
			End:     end,
			NewText: []byte(",\n"),
		})
	}

	return analysis.SuggestedFix{
		Message:   "put each field on its own line",
		TextEdits: edits,
	}, true
}
