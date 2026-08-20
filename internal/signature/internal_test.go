package signature

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"golang.org/x/tools/go/analysis"
)

// The tests below drive the shared justification machinery directly.
// Every marker of docs/spec/003-implementation.md uses it, so a fixture
// of one rule cannot pin it for the others.

const trailingMarkerSrc = `package p

var x = 1 // CONTRACT: this comment trails code.
func F(v any) {}
`

// analysistest always runs with a driver that provides ReadFile and
// readable sources, so the fallback and the fail-open path of the
// own-line test need direct tests.
func TestMarkedAboveReadsTheOwnLineTestFromTheSource(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", trailingMarkerSrc, parser.ParseComments)
	if err != nil {
		t.Fatalf("the fixture source does not parse: %v", err)
	}
	// The marker sits on line 3 and the declaration on line 4.
	const declaration = 4

	readable := func(string) ([]byte, error) { return []byte(trailingMarkerSrc), nil }
	unreadable := func(string) ([]byte, error) { return nil, errors.New("unreadable") }

	tests := []struct {
		name     string
		readFile func(string) ([]byte, error)
		want     bool
	}{
		{"a readable source rejects the comment that trails code", readable, false},
		{"an unreadable source fails open", unreadable, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass := &analysis.Pass{Fset: fset, Files: []*ast.File{file}, ReadFile: tt.readFile}
			got := NewJustifications(pass, contractMarker).MarkedAbove(file.FileStart, []int{declaration})
			if got != tt.want {
				t.Errorf("MarkedAbove = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestMarkerQuotesTheMarkerWord pins the quote of the marker word. The
// word goes into a regular expression, so an unquoted character with a
// meaning there would match text that holds no marker.
func TestMarkerQuotesTheMarkerWord(t *testing.T) {
	marker := markerExpr("A.C")
	if marker.MatchString("ABC: the text holds no marker.\n") {
		t.Error("the marker matched text that does not hold the marker word")
	}
	if !marker.MatchString("A.C: the text holds the marker.\n") {
		t.Error("the marker did not match its own word")
	}
}

func TestSourceReaderFallsBackToTheFileSystem(t *testing.T) {
	got, err := sourceReader(&analysis.Pass{})("justify.go")
	if err != nil {
		t.Fatalf("read of the package source returned %v", err)
	}
	if len(got) == 0 {
		t.Error("read of the package source returned no bytes")
	}
}
