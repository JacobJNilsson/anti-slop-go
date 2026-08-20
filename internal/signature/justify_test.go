package signature_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"testing"

	"golang.org/x/tools/go/ast/inspector"

	"github.com/JacobJNilsson/anti-slop-go/internal/signature"
)

// The placement rule of a justification comment has one implementation,
// and rules G01 and G11 both read it. A fixture of one rule cannot pin
// it for the other, so the tests below drive the walk directly.
//
// Each call of mark in the source below carries its own label. The want
// values are line offsets from the line of that call: 0 is the line of
// the call itself, and -1 is the line above it.
const stmtLinesSrc = `package p

func mark(label string) bool { return label != "" }

var packageLevel = mark("packageLevel")

func simple() {
	mark("simple")
}

func condition(ready bool) {
	if ready && mark("condition") {
		_ = ready
	}
}

func body(ready bool) {
	if ready {
		mark("body")
	}
}

func clauses(kind int, ch chan int) {
	switch kind {
	case 1:
		mark("caseFirst")
		mark("caseLater")
	}
	select {
	case <-ch:
		mark("commFirst")
	}
}

func literal() {
	go func() {
		mark("literal")
	}()
}
`

func TestEnclosingStmtLines(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", stmtLinesSrc, parser.ParseComments)
	if err != nil {
		t.Fatalf("the fixture source does not parse: %v", err)
	}
	got := stmtLineOffsets(t, fset, file)

	tests := []struct {
		label string
		want  []int
		why   string
	}{
		{"simple", []int{0}, "the statement of the call is the only statement around it"},
		{"condition", []int{0}, "the if holds the call, and the if starts on the line of the call"},
		{"body", []int{0}, "the body of the if is a block, so the walk stops below the if"},
		{"caseFirst", []int{0, -1}, "the clause line sits above the first statement of the clause"},
		{"caseLater", []int{0}, "a later statement of the clause needs its own comment"},
		{"commFirst", []int{0, -1}, "a communication clause holds a statement list too"},
		{"literal", []int{0}, "the body of the literal is a block, so the go statement stays out"},
		{"packageLevel", nil, "a call outside every statement has no statement around it"},
	}
	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			offsets, found := got[tt.label]
			if !found {
				t.Fatalf("the fixture holds no call labelled %q", tt.label)
			}
			if !slices.Equal(offsets, tt.want) {
				t.Errorf("offsets = %v; want %v, because %s", offsets, tt.want, tt.why)
			}
		})
	}
	if len(got) != len(tests) {
		t.Errorf("the fixture holds %d labelled calls; the table names %d", len(got), len(tests))
	}
}

// stmtLineOffsets runs the walk for every labelled call of the file. It
// returns the lines that the walk gave, counted from the line of the
// call, so the table above reads without line numbers.
func stmtLineOffsets(t *testing.T, fset *token.FileSet, file *ast.File) map[string][]int {
	t.Helper()

	offsets := make(map[string][]int)
	insp := inspector.New([]*ast.File{file})
	insp.WithStack([]ast.Node{(*ast.CallExpr)(nil)}, func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push {
			return true
		}
		call, isCall := n.(*ast.CallExpr)
		if !isCall || len(call.Args) != 1 {
			return true
		}
		label, isLabel := call.Args[0].(*ast.BasicLit)
		if !isLabel {
			return true
		}
		callLine := signature.LineOf(fset, call.Pos())
		var relative []int
		for _, line := range signature.EnclosingStmtLines(fset, stack) {
			relative = append(relative, line-callLine)
		}
		// The literal of the label carries its quotes, and the map key
		// drops them.
		offsets[label.Value[1:len(label.Value)-1]] = relative

		return true
	})

	return offsets
}
