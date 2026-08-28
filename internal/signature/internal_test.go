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
// Every justification of docs/spec/003-implementation.md uses it, so a
// fixture of one rule cannot pin it for the others.

const trailingMarkerSrc = `package p

var x = 1 // CONTRACT: this comment trails code.
func F(v any) {}
`

// analysistest always runs with a driver that provides ReadFile and
// readable sources, so the fallback and the fail-open path of the
// own-line test need direct tests.
func TestCommentAboveReadsTheOwnLineTestFromTheSource(t *testing.T) {
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
			got := NewJustifications(pass).CommentAbove(file.FileStart, []int{declaration})
			if got != tt.want {
				t.Errorf("CommentAbove = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestCommentAboveWithoutAMarkerAcceptsAnyText pins the entry of the
// rules that read no marker word. A comment that owns its line
// justifies the code below it, whatever its text. A comment that trails
// code still justifies nothing.
func TestCommentAboveWithoutAMarkerAcceptsAnyText(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want bool
	}{
		{"a line comment", "package p\n\n// The loader checks the payload.\nvar x = 1\n", true},
		{"a block comment", "package p\n\n/* The loader checks the payload. */\nvar x = 1\n", true},
		{"a group of two lines", "package p\n\n// The loader\n// checks the payload.\nvar x = 1\n", true},
		{"a comment that trails code", "package p\n\nvar y = 1 // The loader checks the payload.\nvar x = 1\n", false},
		{"no comment", "package p\n\nvar y = 1\nvar x = 1\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "p.go", tt.src, parser.ParseComments)
			if err != nil {
				t.Fatalf("the fixture source does not parse: %v", err)
			}
			declaration := LineOf(fset, file.Decls[len(file.Decls)-1].Pos())
			pass := &analysis.Pass{Fset: fset, Files: []*ast.File{file}, ReadFile: func(string) ([]byte, error) { return []byte(tt.src), nil }}
			if got := NewJustifications(pass).CommentAbove(file.FileStart, []int{declaration}); got != tt.want {
				t.Errorf("CommentAbove = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsDocCommentReadsTheShapeOfADocComment pins the test that keeps
// a doc comment from justifying a signature. Go states the shape: the
// text starts with the name of the declaration, after an optional
// article. The input is the text that ast.CommentGroup.Text returns.
func TestIsDocCommentReadsTheShapeOfADocComment(t *testing.T) {
	names := map[string]bool{"Handle": true, "handler": true}
	tests := []struct {
		name    string
		comment string
		want    bool
	}{
		{"the name first", "// Handle processes one event.", true},
		{"an article before the name", "// A Handle processes one event.", true},
		{"the article The", "// The Handle processes one event.", true},
		{"punctuation after the name", "// Handle: processes one event.", true},
		{"a second name of the group", "// handler holds the callback.", true},
		{"a block comment", "/* Handle processes one event. */", true},
		{"a block gutter of stars", "/*\n * Handle processes one event.\n */", true},
		{"another first word", "// bus.Subscribe sets this signature.", false},
		{"the name inside a sentence", "// The bus calls Handle with any value.", false},
		{"a lowercase form of the name", "// handle processes one event.", false},
		{"a longer word with the name as prefix", "// Handlers process events.", false},
		{"an article alone", "// A", false},
		{"no text", "//", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDocComment(commentText(t, tt.comment), names); got != tt.want {
				t.Errorf("isDocComment = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestContractMarkerStartsALine pins the one place where the signature
// rules still read a marker: inside a doc comment.
func TestContractMarkerStartsALine(t *testing.T) {
	tests := []struct {
		name    string
		comment string
		want    bool
	}{
		{"the second line of a doc comment", "// Handle processes one event.\n// CONTRACT: bus.Subscribe sets this signature.", true},
		{"a space before the colon", "/* CONTRACT : bus.Subscribe sets this signature. */", true},
		{"a block gutter of stars", "/*\n * CONTRACT: bus.Subscribe sets this signature.\n */", true},
		{"a prefix before the marker", "// NOT-CONTRACT: a hyphen is no line start.", false},
		{"the marker inside a sentence", "// The word CONTRACT: here sits inside a sentence.", false},
		{"a lowercase marker", "// contract: the marker is case sensitive.", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := contractMarker.MatchString(commentText(t, tt.comment)); got != tt.want {
				t.Errorf("match = %v, want %v", got, tt.want)
			}
		})
	}
}

// commentText returns the text that go/ast reports for one comment
// group, so a test asserts against the real input of the test.
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
