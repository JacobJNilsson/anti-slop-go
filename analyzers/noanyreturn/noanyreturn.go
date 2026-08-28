// Package noanyreturn implements rule G04 of the anti-slop rule set.
// The rule rejects a function result of the empty interface type,
// because such a result makes every caller assert to get the value back.
package noanyreturn

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/JacobJNilsson/anti-slop-go/internal/signature"
)

const doc = `reject results of the empty interface type

A result of type any, or interface{}, hides the type the function
already built, so every caller must assert to get it back. Return the
concrete type, a small interface the caller consumes, or a generic
result.

The analyzer reports a result of a function declaration, a method, a
function literal, an interface method, and a declared function type. An
alias, such as "type A = any", is the same type, so the analyzer reports
its use sites. A defined type, such as "type Payload any", is a domain
type, so the analyzer accepts it. A type parameter is not the empty
interface, so "[T any]" and its uses stay clean.

The analyzer accepts a method that an interface of an imported package
declares with the same result, such as Value(key any) any of
context.Context, because the receiver must satisfy the whole interface.

Every other external contract needs a justification: a comment that
names the API which sets the signature, directly above the declaration.
sync.Pool.New is such an API. A doc comment, whose text starts with the
name of the declaration, justifies nothing, unless a line of it starts
with CONTRACT:. The analyzer cannot judge the text; review must.

The analyzer skips generated files.`

// Analyzer is the G04 analyzer. Consumers get it through
// antislop.Analyzers().
var Analyzer = &analysis.Analyzer{
	Name:     "noanyreturn",
	Doc:      doc,
	URL:      "https://github.com/JacobJNilsson/anti-slop-go/blob/main/docs/spec/002-rules.md",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

const advice = "return a concrete type" +
	", or name the API that sets the signature in a comment directly above the declaration"

// CONTRACT: analysis.Analyzer.Run fixes this signature.
func run(pass *analysis.Pass) (any, error) {
	// SAFETY: inspect.Analyzer is in Requires, so the driver always
	// supplies its result, and that result is an *inspector.Inspector.
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	contracts := signature.NewContracts(pass)

	// A FuncType covers a function declaration, a method, a function
	// literal, an interface method, and a "type H func(...)" declaration
	// in one visit, which keeps every signature to a single report.
	insp.WithStack([]ast.Node{(*ast.FuncType)(nil)}, func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push {
			return true
		}
		// SAFETY: the node filter above admits this type only.
		sig := n.(*ast.FuncType)
		if sig.Results == nil || contracts.Generated(sig.Pos()) {
			return true
		}
		// A FuncType always has a parent: a declaration, a literal, or a
		// field holds it.
		parent := stack[len(stack)-2]

		index := 0
		for _, field := range sig.Results.List {
			start, span := index, signature.NameCount(field)
			index += span
			if !signature.IsEmptyInterface(pass.TypesInfo.TypeOf(field.Type)) {
				continue
			}
			if contracts.Justified(stack) {
				// The comment covers every result of the signature.
				return true
			}
			if contracts.Implements(parent, func(declared *types.Signature) bool {
				return anyResults(declared, start, span)
			}) {
				continue
			}
			pass.Reportf(field.Type.Pos(), "result uses %s; %s", types.ExprString(field.Type), advice)
		}
		return true
	})
	return nil, nil
}

// anyResults reports whether declared makes the empty interface
// mandatory at every result from start to start+span.
func anyResults(declared *types.Signature, start, span int) bool {
	if declared == nil {
		return false
	}
	if start+span > declared.Results().Len() {
		return false
	}
	for i := start; i < start+span; i++ {
		if !signature.IsEmptyInterface(declared.Results().At(i).Type()) {
			return false
		}
	}
	return true
}
