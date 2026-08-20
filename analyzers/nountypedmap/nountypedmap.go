// Package nountypedmap implements rule G02 of the anti-slop rule set.
// The rule rejects a map with an empty interface value type, because
// such a map names no field and carries no evidence about its content.
package nountypedmap

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const doc = `reject maps with an empty interface value type

A map[string]any describes nothing: the keys are invisible and the
values carry no type. The analyzer reports such a map in a function
parameter, a function result, a struct field, or a package-level
variable. Describe the data with a named struct and decode into that
struct at the boundary.

An alias, such as "type Alias = map[string]any", is the same type, so
the analyzer reports its use sites. A defined type, such as
"type Headers map[string]any", is a domain type, so the analyzer
accepts it.

Local variables are out of scope. A parameter of a function literal
stays in scope, wherever the literal is. The analyzer tests the value
type of the map only, so it reports neither a map inside another type,
such as []map[string]any, nor a variadic ...map[string]any. Generic
instantiations, such as Box[any], and type parameter constraints are
out of scope: G03 and G04 cover the "any" type itself.`

// Analyzer is the G02 analyzer. Consumers get it through
// antislop.Analyzers().
var Analyzer = &analysis.Analyzer{
	Name:     "nountypedmap",
	Doc:      doc,
	URL:      "https://github.com/JacobJNilsson/anti-slop-go/blob/main/docs/spec/002-rules.md",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

const advice = "describe the data with a named struct"

func run(pass *analysis.Pass) (any, error) {
	// SAFETY: inspect.Analyzer is in Requires, so the driver always
	// supplies its result, and that result is an *inspector.Inspector.
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	generated := generatedFiles(pass)

	// report tests the type the compiler gave the written expression,
	// so an alias resolves to the map it names.
	report := func(e ast.Expr, where string) {
		if !isUntypedMap(pass.TypesInfo.TypeOf(e)) {
			return
		}
		if generated[pass.Fset.File(e.Pos())] {
			return
		}
		pass.Reportf(e.Pos(), "%s uses %s; %s", where, types.ExprString(e), advice)
	}

	checkFieldList := func(fl *ast.FieldList, where string) {
		if fl == nil {
			return
		}
		for _, f := range fl.List {
			report(f.Type, where)
		}
	}

	checkPackageVar := func(decl *ast.GenDecl) {
		for _, spec := range decl.Specs {
			// SAFETY: the parser only produces a ValueSpec under a var
			// declaration.
			vs := spec.(*ast.ValueSpec)
			if vs.Type != nil {
				report(vs.Type, "package-level variable")
				continue
			}
			// The type is inferred, so the name carries the position.
			for _, name := range vs.Names {
				obj := pass.TypesInfo.Defs[name]
				if obj == nil || !isUntypedMap(obj.Type()) {
					continue
				}
				if generated[pass.Fset.File(name.Pos())] {
					continue
				}
				// RelativeTo keeps the message in the reader's terms: a
				// type of the analysed package prints without its path.
				written := types.TypeString(obj.Type(), types.RelativeTo(pass.Pkg))
				pass.Reportf(name.Pos(), "package-level variable uses %s; %s", written, advice)
			}
		}
	}

	// A FuncType covers a function declaration, a function literal, an
	// interface method, and a "type H func(...)" declaration in one
	// visit, which keeps every signature to a single report. Method
	// receivers need no visit: Go accepts only a defined base type as a
	// receiver, and this rule never reports a defined type.
	nodes := []ast.Node{
		(*ast.FuncType)(nil),
		(*ast.StructType)(nil),
		(*ast.GenDecl)(nil),
	}
	insp.WithStack(nodes, func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push {
			return true
		}
		switch x := n.(type) {
		case *ast.FuncType:
			checkFieldList(x.Params, "parameter")
			checkFieldList(x.Results, "result")
		case *ast.StructType:
			checkFieldList(x.Fields, "struct field")
		case *ast.GenDecl:
			if x.Tok != token.VAR {
				return true
			}
			// A package-level declaration hangs directly off the file. A
			// local one sits in a DeclStmt, which the rule leaves alone.
			if _, atFile := stack[len(stack)-2].(*ast.File); !atFile {
				return true
			}
			checkPackageVar(x)
		}
		return true
	})
	return nil, nil
}

// isUntypedMap reports whether t is a map whose value type is the empty
// interface. It unaliases both types, so "any", "interface{}", and an
// alias of the map are the same type. It never takes the underlying
// type: a defined map type and a named empty interface are domain
// types, and the rule accepts them.
func isUntypedMap(t types.Type) bool {
	m, isMap := types.Unalias(t).(*types.Map)
	if !isMap {
		return false
	}
	elem, isIface := types.Unalias(m.Elem()).(*types.Interface)
	return isIface && elem.Empty()
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
