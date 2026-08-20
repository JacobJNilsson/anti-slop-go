// Package signature holds the machinery that more than one rule needs.
//
// Two groups live here. This file holds the tests that rules G03
// (noanyparam) and G04 (noanyreturn) share: both rules read a
// signature, and both accept the empty interface where an external API
// sets the shape. The file justify.go holds the justification comment
// contract of docs/spec/003-implementation.md, which every rule with a
// marker uses, and the generated-file test that every rule uses.
//
// The rules stay in their own analyzer packages; only the shared
// machinery lives here. One implementation of a contract cannot drift
// from itself.
package signature

import (
	"go/ast"
	"go/token"
	"go/types"

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

// contractMarker is the marker of the justification comment for rules
// G03 and G04. NewJustifications holds the contract that every marker
// shares.
const contractMarker = "CONTRACT"

// Contracts answers, for one analysis pass, whether the author may keep
// the empty interface in a signature. It holds the state that answer
// needs: the shared justification tests, and the interfaces of the
// imported packages.
type Contracts struct {
	pass           *analysis.Pass
	justifications *Justifications
	ifaces         []*types.Interface
	ifacesDone     bool
}

// NewContracts prepares the shared tests for one pass.
func NewContracts(pass *analysis.Pass) *Contracts {
	return &Contracts{pass: pass, justifications: NewJustifications(pass, contractMarker)}
}

// Generated reports whether pos sits in a generated file. Both rules
// state the shape of hand-written code, so a report against a file that
// a program writes has no reader who can act on it.
func (c *Contracts) Generated(pos token.Pos) bool {
	return c.justifications.Generated(pos)
}

// Justified reports whether a CONTRACT comment sits directly above the
// signature at the end of stack. The comment must own its line and must
// end on the line directly above the signature, or above the
// declaration, the field, the specification, or the statement that
// holds it. The analyzer cannot judge the text; review must.
func (c *Contracts) Justified(stack []ast.Node) bool {
	pos := stack[len(stack)-1].Pos()
	return c.justifications.MarkedAbove(pos, justifyLines(c.pass.Fset, stack))
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

// interfaces returns the exported interfaces with a method set that the
// directly imported packages declare. The list is built once, and only
// when a signature needs it.
func (c *Contracts) interfaces() []*types.Interface {
	if c.ifacesDone {
		return c.ifaces
	}
	c.ifacesDone = true
	for _, imported := range c.pass.Pkg.Imports() {
		scope := imported.Scope()
		for _, name := range scope.Names() {
			declared, isType := scope.Lookup(name).(*types.TypeName)
			if !isType || !declared.Exported() {
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
