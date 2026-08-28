// Package justifypanic implements rule G11 of the anti-slop rule set.
// A call that stops the process in library code must carry a PANICS
// comment. The comment states why the process cannot continue.
package justifypanic

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"

	"github.com/JacobJNilsson/anti-slop-go/internal/pathmatch"
	"github.com/JacobJNilsson/anti-slop-go/internal/signature"
)

const doc = `require a PANICS comment for panic in library code

A panic outside main, outside init, and outside a test file is an API
decision. The author of a library stops the process of somebody else.
That author must state why the process cannot continue, in a comment
that sits directly above the call or above the statement that contains
it. The marker PANICS: must start a line of that comment.

The rule reads the object that the type checker resolved, and never a
name. It reports the builtin panic, os.Exit, and the Fatal, Fatalf,
Fatalln, Panic, Panicf, and Panicln of package log. The package
function and the method of log.Logger both stop the process, so the
rule reads them the same way. The message names the function of package
log in both forms, and the position names the call. A logger of the
project is another type, and the rule reads no call of it.

The rule leaves alone: the function main of a main package, an init
function of any package, every test file, a generated file, and a
rethrow of a recovered value.

A test file is a file whose name ends in _test.go. Every file of a
package that the testpackages flag names is a test file as well. The golangci-lint plugin
takes the same patterns in the test-packages setting. A package that
serves tests and carries no _test.go name, such as a shared suite,
needs that entry. A pattern matches the whole import path, and 003
states the syntax.`

// Analyzer reports a call that stops the process and carries no PANICS
// justification. It implements rule G11; see docs/spec/002-rules.md.
// The rule is opt-in, so the golangci-lint plugin runs it only when the
// enable setting names it. Consumers get it through
// antislop.Analyzers(), and cmd/antislop registers its flag.
var Analyzer = New(nil)

// New returns an analyzer that reads the packages the patterns name as
// test code. A programmatic consumer, such as the golangci-lint plugin,
// builds one instance for each configuration, because two runs can hold
// different patterns and the package-level value is shared.
//
// The instance carries its own flag, which writes into the
// configuration of that instance. The flag is the configuration surface
// of cmd/antislop and of go vet -vettool, which read no settings file.
func New(testPackages []string) *analysis.Analyzer {
	cfg := &config{testPackages: testPackages}
	a := &analysis.Analyzer{
		Name:     "justifypanic",
		Doc:      doc,
		URL:      "https://github.com/JacobJNilsson/anti-slop-go/blob/main/docs/spec/002-rules.md",
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Run:      cfg.run,
	}
	a.Flags.Var(&cfg.testPackages, "testpackages",
		"package path patterns whose files count as test files, separated by commas; a repeated flag adds patterns")

	return a
}

// config holds the settings of one analyzer instance.
type config struct {
	testPackages pathmatch.List
}

// message is the single diagnostic this rule emits. It names the call
// that stops the process, the missing justification, and the two fixes.
const message = "%s in library code has no PANICS justification; " +
	"state why the process cannot continue in a PANICS: comment directly above it, " +
	"or return an error to the caller"

// builtinPanic is the name of the builtin that raises a panic. The
// rethrow exemption reads it: log.Fatal and os.Exit raise nothing, so
// no call of them re-raises a recovered value.
const builtinPanic = "panic"

// terminators names, for each package, the functions that stop the
// process. Every Fatal of package log calls os.Exit, and every Panic of
// it calls the builtin. One name therefore holds the package function
// and the method of log.Logger.
//
// runtime.Goexit is no entry. It ends one goroutine and leaves the
// process running, so it is control flow and no termination.
var terminators = map[string][]string{
	"log": {"Fatal", "Fatalf", "Fatalln", "Panic", "Panicf", "Panicln"},
	"os":  {"Exit"},
}

// CONTRACT: analysis.Analyzer.Run fixes this signature.
func (c *config) run(pass *analysis.Pass) (any, error) {
	// SAFETY: inspect.Analyzer is in Requires, so the driver puts its
	// result type in ResultOf before it calls this function.
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	justifications := signature.NewMarkedJustifications(pass, "PANICS")
	testFile := signature.TestFiles(pass, c.testPackages)

	insp.WithStack([]ast.Node{(*ast.CallExpr)(nil)}, func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push {
			return true
		}
		// SAFETY: the node filter above admits this type only.
		call := n.(*ast.CallExpr)
		name, stops := terminator(pass.TypesInfo, call)
		if !stops {
			return true
		}
		if testFile(call.Pos()) || justifications.Generated(call.Pos()) {
			return true
		}
		if programEntry(pass, stack) {
			return true
		}
		if name == builtinPanic && rethrows(pass.TypesInfo, call, stack) {
			return true
		}
		if justifications.CommentAbove(call.Pos(), candidateLines(pass.Fset, call, stack)) {
			return true
		}
		// The report sits at the call and not at the statement. A
		// deferred call starts later than the statement that holds it,
		// and the call is the line the author justifies.
		pass.Reportf(call.Pos(), message, name)

		return true
	})

	return nil, nil
}

// terminator returns the name of the call that stops the process, and
// reports whether the call stops it. It reads the object that the type
// checker resolved, so a renamed import, a dot import, and a promoted
// method all give the same answer. A local value that carries the name
// of a package resolves to another object and stops nothing.
//
// The rule reads the call and follows no variable. A function value
// that holds os.Exit therefore gives no report at its call site.
func terminator(info *types.Info, call *ast.CallExpr) (string, bool) {
	switch callee := typeutil.Callee(info, call).(type) {
	case *types.Builtin:
		return builtinPanic, callee.Name() == builtinPanic
	case *types.Func:
		pkg := callee.Pkg()
		if pkg == nil {
			return "", false // A method of a builtin type, such as an error.
		}
		if !slices.Contains(terminators[pkg.Path()], callee.Name()) {
			return "", false
		}

		return pkg.Name() + "." + callee.Name(), true
	}

	return "", false
}

// programEntry reports whether the call sits in the function that
// starts the program, or in a function that runs before it. Both are
// the program itself, and nobody stands behind them.
//
// The exemption reads the function declaration that holds the call, so
// a function literal inside main is main. A declaration beside main in
// the same package is library code. The reader of that function still
// stands behind the call, and the name of the package says nothing.
func programEntry(pass *analysis.Pass, stack []ast.Node) bool {
	for _, node := range stack {
		decl, isFunc := node.(*ast.FuncDecl)
		if !isFunc {
			continue
		}
		// A method carries a receiver, and the runtime calls no method.
		// A method named init or main is an ordinary method.
		if decl.Recv != nil {
			return false
		}

		return decl.Name.Name == "init" || (decl.Name.Name == "main" && pass.Pkg.Name() == "main")
	}

	// A call outside every declaration sits in a function literal at
	// package level, which is library code.
	return false
}

// rethrows reports whether a panic call re-raises a value that a
// recover call of the same function produced. Such a call preserves the
// panic of another function and decides nothing, so it justifies
// nothing either.
//
// The shape is exact: the argument is a recover call, or a variable
// that a recover call of the same function filled. A new value, such as
// a wrapped message, is a new decision and keeps its report.
func rethrows(info *types.Info, call *ast.CallExpr, stack []ast.Node) bool {
	// Go gives panic exactly one argument, and every driver of an
	// analyzer reads a package that type-checks, so the index holds.
	arg := ast.Unparen(call.Args[0])
	if isRecover(info, arg) {
		return true
	}
	name, isIdent := arg.(*ast.Ident)
	if !isIdent {
		return false // A field or an entry names no variable to follow.
	}
	object := info.Uses[name]

	return object != nil && recovered(info, stack)[object]
}

// recovered returns the objects that a recover call of the function
// around the node filled. The function is the innermost declaration or
// literal of the stack, and a node outside every function gives an
// empty result.
func recovered(info *types.Info, stack []ast.Node) map[types.Object]bool {
	objects := make(map[types.Object]bool)
	for i := len(stack) - 1; i >= 0; i-- {
		body, isFunc := functionBody(stack[i])
		if !isFunc {
			continue
		}
		collectRecovered(info, body, objects)

		break
	}

	return objects
}

// functionBody returns the body of a function declaration or of a
// function literal, and reports whether the node is one. The body of
// the enclosing function of a call always exists: the call sits in it.
func functionBody(node ast.Node) (*ast.BlockStmt, bool) {
	switch fn := node.(type) {
	case *ast.FuncLit:
		return fn.Body, true
	case *ast.FuncDecl:
		return fn.Body, true
	}

	return nil, false
}

// collectRecovered records every object that a recover call of one
// function body filled. The walk stops at a nested function literal: a
// recover there belongs to that literal, which runs at another time.
func collectRecovered(info *types.Info, body *ast.BlockStmt, objects map[types.Object]bool) {
	ast.Inspect(body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncLit:
			return false
		case *ast.AssignStmt:
			if len(node.Rhs) == 1 && isRecover(info, node.Rhs[0]) {
				markObjects(info, node.Lhs, objects)
			}
		case *ast.ValueSpec:
			if len(node.Values) == 1 && isRecover(info, node.Values[0]) {
				markObjects(info, identExprs(node.Names), objects)
			}
		}

		return true
	})
}

// markObjects records the object of every target that names one. A
// short declaration puts its name in Defs, and an assignment to an
// existing variable puts it in Uses. The blank identifier names no
// object, and it drops out here.
func markObjects(info *types.Info, targets []ast.Expr, objects map[types.Object]bool) {
	for _, target := range targets {
		name, isIdent := ast.Unparen(target).(*ast.Ident)
		if !isIdent {
			continue
		}
		if object := info.Defs[name]; object != nil {
			objects[object] = true

			continue
		}
		if object := info.Uses[name]; object != nil {
			objects[object] = true
		}
	}
}

// identExprs widens the names of a value specification to expressions,
// so one target walk serves both declaration forms.
func identExprs(names []*ast.Ident) []ast.Expr {
	exprs := make([]ast.Expr, 0, len(names))
	for _, name := range names {
		exprs = append(exprs, name)
	}

	return exprs
}

// isRecover reports whether an expression is a call of the builtin
// recover.
func isRecover(info *types.Info, expr ast.Expr) bool {
	call, isCall := ast.Unparen(expr).(*ast.CallExpr)
	if !isCall {
		return false
	}
	builtin, isBuiltin := typeutil.Callee(info, call).(*types.Builtin)

	return isBuiltin && builtin.Name() == "recover"
}

// candidateLines returns the lines a PANICS comment may end directly
// above: the line of the call, and the lines of the statements that
// contain it.
//
// The statements come from the shared walk of the justification
// contract, which rule G01 reads as well. This rule adds no candidate
// of its own. A call starts where it starts, and no operand of it
// pushes the report down a line.
func candidateLines(fset *token.FileSet, call *ast.CallExpr, stack []ast.Node) []int {
	lines := []int{signature.LineOf(fset, call.Pos())}

	return append(lines, signature.EnclosingStmtLines(fset, stack)...)
}
