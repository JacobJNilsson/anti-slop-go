// Package signature holds the machinery that more than one rule needs.
//
// Two groups live here. This file holds the tests that rules G03
// (noanyparam) and G04 (noanyreturn) share: both rules read a
// signature, and both accept the empty interface where an external API
// sets the shape. The file justify.go holds the justification comment
// contract of docs/spec/003-implementation.md, which every rule with a
// justification uses, and the generated-file test that every rule uses.
//
// The rules stay in their own analyzer packages; only the shared
// machinery lives here. One implementation of a contract cannot drift
// from itself.
package signature

import (
	"go/ast"
	"go/token"
	"go/types"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/tools/go/analysis"
)

// IsEmptyInterface reports whether t is the empty interface. It
// unaliases t, so "any", "interface{}", and an alias of either are one
// type. It never takes the underlying type: a defined type, such as
// "type Payload any", is a domain type, and both rules accept it.
func IsEmptyInterface(t types.Type) bool {
	iface, isInterface := types.Unalias(t).(*types.Interface)
	return isInterface && iface.Empty()
}

// NameCount returns the number of parameters or results a field
// declares. Go groups names, so "a, b any" is one field and two
// entries, and an unnamed entry is one field and one entry.
func NameCount(field *ast.Field) int {
	if len(field.Names) == 0 {
		return 1
	}
	return len(field.Names)
}

// contractMarker matches a line of a doc comment that carries the
// marker CONTRACT:. The signature rules read no marker in a plain
// comment. A doc comment cannot justify on its own, because every
// documented declaration has one, so the marker is the way to justify
// inside a doc comment. The expression follows the contract of
// docs/spec/003-implementation.md: the marker starts a line of the
// text that ast.CommentGroup.Text returns, after the space or the
// star gutter of a block comment.
var contractMarker = regexp.MustCompile(`(?m)^[\s*]*CONTRACT\s*:`)

// articles are the words that a doc comment may put before the name
// of the declaration. golint and revive accept them.
var articles = map[string]bool{"A": true, "An": true, "The": true}

// Contracts answers, for one analysis pass, whether the author may keep
// the empty interface in a signature. It holds the state that answer
// needs: the shared justification tests, and the interfaces of the
// imported packages.
type Contracts struct {
	pass           *analysis.Pass
	justifications *Justifications
	// home widens the interface scan to the package under analysis.
	home       bool
	ifaces     []*types.Interface
	ifacesDone bool
}

// NewContracts prepares the shared tests for one pass. The interface
// scan reads the directly imported packages only. An interface of the
// package under analysis is code the author owns, so it fixes nothing
// that the author cannot change.
//
// Rules G03 and G06 take this entry. Both read a parameter, and the
// author of a local interface can widen the parameter of that
// interface with the parameter of the method.
func NewContracts(pass *analysis.Pass) *Contracts {
	return newContracts(pass, false)
}

// NewContractsWithHome prepares the same tests, and the interface scan
// reads the package under analysis as well.
//
// Rule G09 takes this entry, because it reports a result. A method
// cannot narrow a result that a local interface declares: the concrete
// type no longer satisfies that interface, and the package stops
// compiling. The advice of the rule must compile, so such a method is
// no finding. The scan reads an unexported local interface too, because
// the compiler answers the same way for it. 002 states both stances.
func NewContractsWithHome(pass *analysis.Pass) *Contracts {
	return newContracts(pass, true)
}

func newContracts(pass *analysis.Pass, home bool) *Contracts {
	return &Contracts{pass: pass, home: home, justifications: NewJustifications(pass)}
}

// Generated reports whether pos sits in a generated file. Both rules
// state the shape of hand-written code, so a report against a file that
// a program writes has no reader who can act on it.
func (c *Contracts) Generated(pos token.Pos) bool {
	return c.justifications.Generated(pos)
}

// Justified reports whether a justification comment sits directly
// above the signature at the end of stack. The comment must own its
// line and must end on the line directly above the signature, or above
// the declaration, the field, the specification, or the statement that
// holds it.
//
// A doc comment justifies nothing, unless a line of it carries the
// marker CONTRACT:. Every documented declaration has a doc comment on
// that line, so a test that accepted any text would exempt every
// documented signature. Go states the shape of a doc comment: the text
// starts with the name of the declaration, after an optional article.
// A comment of another shape is a justification. The analyzer cannot
// judge the text; review must.
func (c *Contracts) Justified(stack []ast.Node) bool {
	pos := stack[len(stack)-1].Pos()
	names := declaredNames(stack)
	accept := func(text string) bool {
		return !isDocComment(text, names) || contractMarker.MatchString(text)
	}
	return c.justifications.CommentAboveWhere(pos, justifyLines(c.pass.Fset, stack), accept)
}

// declaredNames returns the names that the declarations of stack
// introduce: a function, a type, the names of a variable specification,
// and the names of a field. A doc comment above any of them starts with
// one of these names.
func declaredNames(stack []ast.Node) map[string]bool {
	names := make(map[string]bool)
	for _, node := range stack {
		switch n := node.(type) {
		case *ast.FuncDecl:
			names[n.Name.Name] = true
		case *ast.TypeSpec:
			names[n.Name.Name] = true
		case *ast.ValueSpec:
			for _, name := range n.Names {
				names[name.Name] = true
			}
		case *ast.Field:
			for _, name := range n.Names {
				names[name.Name] = true
			}
		}
	}
	return names
}

// isDocComment reports whether text has the shape of a doc comment of
// one of names: the first word, after an optional article, is one of
// the names. The first word ends at the first character that cannot be
// part of an identifier.
func isDocComment(text string, names map[string]bool) bool {
	// A block comment keeps its gutter of stars in the text.
	words := strings.Fields(strings.TrimLeft(text, " \t\n*"))
	if len(words) > 0 && articles[words[0]] {
		words = words[1:]
	}
	if len(words) == 0 {
		return false
	}
	return names[identifierPrefix(words[0])]
}

// identifierPrefix returns the identifier at the start of word. A doc
// comment may follow the name with punctuation, such as "Handle:".
func identifierPrefix(word string) string {
	for i, r := range word {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return word[:i]
		}
	}
	return word
}

// Implements reports whether an exported interface of a directly
// imported package declares the method that parent defines, and matches
// accepts the signature that the interface declares. The receiver must
// satisfy the whole interface, so the author cannot change the method.
//
// The caller tests the position first through matches, because that
// test is cheap and it rejects a method that only shares a name.
func (c *Contracts) Implements(parent ast.Node, matches func(declared *types.Signature) bool) bool {
	decl, isDecl := parent.(*ast.FuncDecl)
	if !isDecl || decl.Recv == nil {
		return false
	}
	// SAFETY: the declaration carries a receiver, so the type checker
	// defined a method object for its name. The driver skips a package
	// that does not type-check.
	fn := c.pass.TypesInfo.Defs[decl.Name].(*types.Func)
	// SAFETY: the type of a *types.Func is always a *types.Signature, and
	// the signature of a method carries a receiver.
	recv := fn.Type().(*types.Signature).Recv().Type()
	// An interface may need the method set of the pointer. A pointer
	// receiver gives a pointer to a pointer, whose method set is empty,
	// so the second entry changes no answer there.
	receivers := []types.Type{recv, types.NewPointer(recv)}
	for _, iface := range c.interfaces() {
		if !matches(externalSignature(iface, fn.Name())) {
			continue
		}
		for _, receiver := range receivers {
			if types.Implements(receiver, iface) {
				return true
			}
		}
	}
	return false
}

// interfaces returns the interfaces with a method set that the scan
// reads: the exported ones of the directly imported packages, and every
// one of the package under analysis when the caller asked for the home
// package. The list is built once, and only when a signature needs it.
func (c *Contracts) interfaces() []*types.Interface {
	if c.ifacesDone {
		return c.ifaces
	}
	c.ifacesDone = true
	for _, source := range c.sources() {
		scope := source.Scope()
		for _, name := range scope.Names() {
			declared, isType := scope.Lookup(name).(*types.TypeName)
			// An unexported type of an imported package names no
			// signature that the package under analysis can write.
			if !isType || (!declared.Exported() && source != c.pass.Pkg) {
				continue
			}
			iface, isIface := declared.Type().Underlying().(*types.Interface)
			// The method-set test is load-bearing: types.Implements answers
			// true for a constraint whose method set a type matches, but no
			// value implements a constraint. The method-count test only
			// keeps the candidate list short.
			if !isIface || !iface.IsMethodSet() || iface.NumMethods() == 0 {
				continue
			}
			c.ifaces = append(c.ifaces, iface)
		}
	}
	return c.ifaces
}

// sources returns the packages the interface scan reads. The package
// under analysis comes first, because a method that a local interface
// fixes needs no further search.
func (c *Contracts) sources() []*types.Package {
	imports := c.pass.Pkg.Imports()
	if !c.home {
		return imports
	}

	// The slice of the type checker belongs to the package, so the copy
	// keeps this function from writing to it.
	return append([]*types.Package{c.pass.Pkg}, imports...)
}

// externalSignature returns the signature of the method that iface
// declares under name, or nil when iface declares no such method.
func externalSignature(iface *types.Interface, name string) *types.Signature {
	for i := range iface.NumMethods() {
		method := iface.Method(i)
		if method.Name() != name {
			continue
		}
		// SAFETY: the type of a *types.Func is always a *types.Signature.
		return method.Type().(*types.Signature)
	}
	return nil
}

// justifyLines returns the lines a CONTRACT comment may end directly
// above: the line of the signature itself, and the line of the node
// that holds it.
func justifyLines(fset *token.FileSet, stack []ast.Node) []int {
	lines := []int{LineOf(fset, stack[len(stack)-1].Pos())}
	for i := len(stack) - 1; i >= 0; i-- {
		if !holdsSignature(stack[i]) {
			continue
		}
		lines = append(lines, LineOf(fset, stack[i].Pos()))
		break
	}
	return lines
}

// holdsSignature reports whether a node can carry the comment for a
// signature that starts on a later line, such as a function literal in
// a call that spans lines. A variable specification and a statement
// can. A function declaration and a type specification always start on
// the line of the signature itself. A field can start on an earlier
// line, because gofmt keeps a split name list; the rule then keeps the
// report, and the author moves the comment.
func holdsSignature(node ast.Node) bool {
	if _, isSpec := node.(*ast.ValueSpec); isSpec {
		return true
	}
	_, isStatement := node.(ast.Stmt)
	return isStatement
}
