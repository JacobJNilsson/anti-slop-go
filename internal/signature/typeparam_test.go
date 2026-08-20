package signature_test

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/JacobJNilsson/anti-slop-go/internal/signature"
)

// paramSrc declares one parameter for each type constructor that the
// walk reads. Rules G05 and G06 both ask whether a type parameter sits
// inside a type, so the test lives here with the implementation.
const paramSrc = `package p

type Box[T any] struct{ v T }

func Params[T comparable](
	bare T,
	named Box[T],
	instantiated Box[int],
	keyed map[T]int,
	valued map[int]T,
	fields struct{ f T },
	fn func(T) int,
	result func(int) T,
	pointer *T,
	slice []T,
	array [2]T,
	channel chan T,
	plain int,
	interfaced any,
) {
}
`

// TestMentionsTypeParam pins the walk against every constructor an
// assertion or a conversion can name.
func TestMentionsTypeParam(t *testing.T) {
	params := paramsOf(t, paramSrc, "Params")

	tests := []struct {
		name string
		want bool
	}{
		{"bare", true},
		{"named", true},
		{"instantiated", false},
		{"keyed", true},
		{"valued", true},
		{"fields", true},
		{"fn", true},
		{"result", true},
		{"pointer", true},
		{"slice", true},
		{"array", true},
		{"channel", true},
		{"plain", false},
		{"interfaced", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := signature.MentionsTypeParam(paramType(t, params, tt.name)); got != tt.want {
				t.Errorf("MentionsTypeParam(%s) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// A tuple carries the parameters of a signature. The walk reads one
// when it reads a signature, and this test names the tuple itself.
func TestMentionsTypeParamReadsATuple(t *testing.T) {
	params := paramsOf(t, paramSrc, "Params")
	if !signature.MentionsTypeParam(params) {
		t.Error("MentionsTypeParam rejected a tuple that holds a type parameter")
	}

	empty := types.NewTuple()
	if signature.MentionsTypeParam(empty) {
		t.Error("MentionsTypeParam accepted an empty tuple")
	}
}

// paramsOf type-checks src and returns the parameters of one function.
func paramsOf(t *testing.T, src, name string) *types.Tuple {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatalf("the fixture source does not parse: %v", err)
	}
	conf := types.Config{Importer: importer.Default()}
	pkg, err := conf.Check("p", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatalf("the fixture source does not type-check: %v", err)
	}
	fn, isFunc := pkg.Scope().Lookup(name).(*types.Func)
	if !isFunc {
		t.Fatalf("the fixture declares no function %q", name)
	}
	declared, isSignature := fn.Type().(*types.Signature)
	if !isSignature {
		t.Fatalf("the object %q carries no signature", name)
	}

	return declared.Params()
}

// paramType returns the type of one named parameter.
func paramType(t *testing.T, params *types.Tuple, name string) types.Type {
	t.Helper()

	for i := range params.Len() {
		if params.At(i).Name() == name {
			return params.At(i).Type()
		}
	}
	t.Fatalf("the fixture declares no parameter %q", name)

	return nil
}
