// Package separategotwant implements rule G14 of the anti-slop rule
// set. The rule rejects an assertion whose argument calls a test
// helper that takes the testing value. Such a helper asserts, logs,
// and stops the test from inside an expression. The reader of the
// comparison then sees no value at all.
package separategotwant

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"

	"github.com/JacobJNilsson/anti-slop-go/internal/pathmatch"
	"github.com/JacobJNilsson/anti-slop-go/internal/signature"
)

const doc = `bind got and want before the assertion

An assertion argument that calls a test helper hides an act inside the
check. A helper that takes a testing.TB value can assert, log, and stop
the test from inside an expression. The failure then points into the
helper, and the reader of the comparison cannot see the value that
arrived. Bind got and want to variables first, then assert.

The rule reads the Diff function of go-cmp, and every function of the
testify require and assert packages. It resolves the import path of the
call, so another package with one of those names gives no assertion.

The rule reports a nested call of a value argument that receives a
testing.TB value: a *testing.T, a *testing.B, a *testing.F, or the
testing.TB interface. The testing value that a testify assertion takes
first counts for nothing, and the rule reads every argument of cmp.Diff.
A pure builder such as time.Date takes no testing value, and it stays
legal inline.

Each nested call gets one report, at the call. A call inside a function
literal runs after the assertion built its arguments, so the rule reads
no line of such a body. A type conversion names no callable, so it earns
no report of its own.

The rule reads test files only, and it skips generated files. A test
file is a file whose name ends in _test.go. Every file of a package that
the testpackages flag names is a test file as well. The golangci-lint
plugin takes the same patterns in the test-packages setting. A package
that serves tests and carries no _test.go name, such as a shared suite,
needs that entry.

No comment stops a report. The fix is always available and costs one
line. A justification could only state convenience, which is no
invariant. The escapes are the opt-in severity of the rule and the
disable setting.`

// The import paths the rule resolves. The rule reads the package of the
// object the type checker resolved, so a local package with one of
// these names reports nothing.
const (
	cmpPath     = "github.com/google/go-cmp/cmp"
	requirePath = "github.com/stretchr/testify/require"
	assertPath  = "github.com/stretchr/testify/assert"
	testingPath = "testing"
)

// diffName is the one function of go-cmp that the rule reads. cmp.Equal
// answers a comparison too, and rule G12 asks the author to write
// cmp.Diff, so the rule follows that shape.
const diffName = "Diff"

// message names the problem and the fix. The fix is one line, so the
// message states it as an instruction.
const message = "the assertion argument calls a helper that takes a testing value; " +
	"bind got and want first, then assert"

// The testing types that carry the control flow of a test. A pointer
// argument names one of the first three, and an interface argument
// names TB. A helper that takes one of them can fail the test.
var (
	testingPointers   = map[string]bool{"T": true, "B": true, "F": true}
	testingInterfaces = map[string]bool{"TB": true}
)

// Analyzer is the G14 analyzer. The rule is opt-in, so the
// golangci-lint plugin runs it only when the enable setting names it.
// Consumers get it through antislop.Analyzers(), and cmd/antislop
// registers its flags.
var Analyzer = New(nil)

// New returns an analyzer that reads the packages the testPackages
// patterns name as test code. A programmatic consumer, such as the
// golangci-lint plugin, builds one instance for each configuration. Two
// runs can hold different settings, and the package-level value is
// shared.
//
// The instance carries its own flag, which writes into the
// configuration of that instance. The flag is the configuration surface
// of cmd/antislop and of go vet -vettool, which read no settings file.
func New(testPackages []string) *analysis.Analyzer {
	cfg := &config{testPackages: testPackages}
	a := &analysis.Analyzer{
		Name:     "separategotwant",
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

// CONTRACT: analysis.Analyzer.Run fixes this signature.
func (c *config) run(pass *analysis.Pass) (any, error) {
	// SAFETY: inspect.Analyzer is in Requires, so the driver always
	// supplies its result, and that result is an *inspector.Inspector.
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	generated := signature.GeneratedFiles(pass)
	testFile := signature.TestFiles(pass, c.testPackages)
	// One nested call gets one report. An assertion can sit inside
	// another assertion, as require.Empty(t, cmp.Diff(want, load(t)))
	// does, and both readings then reach the same call.
	reported := make(map[token.Pos]bool)

	insp.Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(n ast.Node) {
		if !testFile(n.Pos()) || generated(n.Pos()) {
			return
		}
		// SAFETY: the node filter above admits this type only.
		call := n.(*ast.CallExpr)
		reportArguments(pass, call, reported)
	})

	return nil, nil
}

// reportArguments reports the nested calls of one assertion. A call
// that is no assertion carries no argument the rule reads.
func reportArguments(pass *analysis.Pass, call *ast.CallExpr, reported map[token.Pos]bool) {
	args, isAssertion := valueArgs(pass, call)
	if !isAssertion {
		return
	}
	for _, arg := range args {
		reportNested(pass, arg, reported)
	}
}

// valueArgs returns the arguments of an assertion that carry a value,
// and reports whether the call is an assertion at all.
//
// A testify function takes the testing value first, and that argument
// counts for nothing. The receiver form holds the testing value in the
// Assertions value, so every argument of such a call carries a value.
// cmp.Diff takes no testing value, so the rule reads every argument of
// it, the first one included.
func valueArgs(pass *analysis.Pass, call *ast.CallExpr) ([]ast.Expr, bool) {
	fn, _ := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
	if fn == nil || fn.Pkg() == nil {
		return nil, false
	}
	switch fn.Pkg().Path() {
	case cmpPath:
		if fn.Name() != diffName {
			return nil, false
		}

		return call.Args, true
	case requirePath, assertPath:
		if fn.Signature().Recv() != nil {
			return call.Args, true
		}

		return call.Args[min(1, len(call.Args)):], true
	default:
		return nil, false
	}
}

// reportNested reports every call of one argument that receives a
// testing value.
//
// The walk stops at a function literal. Such a body runs after the
// assertion built its arguments, so a call there is no part of the
// value that arrived. The fix of the rule is also unavailable there,
// because the closure holds the statement.
func reportNested(pass *analysis.Pass, arg ast.Expr, reported map[token.Pos]bool) {
	ast.Inspect(arg, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncLit:
			return false
		case *ast.CallExpr:
			// A type conversion names no callable, so it earns no
			// report. The walk enters its argument, because the
			// converted value can hold a call.
			if pass.TypesInfo.Types[node.Fun].IsType() {
				return true
			}
			if takesTestingValue(pass, node) && !reported[node.Pos()] {
				reported[node.Pos()] = true
				pass.Reportf(node.Pos(), "%s", message)
			}
		}

		return true
	})
}

// takesTestingValue reports whether one call receives a testing value.
// Such a call holds the control flow of the test, and it can end the
// test from inside the assertion.
func takesTestingValue(pass *analysis.Pass, call *ast.CallExpr) bool {
	for _, arg := range call.Args {
		if isTestingValue(pass.TypesInfo.TypeOf(arg)) {
			return true
		}
	}

	return false
}

// isTestingValue reports whether a type is a testing value: a pointer
// to testing.T, testing.B, or testing.F, or the testing.TB interface.
// The three pointer types and the interface are the values that stop a
// test.
func isTestingValue(t types.Type) bool {
	if pointer, isPointer := types.Unalias(t).(*types.Pointer); isPointer {
		return isTestingType(pointer.Elem(), testingPointers)
	}

	return isTestingType(t, testingInterfaces)
}

// isTestingType reports whether a type is one of the named types of the
// testing package. The test reads the package of the type object, so a
// local type named T is another type.
func isTestingType(t types.Type, names map[string]bool) bool {
	named, isNamed := types.Unalias(t).(*types.Named)
	if !isNamed {
		return false
	}
	obj := named.Obj()

	return obj.Pkg() != nil && obj.Pkg().Path() == testingPath && names[obj.Name()]
}
