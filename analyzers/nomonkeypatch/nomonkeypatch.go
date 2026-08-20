// Package nomonkeypatch implements rule G08 of the anti-slop rule set.
// The rule rejects a test that rewires production code through a
// mutable global, through a runtime patching library, or through the
// linker. The seam belongs in the design.
package nomonkeypatch

import (
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/JacobJNilsson/anti-slop-go/internal/signature"
)

const doc = `reject monkey patching in tests

A test must not rewire production code through a mutable global. The
seam belongs in the design: accept an interface or a function value as
a parameter or as a field, and give the test its own value.

The analyzer reads test files, which are the files whose name ends in
"_test.go". It reports three shapes there.

An assignment rewires behaviour that a package-level variable holds.
Behaviour is a function value or an interface. The variable belongs to
the package under test, or to an imported package, and both are
reports. A defined type, such as "type Hook func()", carries the
behaviour of the type it names.

The target reaches the variable through a field, an index, or a
pointer, so "Options.Now" and "Registry[name]" are reports as well.
The variable at the root of the target decides. A local variable, a
container that the test builds, and a variable that a test file
declares all belong to the test. An assignment to one of them stands.
A range clause that assigns with "=" is an assignment as well. The
analyzer reads the target of the assignment and never the value.

An import names a runtime patching library, which rewrites a function
at run time. The analyzer reports the import itself, because the import
is the decision.

A //go:linkname directive reaches a symbol that the test does not own.
The analyzer reports the directive in a test file. A directive in a
production file is systems programming, such as a runtime shim, and the
rule cannot judge it.

No comment justifies a report. The fix is an injected dependency, so
the author changes the design and the report goes away. The analyzer
skips generated files.`

// Analyzer is the G08 analyzer. Consumers get it through
// antislop.Analyzers().
var Analyzer = &analysis.Analyzer{
	Name:     "nomonkeypatch",
	Doc:      doc,
	URL:      "https://github.com/JacobJNilsson/anti-slop-go/blob/main/docs/spec/002-rules.md",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// patchLibraries holds the module paths of the runtime patching
// libraries the rule rejects. Each entry matches the path itself and
// every directory under it, so a version path such as
// "github.com/agiledragon/gomonkey/v2" matches as well.
//
// The list is a constant of the rule. A project that needs another
// entry files an issue, and the list grows here for every project.
var patchLibraries = []string{
	"bou.ke/monkey",
	"github.com/agiledragon/gomonkey",
}

// linknameDirective is the compiler directive that binds a name in
// this package to a symbol of another package.
const linknameDirective = "//go:linkname"

// The three messages the rule emits. Each one names the thing it
// rejects, and each one ends with the same fix.
const (
	variableMessage  = "test assigns to the package-level variable %s; %s"
	libraryMessage   = "test imports the runtime patching library %q; %s"
	directiveMessage = "test uses a " + linknameDirective + " directive; %s"
)

const advice = "inject the dependency through a parameter or a field"

// CONTRACT: analysis.Analyzer.Run fixes this signature.
func run(pass *analysis.Pass) (any, error) {
	// SAFETY: inspect.Analyzer is in Requires, so the driver always
	// supplies its result, and that result is an *inspector.Inspector.
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	generated := signature.GeneratedFiles(pass)

	// tests holds every file the rule reads: a test file that no
	// program writes. A report against a generated file has no reader
	// who can act on it.
	tests := make(map[*token.File]bool, len(pass.Files))
	for _, file := range pass.Files {
		tokenFile := pass.Fset.File(file.FileStart)
		if !signature.IsTestFile(tokenFile) || generated(file.FileStart) {
			continue
		}
		tests[tokenFile] = true
		reportImports(pass, file)
		reportDirectives(pass, file)
	}

	// report tests one assignment target. Parentheses around a target
	// are legal and change no meaning, so the message names the
	// expression under them.
	report := func(target ast.Expr) {
		target = ast.Unparen(target)
		if patches(pass, target) {
			pass.Reportf(target.Pos(), variableMessage, types.ExprString(target), advice)
		}
	}

	nodes := []ast.Node{(*ast.AssignStmt)(nil), (*ast.RangeStmt)(nil)}
	insp.Preorder(nodes, func(n ast.Node) {
		if !tests[pass.Fset.File(n.Pos())] {
			return
		}
		switch stmt := n.(type) {
		case *ast.AssignStmt:
			for _, target := range stmt.Lhs {
				report(target)
			}
		case *ast.RangeStmt:
			// A range clause with "=" assigns to variables that another
			// statement declares, so it patches like an assignment. With
			// ":=" it declares its own variables, and the token is then
			// DEFINE. A clause with no variable carries no token at all.
			if stmt.Tok != token.ASSIGN {
				return
			}
			report(stmt.Key)
			if stmt.Value != nil {
				report(stmt.Value)
			}
		}
	})
	return nil, nil
}

// patches reports whether an assignment target rewires behaviour that
// a package-level variable of production code holds.
//
// The analyzer reads the target and never the value. A method value, a
// function literal, and a named function all rewire the same seam.
func patches(pass *analysis.Pass, target ast.Expr) bool {
	root := rootVariable(pass.TypesInfo, target)
	if root == nil || !packageLevel(root) {
		return false
	}
	// A variable that a test file declares is test infrastructure, and
	// the test files of the package own it. The test that assigns to it
	// rewires no production code.
	//
	// The file of the declaration answers this question, and the file
	// of the assignment does not. A helper declares the variable in one
	// test file, and another test file of the package assigns to it.
	if declaredInTestFile(pass.Fset, root.Pos()) {
		return false
	}
	// The root carries the ownership, and the target carries the type.
	// A package-level container holds fields and entries of every type,
	// and only the ones that carry behaviour are a seam. The type
	// checker records every target that reaches a variable, so the type
	// here is never nil.
	return holdsBehaviour(pass.TypesInfo.TypeOf(target))
}

// rootVariable returns the variable that an assignment target reaches,
// and nil where the target reaches none. A field selector, an index
// expression, and a pointer indirection all lead to the variable that
// holds the value. So "Options.Now" and "Registry[key]" both reach a
// package-level container.
//
// A qualified name, such as "clock.Now", is the one selector that
// stops the walk. The name before the dot is a package, and a package
// is no variable.
func rootVariable(info *types.Info, target ast.Expr) *types.Var {
	for {
		switch expr := ast.Unparen(target).(type) {
		case *ast.Ident:
			// Uses holds the identifiers that read an object. A short
			// variable declaration puts its new names in Defs, so a
			// local never arrives here, whatever the token is.
			variable, _ := info.Uses[expr].(*types.Var)
			return variable
		case *ast.SelectorExpr:
			if isQualifier(info, expr.X) {
				variable, _ := info.Uses[expr.Sel].(*types.Var)
				return variable
			}
			target = expr.X
		case *ast.IndexExpr:
			target = expr.X
		case *ast.StarExpr:
			target = expr.X
		default:
			return nil
		}
	}
}

// isQualifier reports whether an expression names an imported package.
func isQualifier(info *types.Info, expr ast.Expr) bool {
	name, isIdent := ast.Unparen(expr).(*ast.Ident)
	if !isIdent {
		return false
	}
	_, isPackage := info.Uses[name].(*types.PkgName)
	return isPackage
}

// packageLevel reports whether a variable sits in the scope of its
// package. A local variable has the scope of a block, and a struct
// field has no scope at all, so both answer false.
func packageLevel(variable *types.Var) bool {
	return variable.Pkg() != nil && variable.Parent() == variable.Pkg().Scope()
}

// holdsBehaviour reports whether a type carries behaviour that a test
// can replace. A function value is one such type. An interface is the
// other. A variable of an interface type holds an implementation, and
// an assignment gives the package another one.
//
// The test takes the underlying type. A defined function type, such as
// "type Hook func()", and a named interface both rewire the same seam.
func holdsBehaviour(t types.Type) bool {
	switch types.Unalias(t).Underlying().(type) {
	case *types.Signature, *types.Interface:
		return true
	}
	return false
}

// declaredInTestFile reports whether a declaration sits in a test file.
// A position of another package can be absent from the file set, and
// the rule then reads the declaration as production code. That is the
// safe answer: no test file of the package under analysis can declare
// a variable of another package.
func declaredInTestFile(fset *token.FileSet, pos token.Pos) bool {
	file := fset.File(pos)
	return file != nil && signature.IsTestFile(file)
}

// reportImports reports every import of a runtime patching library.
// The import is the decision, so the position of the import serves the
// reader better than the position of a call.
func reportImports(pass *analysis.Pass, file *ast.File) {
	for _, spec := range file.Imports {
		// The parser accepts a quoted string only, so the path always
		// unquotes. A broken literal gives the empty path, which
		// matches no library.
		path, _ := strconv.Unquote(spec.Path.Value)
		if isPatchLibrary(path) {
			pass.Reportf(spec.Pos(), libraryMessage, path, advice)
		}
	}
}

// isPatchLibrary reports whether an import path names a runtime
// patching library. A path under the module path, such as a version
// directory, is the same library. A path that only starts with the
// same letters is another library.
func isPatchLibrary(path string) bool {
	for _, library := range patchLibraries {
		if path == library || strings.HasPrefix(path, library+"/") {
			return true
		}
	}
	return false
}

// reportDirectives reports every //go:linkname directive of a file.
func reportDirectives(pass *analysis.Pass, file *ast.File) {
	for _, group := range file.Comments {
		for _, comment := range group.List {
			if isLinkname(comment.Text) {
				pass.Reportf(comment.Pos(), directiveMessage, advice)
			}
		}
	}
}

// isLinkname reports whether the text of a comment is a //go:linkname
// directive. A directive carries no space between the slashes and the
// name, and a space or a tab separates the name from its arguments.
// Another name that starts with the same letters, such as
// "//go:linknamex", is another directive.
func isLinkname(text string) bool {
	rest, isDirective := strings.CutPrefix(text, linknameDirective)
	if !isDirective || rest == "" {
		return false // The compiler rejects a directive with no argument.
	}
	return rest[0] == ' ' || rest[0] == '\t'
}
