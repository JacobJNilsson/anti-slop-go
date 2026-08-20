package safetyassert

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"golang.org/x/tools/go/analysis"
)

// analysistest always runs with a driver that provides ReadFile, so the
// fallback and the unreadable-source path need direct tests.

func TestSourceReaderPrefersThePassReader(t *testing.T) {
	want := []byte("from the pass")
	pass := &analysis.Pass{ReadFile: func(string) ([]byte, error) { return want, nil }}

	got, err := sourceReader(pass)("any name")
	if err != nil {
		t.Fatalf("read returned %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("read returned %q, want %q", got, want)
	}
}

func TestSourceReaderFallsBackToTheFileSystem(t *testing.T) {
	got, err := sourceReader(&analysis.Pass{})("safetyassert.go")
	if err != nil {
		t.Fatalf("read of the package source returned %v", err)
	}
	if len(got) == 0 {
		t.Error("read of the package source returned no bytes")
	}
}

const trailingMarkerSrc = `package p

func f(x any) {
	g() // SAFETY: this comment trails the statement beside it.
	a := x.(T)
	_ = a
}
`

// The comment above the assertion is not on its own line. With the
// source at hand the index rejects it. Without the source the index
// cannot tell, and it must fail open instead of reporting.
func TestCommentIndexUsesTheSourceForTheOwnLineTest(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", trailingMarkerSrc, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse of the fixture source returned %v", err)
	}
	tokenFile := fset.File(file.FileStart)
	assertionLine := lineOf(fset, file.Comments[0].End()) + 1

	readable := func(string) ([]byte, error) { return []byte(trailingMarkerSrc), nil }
	unreadable := func(string) ([]byte, error) { return nil, errors.New("unreadable") }

	tests := []struct {
		name     string
		readFile func(string) ([]byte, error)
		want     bool
	}{
		{"readable source rejects the trailing comment", readable, false},
		{"unreadable source fails open", unreadable, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			index := newCommentIndex(fset, []*ast.File{file}, tt.readFile)
			if got := index.hasSafetyAbove(tokenFile, []int{assertionLine}); got != tt.want {
				t.Errorf("hasSafetyAbove = %v, want %v", got, tt.want)
			}
		})
	}
}
