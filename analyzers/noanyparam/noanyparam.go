// Package noanyparam implements rule G03 of the anti-slop rule set.
// The rule rejects a parameter of the empty interface type, because such
// a parameter drops what the caller knows and makes the callee find it
// again.
package noanyparam

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/JacobJNilsson/anti-slop-go/internal/signature"
)

const doc = `reject parameters of the empty interface type

A parameter of type any, or interface{}, accepts every value, so the
callee must find the type that the caller already knew. Accept a named
domain type, or a type parameter with a constraint.

The analyzer reports a parameter of a function declaration, a method, a
function literal, an interface method, and a declared function type. An
alias, such as "type A = any", is the same type, so the analyzer reports
its use sites. A defined type, such as "type Payload any", is a domain
type, so the analyzer accepts it. A type parameter is not the empty
interface, so "[T any]" and its uses stay clean.

The analyzer accepts three shapes. A variadic "...any" tail belongs to
an fmt-style helper when an earlier parameter has the type string and
either that parameter's name starts or ends with "format" or the
signature has a name of more than one letter that ends in "f". A
parameter named "cause" is the upstream error-wrapping shape. A method
that an interface of an imported package declares with the same
parameter keeps that parameter, because the receiver must satisfy the
whole interface.

Every other external contract needs a justification: a comment that
matches CONTRACT: and names the API which sets the signature, directly
above the declaration. The analyzer cannot judge the text; review must.

The analyzer skips generated files.`

// Analyzer is the G03 analyzer. Consumers get it through
// antislop.Analyzers().
var Analyzer = &analysis.Analyzer{
	Name:     "noanyparam",
	Doc:      doc,
	URL:      "https://github.com/JacobJNilsson/anti-slop-go/blob/main/docs/spec/002-rules.md",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

const advice = "accept a named domain type"

// causeName is the parameter name of the upstream error-wrapping
// helper. The analyzer cannot read intent, so the name is the contract.
const causeName = "cause"

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
		if contracts.Generated(sig.Pos()) {
			return true
		}
		// A FuncType always has a parent: a declaration, a literal, or a
		// field holds it.
		parent := stack[len(stack)-2]

		index := 0
		for _, field := range sig.Params.List {
			start, span := index, signature.NameCount(field)
			index += span
			if !signature.IsEmptyInterface(pass.TypesInfo.TypeOf(elementType(field.Type))) {
				continue
			}
			if wrapsCause(field) || formatStyle(pass, sig, field, signatureName(parent)) {
				continue
			}
			if contracts.Justified(stack) {
				// The comment covers every parameter of the signature.
				return true
			}
			if contracts.Implements(parent, func(declared *types.Signature) bool {
				return anyParams(declared, start, span)
			}) {
				continue
			}
			pass.Reportf(field.Type.Pos(), "parameter uses %s; %s", types.ExprString(field.Type), advice)
		}
		return true
	})
	return nil, nil
}

// elementType returns the type the rule tests. A variadic parameter
// writes "...T" and accepts a T, so the rule tests T.
func elementType(e ast.Expr) ast.Expr {
	if ellipsis, isVariadic := e.(*ast.Ellipsis); isVariadic {
		return ellipsis.Elt
	}
	return e
}

// wrapsCause reports whether every name a field declares is "cause". A
// group such as "(cause, value any)" keeps the report, because only one
// of its names carries the error-wrapping contract.
func wrapsCause(field *ast.Field) bool {
	if len(field.Names) == 0 {
		return false
	}
	for _, name := range field.Names {
		if name.Name != causeName {
			return false
		}
	}
	return true
}

// formatStyle reports whether sig is an fmt-style helper and field is
// its variadic tail. Go allows the variadic form on the last parameter
// only, so an ellipsis identifies the tail.
func formatStyle(pass *analysis.Pass, sig *ast.FuncType, field *ast.Field, name string) bool {
	if _, isVariadic := field.Type.(*ast.Ellipsis); !isVariadic {
		return false
	}
	stringParam := false
	for _, earlier := range sig.Params.List {
		if earlier == field {
			break
		}
		if !types.Identical(pass.TypesInfo.TypeOf(earlier.Type), types.Typ[types.String]) {
			continue
		}
		stringParam = true
		for _, id := range earlier.Names {
			if namesFormat(id.Name) {
				return true
			}
		}
	}
	return stringParam && len(name) > 1 && strings.HasSuffix(name, "f")
}

// namesFormat reports whether a parameter name says "format". The test
// takes a prefix or a suffix, and ignores case, so it accepts "format",
// "formatString", and "msgFormat". A name that only holds the word
// inside itself, such as "informationText", is another word.
func namesFormat(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasPrefix(lower, "format") || strings.HasSuffix(lower, "format")
}

// signatureName returns the name the source gives a signature: the name
// of a function or method, of an interface method, of a function field,
// or of a declared function type. A function literal and an unnamed
// parameter have no name.
func signatureName(parent ast.Node) string {
	switch p := parent.(type) {
	case *ast.FuncDecl:
		return p.Name.Name
	case *ast.TypeSpec:
		return p.Name.Name
	case *ast.Field:
		if len(p.Names) > 0 {
			return p.Names[0].Name
		}
	}
	return ""
}

// anyParams reports whether declared makes the empty interface
// mandatory at every parameter from start to start+span.
func anyParams(declared *types.Signature, start, span int) bool {
	if declared == nil {
		return false
	}
	if start+span > declared.Params().Len() {
		return false
	}
	for i := start; i < start+span; i++ {
		if !signature.IsEmptyInterface(paramType(declared, i)) {
			return false
		}
	}
	return true
}

// paramType returns the type that parameter i of declared accepts. A
// variadic tail holds a slice, and a caller passes its element type.
func paramType(declared *types.Signature, i int) types.Type {
	t := declared.Params().At(i).Type()
	if slice, isSlice := t.(*types.Slice); isSlice && declared.Variadic() && i == declared.Params().Len()-1 {
		return slice.Elem()
	}
	return t
}
