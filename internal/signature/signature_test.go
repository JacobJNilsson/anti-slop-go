package signature_test

import (
	"go/ast"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/JacobJNilsson/anti-slop-go/internal/signature"
)

// probe is a test-only analyzer. It reports every parameter of the
// empty interface type that the shared tests do not accept, so one run
// drives the whole package the way rules G03 and G04 drive it. The
// package holds no knowledge of parameters against results, so the
// parameter side alone reaches every path.
var probe = &analysis.Analyzer{
	Name:     "probe",
	Doc:      "report empty interface parameters that no contract accepts",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      probeRun,
}

// CONTRACT: analysis.Analyzer.Run fixes this signature.
func probeRun(pass *analysis.Pass) (any, error) {
	// SAFETY: inspect.Analyzer is in Requires, so the driver always
	// supplies its result, and that result is an *inspector.Inspector.
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	contracts := signature.NewContracts(pass)

	insp.WithStack([]ast.Node{(*ast.FuncType)(nil)}, func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push {
			return true
		}
		// SAFETY: the node filter above admits this type only.
		sig := n.(*ast.FuncType)
		if contracts.Generated(sig.Pos()) {
			return true
		}
		parent := stack[len(stack)-2]

		index := 0
		for _, field := range sig.Params.List {
			start, span := index, signature.NameCount(field)
			index += span
			if !signature.IsEmptyInterface(pass.TypesInfo.TypeOf(field.Type)) {
				continue
			}
			if contracts.Justified(stack) {
				return true
			}
			if contracts.Implements(parent, func(declared *types.Signature) bool {
				return declaresEmpty(declared, start, span)
			}) {
				continue
			}
			pass.Reportf(field.Type.Pos(), "parameter uses the empty interface")
		}
		return true
	})
	return nil, nil
}

// declaresEmpty reports whether declared holds the empty interface at
// every parameter from start to start+span.
func declaresEmpty(declared *types.Signature, start, span int) bool {
	if declared == nil || start+span > declared.Params().Len() {
		return false
	}
	for i := start; i < start+span; i++ {
		if !signature.IsEmptyInterface(declared.Params().At(i).Type()) {
			return false
		}
	}
	return true
}

func TestContracts(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), probe, "a")
}
