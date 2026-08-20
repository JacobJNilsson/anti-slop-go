package signature

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
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

// TestMarkerNeedsTheMarkerAtTheStartOfALine pins the marker contract
// against the text that go/ast gives the analyzer.
// ast.CommentGroup.Text removes the comment markers and the first space
// of a line comment. It keeps the rest of the leading text of a line.
// The expression must therefore accept the space of a block comment and
// the star of a block gutter.
func TestMarkerNeedsTheMarkerAtTheStartOfALine(t *testing.T) {
	tests := []struct {
		name    string
		comment string
		want    bool
	}{
		{"a line comment", "// SAFETY: the loader checks the payload.", true},
		{"no space after the slashes", "//SAFETY: the loader checks the payload.", true},
		{"more spaces after the slashes", "//    SAFETY: the loader checks the payload.", true},
		{"a space before the colon", "/* SAFETY : the loader checks the payload. */", true},
		{"a block gutter of stars", "/*\n * SAFETY: the loader checks the payload.\n */", true},
		{"the second line of a group", "// The loader checks the payload.\n// SAFETY: only T values arrive.", true},
		{"a prefix before the marker", "// NOT-SAFETY: a hyphen is no line start.", false},
		{"the marker inside a sentence", "// The word SAFETY: here sits inside a sentence.", false},
		{"no colon", "// SAFETY needs a colon.", false},
		{"a lowercase marker", "// safety: the marker is case sensitive.", false},
	}
	marker := markerExpr("SAFETY")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := commentText(t, tt.comment)
			if got := marker.MatchString(text); got != tt.want {
				t.Errorf("match of %q = %v, want %v", text, got, tt.want)
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

// commentText returns the text that go/ast reports for one comment
// group, so the test asserts against the real input of the marker.
func commentText(t *testing.T, comment string) string {
	t.Helper()
	src := "package p\n\n" + comment + "\nvar x = 1\n"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("the fixture source does not parse: %v", err)
	}
	if len(file.Comments) != 1 {
		t.Fatalf("the fixture holds %d comment groups, want 1", len(file.Comments))
	}
	return file.Comments[0].Text()
}

// The interface scan has two entries. NewContracts reads the directly
// imported packages, and NewContractsWithHome reads the package under
// analysis as well. Rules G03 and G06 take the first entry, and rule
// G09 takes the second. 002 states why the two stances differ, and the
// fixture suites of those rules hold the integrated behaviour. This
// test pins the difference itself, on one package with no import.
const homeInterfaceSrc = `package p

type Store interface{ Load() string }

// Sink is an exported interface of the package under analysis.
type Sink interface {
	Next() Store
	Kind() string
}

// stream is unexported. The author of a method cannot narrow a result
// that it fixes either, so the scan reads it too.
type stream interface {
	Head() Store
}

type sink struct{}

func (sink) Next() Store  { return nil }
func (sink) Kind() string { return "sink" }

// half declares the method of Sink and implements no interface.
type half struct{}

func (half) Next() Store { return nil }

type list struct{}

func (list) Head() Store { return nil }
`

func TestContractsReadTheHomePackageOnlyThroughTheHomeEntry(t *testing.T) {
	tests := []struct {
		receiver string
		method   string
		want     bool
	}{
		{"sink", "Next", true},
		{"list", "Head", true},
		{"half", "Next", false},
	}

	for _, tt := range tests {
		t.Run(tt.receiver+"."+tt.method, func(t *testing.T) {
			pass := checkPackage(t, homeInterfaceSrc)
			decl := methodDecl(t, pass, tt.receiver, tt.method)
			declares := func(declared *types.Signature) bool { return declared != nil }

			if got := NewContracts(pass).Implements(decl, declares); got {
				t.Error("NewContracts read an interface of the package under analysis")
			}
			if got := NewContractsWithHome(pass).Implements(decl, declares); got != tt.want {
				t.Errorf("NewContractsWithHome(...).Implements = %v, want %v", got, tt.want)
			}
		})
	}
}

// checkPackage type-checks one source file and builds the pass that the
// shared tests read.
func checkPackage(t *testing.T, src string) *analysis.Pass {
	t.Helper()

	fset := token.NewFileSet()
	file := parseFile(t, fset, src)
	info := &types.Info{
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
		Types: make(map[ast.Expr]types.TypeAndValue),
	}
	pkg, err := new(types.Config).Check("p", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatalf("the fixture source does not type-check: %v", err)
	}

	return &analysis.Pass{Fset: fset, Files: []*ast.File{file}, Pkg: pkg, TypesInfo: info}
}

// parseFile returns the syntax of the fixture source.
func parseFile(t *testing.T, fset *token.FileSet, src string) *ast.File {
	t.Helper()

	file, err := parser.ParseFile(fset, "p.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("the fixture source does not parse: %v", err)
	}

	return file
}

// methodDecl returns the declaration of one method of the fixture.
func methodDecl(t *testing.T, pass *analysis.Pass, receiver, method string) *ast.FuncDecl {
	t.Helper()

	for _, decl := range pass.Files[0].Decls {
		fn, isFunc := decl.(*ast.FuncDecl)
		if !isFunc || fn.Recv == nil || fn.Name.Name != method {
			continue
		}
		if name, isName := fn.Recv.List[0].Type.(*ast.Ident); isName && name.Name == receiver {
			return fn
		}
	}
	t.Fatalf("the fixture declares no method %s.%s", receiver, method)

	return nil
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
