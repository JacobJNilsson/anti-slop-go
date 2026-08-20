// Package nointerfacereturn implements rule G09 of the anti-slop rule
// set. The rule rejects an interface result that every return of the
// body builds from one concrete type. The function knows the type it
// built, and the interface drops that evidence at the boundary the
// caller reads.
package nointerfacereturn

import (
	"fmt"
	"go/ast"
	"go/types"
	"slices"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/JacobJNilsson/anti-slop-go/internal/signature"
)

const doc = `reject an interface result that every return builds from one concrete type

A function that always builds one concrete type must return that type.
The interface hides the type the function already built, so the caller
loses the methods and the evidence that come with it. The rule is
opt-in: a project that plans a second implementation keeps the
interface.

The rule reads the body, and it reports a result only when the body
proves the conclusion. Every return statement must give the same
concrete type at that result. Two concrete types are honest interface
use, and the rule accepts them. A return of an expression whose type is
already the interface proves nothing, because the value comes from
somewhere else. A naked return reads a result variable of the interface
type, and it proves nothing either. A return of nil beside one concrete
type keeps the conclusion: the caller sees that type or nothing.

The rule reads a function declaration and a method, exported and
unexported, in a test file as well as a production file. A function
literal takes its signature from the call or the variable that holds
it, so the rule leaves it alone. The returns of a literal belong to the
literal, and never to the function around it.

The rule judges each result position on its own. It exempts the
predeclared error type, because Go returns errors through that
interface. It exempts the empty interface, which rule G04 reports, and
a result type that mentions a type parameter.

The rule accepts a method that an interface declares with the same
result, because the receiver must satisfy the whole interface. An
interface of an imported package fixes the signature, and an interface
of the package under analysis fixes it too: a method that narrows the
result no longer satisfies that interface, and the package stops
compiling. Advice that does not build is no advice.

Every other external contract needs a justification: a comment that
matches CONTRACT: directly above the declaration. The analyzer cannot
judge the text; review must.

Go groups names, so one result field can hold several results. Every
report of a field sits at the type it shares, and two results that
build the same type give one message.

The rule skips generated files.`

// Analyzer is the G09 analyzer. The rule is opt-in, so the
// golangci-lint plugin runs it only when the enable setting names it.
// Consumers get it through antislop.Analyzers().
var Analyzer = &analysis.Analyzer{
	Name:     "nointerfacereturn",
	Doc:      doc,
	URL:      "https://github.com/JacobJNilsson/anti-slop-go/blob/main/docs/spec/002-rules.md",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

const advice = "return the concrete type"

// errorType is the predeclared error interface. The universe holds one
// object for it, and every use of the name error names that object.
var errorType = types.Universe.Lookup("error").Type()

// CONTRACT: analysis.Analyzer.Run fixes this signature.
func run(pass *analysis.Pass) (any, error) {
	// SAFETY: inspect.Analyzer is in Requires, so the driver always
	// supplies its result, and that result is an *inspector.Inspector.
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// The scan reads the interfaces of the package under analysis too. A
	// method cannot narrow a result that a local interface declares,
	// because the package would stop compiling.
	contracts := signature.NewContractsWithHome(pass)

	insp.Preorder([]ast.Node{(*ast.FuncDecl)(nil)}, func(n ast.Node) {
		// SAFETY: the node filter above admits this type only.
		decl := n.(*ast.FuncDecl)
		if decl.Body == nil || decl.Type.Results == nil || contracts.Generated(decl.Pos()) {
			return
		}
		returns := bodyReturns(decl.Body)
		if len(returns) == 0 {
			return
		}

		count := resultCount(decl.Type.Results)
		index := 0
		for _, field := range decl.Type.Results.List {
			start, span := index, signature.NameCount(field)
			index += span
			declared := pass.TypesInfo.TypeOf(field.Type)
			if !judged(declared) {
				continue
			}
			// Go groups names, so one field can hold several results, and
			// every report of the field sits at the type it shares. Two
			// results that build the same type give one message, and the
			// reader needs it once.
			var reported []string
			for position := start; position < start+span; position++ {
				built, orNil, proven := singleConcrete(pass, returns, position, count)
				if !proven {
					continue
				}
				if contracts.Justified([]ast.Node{decl}) {
					// The comment covers every result of the signature.
					return
				}
				if contracts.Implements(decl, func(external *types.Signature) bool {
					return fixesResult(external, position, declared)
				}) {
					continue
				}
				message := fmt.Sprintf("result uses %s, and every return builds %s; %s",
					types.ExprString(field.Type), describe(pass, built, orNil), advice)
				if slices.Contains(reported, message) {
					continue
				}
				reported = append(reported, message)
				pass.Reportf(field.Type.Pos(), "%s", message)
			}
		}
	})

	return nil, nil
}

// judged reports whether the rule reads a result of this type.
//
// The predeclared error type is exempt: Go returns errors through that
// interface, and a concrete error result breaks the caller that
// compares the value with nil. The test is narrow, as in rule G10. An
// interface that embeds error is another type, and it carries no such
// promise.
//
// The empty interface is exempt too, because rule G04 reports it, and
// "return the concrete type" is the advice of that rule as well. A
// defined interface with no method, such as "type Empty interface{}",
// is a domain type and this rule judges it.
//
// A result type that mentions a type parameter is out of scope. Go
// reads and builds such a value through the interface.
func judged(t types.Type) bool {
	return types.IsInterface(t) &&
		!types.Identical(t, errorType) &&
		!signature.IsEmptyInterface(t) &&
		!signature.MentionsTypeParam(t)
}

// bodyReturns returns the return statements of one function body. A
// function literal inside the body carries its own results, so its
// returns belong to it and the walk stops there.
func bodyReturns(body *ast.BlockStmt) []*ast.ReturnStmt {
	var found []*ast.ReturnStmt
	ast.Inspect(body, func(n ast.Node) bool {
		switch stmt := n.(type) {
		case *ast.FuncLit:
			return false
		case *ast.ReturnStmt:
			found = append(found, stmt)
		}

		return true
	})

	return found
}

// resultCount returns the number of results a signature declares. Go
// groups names, so "(a, b Storage)" is one field and two results.
func resultCount(results *ast.FieldList) int {
	count := 0
	for _, field := range results.List {
		count += signature.NameCount(field)
	}

	return count
}

// singleConcrete reads the returns at one result position. It answers
// the type that every return builds there, whether a return gives nil
// beside that type, and whether the body proves the conclusion.
//
// One return that gives no concrete type ends the proof, because the
// author cannot narrow a result the body does not build.
func singleConcrete(pass *analysis.Pass, returns []*ast.ReturnStmt, position, count int) (types.Type, bool, bool) {
	var built types.Type
	orNil := false
	for _, ret := range returns {
		returned, readable := returnedType(pass, ret, position, count)
		if !readable {
			return nil, false, false
		}
		if isUntypedNil(returned) {
			orNil = true

			continue
		}
		if types.IsInterface(returned) {
			return nil, false, false
		}
		if built == nil {
			built = returned

			continue
		}
		if !types.Identical(built, returned) {
			return nil, false, false
		}
	}

	return built, orNil, built != nil
}

// returnedType returns the static type that one return statement gives
// at one result position, and whether the statement carries that
// evidence.
//
// A naked return carries none. It reads the result variable, whose type
// is the interface, and an assignment somewhere in the body set it. The
// rule reads no flow, so it stops there.
func returnedType(pass *analysis.Pass, ret *ast.ReturnStmt, position, count int) (types.Type, bool) {
	switch len(ret.Results) {
	case 0:
		return nil, false
	case count:
		return pass.TypesInfo.TypeOf(ret.Results[position]), true
	default:
		// SAFETY: a return statement carries one expression for several
		// results only when that expression is a call with several
		// results. The type of such a call is a tuple.
		tuple := pass.TypesInfo.TypeOf(ret.Results[0]).(*types.Tuple)

		return tuple.At(position).Type(), true
	}
}

// isUntypedNil reports whether an expression is the nil literal. Such a
// return names no type, so it holds the conclusion of the other
// returns: the caller sees one concrete type or nothing.
func isUntypedNil(t types.Type) bool {
	basic, isBasic := t.(*types.Basic)

	return isBasic && basic.Kind() == types.UntypedNil
}

// fixesResult reports whether an imported interface declares the same
// interface at this result position. The receiver must satisfy the
// whole interface, so the author cannot narrow that result.
func fixesResult(external *types.Signature, position int, declared types.Type) bool {
	if external == nil || position >= external.Results().Len() {
		return false
	}

	return types.Identical(external.Results().At(position).Type(), declared)
}

// describe names the concrete type of a report. The qualifier is
// relative to the package under analysis, so a local type reads as the
// author wrote it.
func describe(pass *analysis.Pass, built types.Type, orNil bool) string {
	name := types.TypeString(built, types.RelativeTo(pass.Pkg))
	if orNil {
		return name + " or nil"
	}

	return name
}
