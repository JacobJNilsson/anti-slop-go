package signature

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"golang.org/x/tools/go/analysis"
)

// analysistest always runs with a driver that provides ReadFile and
// readable sources, so the fallback and the fail-open path of the
// own-line test need direct tests.

const trailingMarkerSrc = `package p

var x = 1 // CONTRACT: this comment trails code.
func F(v any) {}
`

func TestCommentIndexReadsTheOwnLineTestFromTheSource(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", trailingMarkerSrc, parser.ParseComments)
	if err != nil {
		t.Fatalf("the fixture source does not parse: %v", err)
	}
	tokenFile := fset.File(file.FileStart)
	// The marker sits on line 3 and the declaration on line 4.
	const declaration = 4

	readable := &analysis.Pass{
		Fset:     fset,
		Files:    []*ast.File{file},
		ReadFile: func(string) ([]byte, error) { return []byte(trailingMarkerSrc), nil },
	}
	if newCommentIndex(readable).markedAbove(tokenFile, []int{declaration}) {
		t.Error("markedAbove accepted a comment that trails code")
	}

	unreadable := &analysis.Pass{
		Fset:     fset,
		Files:    []*ast.File{file},
		ReadFile: func(string) ([]byte, error) { return nil, errors.New("unreadable") },
	}
	if !newCommentIndex(unreadable).markedAbove(tokenFile, []int{declaration}) {
		t.Error("markedAbove rejected the marker with no source; the own-line test must fail open")
	}
}

func TestSourceReaderFallsBackToTheFileSystem(t *testing.T) {
	got, err := sourceReader(&analysis.Pass{})("signature.go")
	if err != nil {
		t.Fatalf("read of the package source returned %v", err)
	}
	if len(got) == 0 {
		t.Error("read of the package source returned no bytes")
	}
}
