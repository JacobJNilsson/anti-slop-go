// Package noadhoctypeswitch implements rule G06 of the anti-slop rule
// set. The rule rejects a type switch on a value of the empty interface
// type. Such a switch reads the type that an earlier line threw away,
// and it re-parses the data away from the boundary that received it.
package noadhoctypeswitch

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/JacobJNilsson/anti-slop-go/internal/pathmatch"
	"github.com/JacobJNilsson/anti-slop-go/internal/signature"
)

const doc = `reject a type switch on a value of the empty interface type

A type switch on an any value reads a dynamic type that an earlier line
threw away. The switch re-parses the data away from the boundary that
received it, and every new case adds a shape the reader must hold.
Branch on a domain value instead: a kind field, a sealed interface with
a marker method, or one handler for each type.

The rule reads the static type of the operand of the guard. It reports
"switch x.(type)" and "switch v := x.(type)" where that type is the
empty interface, and it reports at the .( token of the guard, so one
switch gets one diagnostic. A defined type, such as "type Payload any",
is a domain type of the package, so the rule accepts it. An error value
carries another type, and rule G10 owns it. A single type assertion is
another shape, which rules G01 and G05 own.

The rule accepts "switch any(v).(type)" where the type of v holds a
type parameter. Go reads a value of a parameterized type through an
interface only, so the widening is no choice of the author. Rule G05
leaves the same widening alone, and both rules read one walk.

The rule holds two escapes. The first is the boundary package. Such a
package decodes input, so every type switch of it is legal. The
boundary flag takes package path patterns, and the golangci-lint plugin
takes the same patterns in the boundary-packages setting. A pattern
matches the whole import path. "*" matches inside one segment, and
"..." matches any run of characters. A pattern that ends in "/..."
names the package above it too. The rule drops a trailing "_test" from
the path of a package, so one entry covers a package and its external
test package.

The second escape is the evidence that rule G03 accepts for an any
parameter: an interface of an imported package declares the parameter,
or a justification comment above the declaration names the API that
sets the signature. Such a parameter is legal under G03, and the switch is the
consumption of that contract. The signature that declares the parameter
answers the question, so a function literal answers for its own
parameter.

No comment above the switch stops a report. The justification belongs
to the signature that admitted the any value. The rule skips generated
files.`

// Analyzer is the G06 analyzer. Consumers get it through
// antislop.Analyzers(), and cmd/antislop registers its boundary flag.
var Analyzer = New(nil)

// New returns an analyzer that accepts the type switches of the
// packages the patterns name. A programmatic consumer, such as the
// golangci-lint plugin, builds one instance for each configuration,
// because two runs can hold different patterns and the package-level
// value is shared.
//
// The instance carries its own boundary flag, which writes into the
// configuration of that instance. The flag is the configuration surface
// of cmd/antislop and of go vet -vettool, which read no settings file.
func New(boundary []string) *analysis.Analyzer {
	cfg := &config{boundary: boundary}
	a := &analysis.Analyzer{
		Name:     "noadhoctypeswitch",
		Doc:      doc,
		URL:      "https://github.com/JacobJNilsson/anti-slop-go/blob/main/docs/spec/002-rules.md",
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Run:      cfg.run,
	}
	a.Flags.Var(&cfg.boundary, "boundary",
		"package path patterns that decode at a boundary, separated by commas; a repeated flag adds patterns")
	return a
}

// config holds the settings of one analyzer instance.
type config struct {
	boundary pathmatch.List
}

// message is the one diagnostic the rule emits. It names the problem
// and one direction for the fix. 002 holds the shapes of the fix: a
// kind field, a sealed interface with a marker method, or one handler
// for each type. The URL of the analyzer points there.
//
// The message names both spellings of the boundary setting, because
// the reader of a diagnostic runs one of two tools.
const message = "this type switch reads the dynamic type of an any value; " +
	"branch on a domain value, or name a boundary package with " +
	"boundary-packages (-noadhoctypeswitch.boundary)"

// CONTRACT: analysis.Analyzer.Run fixes this signature.
func (c *config) run(pass *analysis.Pass) (any, error) {
	if pathmatch.Any(c.boundary, packagePath(pass)) {
		return nil, nil
	}

	// SAFETY: inspect.Analyzer is in Requires, so the driver always
	// supplies its result, and that result is an *inspector.Inspector.
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	contracts := signature.NewContracts(pass)

	insp.WithStack([]ast.Node{(*ast.TypeAssertExpr)(nil)}, func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push {
			return true
		}
		// SAFETY: the node filter above admits this type only.
		assert := n.(*ast.TypeAssertExpr)
		// The guard of a type switch is the one assertion that names no
		// target type, and a switch holds exactly one guard. Every other
		// assertion belongs to rules G01 and G05.
		if assert.Type != nil {
			return true
		}
		operand := ast.Unparen(assert.X)
		if !signature.IsEmptyInterface(pass.TypesInfo.TypeOf(operand)) {
			return true
		}
		if contracts.Generated(assert.Pos()) {
			return true
		}
		if widensTypeParam(pass.TypesInfo, operand) {
			return true
		}
		if admitted(pass, contracts, stack, operand) {
			return true
		}
		// Report at the .( token and not at the start of the operand,
		// which can sit on an earlier line.
		pass.Reportf(assert.Lparen, "%s", message)

		return true
	})

	return nil, nil
}

// packagePath returns the import path the rule matches against the
// boundary patterns.
//
// The external test package of a package carries the path of that
// package with "_test" at the end. Both hold the code of one package,
// and a project that decodes at the boundary tests that code with the
// same switches, so the rule drops the suffix.
func packagePath(pass *analysis.Pass) string {
	return strings.TrimSuffix(pass.Pkg.Path(), "_test")
}

// widensTypeParam reports whether the operand is a conversion of a
// value whose type holds a type parameter.
//
// Go reads a value of a parameterized type through an interface only,
// so "switch any(v).(type)" is the one way to branch on the
// instantiation. The widening is no choice of the author, and no domain
// value replaces an instantiation, so the advice of the message does
// not fit that shape. Rule G05 leaves the same widening alone in an
// assertion. Both rules read the walk of internal/signature, so one
// expression gets one answer.
//
// The test reads a conversion and no call. A function that returns the
// empty interface hides the type behind a signature, and the author
// wrote that signature.
func widensTypeParam(info *types.Info, operand ast.Expr) bool {
	conversion, isCall := operand.(*ast.CallExpr)
	if !isCall || len(conversion.Args) != 1 {
		return false
	}
	// A call whose function names a type is a conversion. The type here
	// is the empty interface, because the caller tested the operand.
	if kind, recorded := info.Types[conversion.Fun]; !recorded || !kind.IsType() {
		return false
	}

	return signature.MentionsTypeParam(info.TypeOf(conversion.Args[0]))
}

// admitted reports whether a signature admitted the operand under the
// evidence that rule G03 accepts for an any parameter. Such a parameter
// is legal there, and this switch is the consumption of that contract.
// Where the parameter is not legal, G03 reports the signature, which is
// the line the author fixes.
//
// The signature that declares the parameter answers the question. A
// function literal therefore answers for its own parameter, and it
// keeps the answer of the function around it for a parameter that it
// reads from there.
func admitted(pass *analysis.Pass, contracts *signature.Contracts, stack []ast.Node, operand ast.Expr) bool {
	ident, isName := operand.(*ast.Ident)
	if !isName {
		return false
	}
	object := pass.TypesInfo.ObjectOf(ident)

	for i := len(stack) - 1; i >= 0; i-- {
		node, declared := signatureOf(pass, stack[i])
		if node == nil {
			continue
		}
		index, declares := parameterIndex(declared, object)
		if !declares {
			continue
		}

		return evidence(contracts, signatureStack(stack, i, node), index)
	}

	return false
}

// signatureOf returns the syntax and the type of the signature a node
// declares. A function declaration and a function literal declare one;
// every other node declares none.
func signatureOf(pass *analysis.Pass, node ast.Node) (*ast.FuncType, *types.Signature) {
	switch fn := node.(type) {
	case *ast.FuncDecl:
		// SAFETY: the type checker defines an object for the name of every
		// function declaration, and the type of a *types.Func is always a
		// *types.Signature. The driver skips a package that does not
		// type-check.
		return fn.Type, pass.TypesInfo.Defs[fn.Name].(*types.Func).Type().(*types.Signature)
	case *ast.FuncLit:
		// SAFETY: the type of a function literal is always a signature.
		return fn.Type, pass.TypesInfo.TypeOf(fn).(*types.Signature)
	}

	return nil, nil
}

// parameterIndex returns the position of object in the parameters of
// declared, and whether declared holds it there. The type checker builds
// one variable for each parameter, and the operand of the switch
// resolves to that variable, so the test is an identity of objects.
func parameterIndex(declared *types.Signature, object types.Object) (int, bool) {
	params := declared.Params()
	for i := range params.Len() {
		if params.At(i) == object {
			return i, true
		}
	}

	return 0, false
}

// signatureStack returns the stack that the shared tests of
// internal/signature read: the nodes above the function at index i, the
// function itself, and its signature at the end. Rule G03 asks the same
// questions with the same stack, so the two rules answer alike.
func signatureStack(stack []ast.Node, i int, sig *ast.FuncType) []ast.Node {
	out := make([]ast.Node, 0, i+2)
	out = append(out, stack[:i+1]...)

	return append(out, sig)
}

// evidence reports whether the signature at the end of stack carries the
// evidence of an external contract for the parameter at index.
func evidence(contracts *signature.Contracts, stack []ast.Node, index int) bool {
	if contracts.Justified(stack) {
		return true
	}
	// The function that holds the signature is the node before it. A
	// method reaches the interfaces of the imported packages here; a
	// function literal reaches none.
	parent := stack[len(stack)-2]

	return contracts.Implements(parent, func(declared *types.Signature) bool {
		return declaresEmptyParam(declared, index)
	})
}

// declaresEmptyParam reports whether an external signature makes the
// empty interface mandatory at one position. A nil signature means the
// interface declares no method of that name.
//
// The test of the type is a filter of cost and no gate. A receiver that
// implements the interface carries the signature that the interface
// declares, so the two agree at every position. The test rejects a
// method that only shares a name, before the test of the whole
// interface runs. The test of the bounds is a gate: a method with more
// parameters than the interface declares reaches this line, and the
// tuple of the interface holds no entry there.
//
// The test reads no variadic tail. A variadic parameter holds a slice
// inside the body of the function, so an operand of the empty interface
// type never names one.
func declaresEmptyParam(declared *types.Signature, index int) bool {
	if declared == nil || index >= declared.Params().Len() {
		return false
	}

	return signature.IsEmptyInterface(declared.Params().At(index).Type())
}
