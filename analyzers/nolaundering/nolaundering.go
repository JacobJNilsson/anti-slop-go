// Package nolaundering implements rule G05 of the anti-slop rule set.
// The rule rejects a value that widens to an interface and comes back
// through an assertion. The widening destroys evidence the program
// already had, and the assertion makes a guess of it again.
package nolaundering

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const doc = `reject a widening step that an assertion takes back

A value that the code knows must not pass through any, or through a
broader interface, and come back through a type assertion. The widening
throws the evidence away, and the assertion guesses it again. Keep the
value in its own type.

The analyzer reports two shapes.

The chained shape puts the widening in the operand of the assertion,
such as "any(v).(T)" and "x.(any).(T)". The rule reports the assertion
when the operand widens a type that the code knows. An assertion that
only narrows an interface has no widening step, such as
"r.(io.ReadWriter)" for an io.Reader. The rule accepts it, and rule
G01 owns the justification.

The binding shape puts the widening in a local variable, such as
"var v any = user". An assertion takes the type back later in the same
function. The rule reports the assertion and names the line of the
widening. It tracks a variable only when every assignment in the
function widens the same concrete type. A variable that also takes a
value the function cannot know holds a real question, and the rule
accepts the assertion on it. Such a value comes from a parameter, from
a call that returns any, from a map read, or from a channel receive. A
narrower interface, a type parameter, and a second concrete type leave
a question too, so they also stop the report.

The rule reports an assertion that takes a concrete type back, and a
type switch, whose cases take concrete types back. An assertion to
another interface asks whether the value satisfies that interface. That
question stands on its own, so the rule leaves it.

The comma-ok form is a report too, because the widening already
answered the question that it asks. A SAFETY comment does not stop a
report: the fix is to delete the widening, not to justify the
assertion.

A type parameter stops a report. The analyzer leaves a widening alone
when a type parameter appears in the type that the widening hides. It
leaves the widening alone when a type parameter appears in the type
that the assertion names too. Go reads and builds a value of such a
type through an interface only. A generic function that launders a
value of another type still gets a report.

The analyzer reads one function at a time. It follows no value through
a parameter, a result, a struct field, a slice, a map, or a channel. A
function literal that captures the variable stops the report too. The
analyzer skips generated files.`

// Analyzer is the G05 analyzer. Consumers get it through
// antislop.Analyzers().
var Analyzer = &analysis.Analyzer{
	Name:     "nolaundering",
	Doc:      doc,
	URL:      "https://github.com/JacobJNilsson/anti-slop-go/blob/main/docs/spec/002-rules.md",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// The two messages the rule emits. Each one names the type that the
// widening hides, because that type is the fix.
const (
	chainMessage   = "this %s takes back a value that the operand widens from %s to %s; %s"
	bindingMessage = "this %s takes back a value that %s widens from %s to %s at line %d; %s"
)

const advice = "remove the widening"

// CONTRACT: analysis.Analyzer.Run fixes this signature.
func run(pass *analysis.Pass) (any, error) {
	// SAFETY: inspect.Analyzer is in Requires, so the driver always
	// supplies its result, and that result is an *inspector.Inspector.
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	generated := generatedFiles(pass)
	laundered := trackBindings(pass, insp)

	insp.Preorder([]ast.Node{(*ast.TypeAssertExpr)(nil)}, func(n ast.Node) {
		// SAFETY: the node filter above admits this type only.
		assert := n.(*ast.TypeAssertExpr)
		if generated[pass.Fset.File(assert.Pos())] {
			return
		}
		// A type switch holds an assertion with no target type. Its
		// cases take concrete types back, so the rule reads it.
		kind := "type switch"
		if assert.Type != nil {
			kind = "assertion"
			target := pass.TypesInfo.TypeOf(assert.Type)
			if interfaceOf(target) != nil {
				// An assertion to an interface asks whether the value
				// satisfies the interface. That question stands on its
				// own, and Go states no negative answer at compile time.
				return
			}
			if mentionsTypeParam(target) {
				// Go builds a parameterized value from a concrete one
				// through an interface only, so the widening is no
				// choice of the author.
				return
			}
		}
		// Report at the .( token, not at the start of the operand. The
		// operand can span several lines, and the reader needs the
		// position of the step that this rule rejects.
		if from, to, widens := chainWidening(pass.TypesInfo, assert.X); widens {
			if mentionsTypeParam(from) {
				// Go reads a parameterized value through an interface
				// only, so the widening is no choice of the author.
				return
			}
			pass.Reportf(assert.Lparen, chainMessage,
				kind, name(pass, from), name(pass, to), advice)
			return
		}
		if b, ok := laundered[assert]; ok {
			first := b.widenings[0]
			// The line comes from the adjusted position, because the
			// driver prints the adjusted position in the header of the
			// diagnostic. A //line directive must move both.
			pass.Reportf(assert.Lparen, bindingMessage,
				kind, b.name, name(pass, first.from), name(pass, b.varType),
				pass.Fset.PositionFor(first.pos, true).Line, advice)
		}
	})
	return nil, nil
}

// mentionsTypeParam reports whether a type parameter appears in a
// type. Go reads and builds a value of such a type through an
// interface only, so a widening of one is no choice of the author.
//
// The walk covers the type constructors that an assertion names: a
// named type carries its type arguments, a map carries two types, a
// struct and a signature carry their members, and every other
// constructor carries one element type.
func mentionsTypeParam(t types.Type) bool {
	switch x := types.Unalias(t).(type) {
	case *types.TypeParam:
		return true
	case *types.Named:
		args := x.TypeArgs()
		return mentionsAny(args.Len(), args.At)
	case *types.Map:
		return mentionsTypeParam(x.Key()) || mentionsTypeParam(x.Elem())
	case *types.Struct:
		return mentionsAny(x.NumFields(), func(i int) types.Type { return x.Field(i).Type() })
	case *types.Tuple:
		return mentionsAny(x.Len(), func(i int) types.Type { return x.At(i).Type() })
	case *types.Signature:
		return mentionsTypeParam(x.Params()) || mentionsTypeParam(x.Results())
	case interface{ Elem() types.Type }:
		// A pointer, a slice, an array, and a channel.
		return mentionsTypeParam(x.Elem())
	}
	return false
}

// mentionsAny reports whether a type parameter appears in one of the
// n types that at returns.
func mentionsAny(n int, at func(int) types.Type) bool {
	for i := range n {
		if mentionsTypeParam(at(i)) {
			return true
		}
	}
	return false
}

// chainWidening reports whether the operand of an assertion widens a
// type that the code knows, and returns the type it hides and the
// interface it hides it in.
//
// A test of the steps themselves would be redundant. Where no step
// hides a type, source returns the operand, the two types are one
// type, and strictlyBroader rejects a type against itself.
func chainWidening(info *types.Info, operand ast.Expr) (from, to types.Type, widens bool) {
	from, to = info.TypeOf(source(info, operand)), info.TypeOf(operand)
	if !strictlyBroader(to, from) {
		return nil, nil, false // The step hides nothing, such as any(v) for an any.
	}
	return from, to, true
}

// widening is one step that puts a known type in an interface.
type widening struct {
	from types.Type
	pos  token.Pos
}

// binding is a local variable that the analyzer tracks. The analyzer
// reports an assertion on the variable when every assignment widens a
// type that the code knows.
type binding struct {
	name      string
	declFunc  ast.Node // The function declaration or literal that declares it.
	varType   types.Type
	widenings []widening
	asserts   []*ast.TypeAssertExpr
	dropped   bool
}

// collector holds the state of the binding walk.
type collector struct {
	pass     *analysis.Pass
	bindings map[types.Object]*binding
}

// trackBindings returns, for each assertion that takes a laundered
// binding back, the binding it takes back.
func trackBindings(pass *analysis.Pass, insp *inspector.Inspector) map[*ast.TypeAssertExpr]*binding {
	c := &collector{pass: pass, bindings: make(map[types.Object]*binding)}

	nodes := []ast.Node{
		(*ast.ValueSpec)(nil),
		(*ast.AssignStmt)(nil),
		(*ast.RangeStmt)(nil),
		(*ast.UnaryExpr)(nil),
		(*ast.Ident)(nil),
		(*ast.TypeAssertExpr)(nil),
	}
	insp.WithStack(nodes, func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push {
			return true
		}
		switch node := n.(type) {
		case *ast.ValueSpec:
			c.declare(node, enclosingFunc(stack))
		case *ast.AssignStmt:
			c.assign(node, enclosingFunc(stack))
		case *ast.RangeStmt:
			// A range clause fills the variable from a source that the
			// analyzer does not read, so it drops the binding.
			c.drop(node.Key)
			c.drop(node.Value)
		case *ast.UnaryExpr:
			if node.Op == token.AND {
				// The address escapes, so another statement can put
				// any value in the variable.
				c.drop(node.X)
			}
		case *ast.Ident:
			c.use(node, enclosingFunc(stack))
		case *ast.TypeAssertExpr:
			c.assertion(node)
		}
		return true
	})

	laundered := make(map[*ast.TypeAssertExpr]*binding)
	for _, b := range c.bindings {
		if b.dropped || len(b.widenings) == 0 {
			continue
		}
		for _, assert := range b.asserts {
			laundered[assert] = b
		}
	}
	return laundered
}

// declare tracks the variables of a var declaration and records the
// value each one takes.
func (c *collector) declare(spec *ast.ValueSpec, fn ast.Node) {
	if fn == nil {
		return // A package-level variable belongs to no function.
	}
	for i, ident := range spec.Names {
		b := c.track(ident, fn)
		switch {
		case b == nil:
			// The name declares no variable, or it declares a blank.
		case len(spec.Values) == 0:
			// The zero value of an interface is nil. An assertion on
			// nil fails, so the zero value justifies no assertion and
			// counts as no assignment.
		case len(spec.Values) == len(spec.Names):
			c.record(b, spec.Values[i])
		default:
			b.dropped = true // One call fills every name.
		}
	}
}

// assign records the value that each variable of an assignment takes.
func (c *collector) assign(stmt *ast.AssignStmt, fn ast.Node) {
	for i, lhs := range stmt.Lhs {
		ident, isIdent := lhs.(*ast.Ident)
		if !isIdent {
			continue // A field or an element is out of scope.
		}
		b := c.track(ident, fn)
		if b == nil {
			b = c.bindings[c.object(ident)]
		}
		if b == nil {
			continue
		}
		if len(stmt.Lhs) != len(stmt.Rhs) {
			// One call, one map read, or one channel receive fills
			// every name, and the analyzer does not read the parts.
			b.dropped = true
			continue
		}
		c.record(b, stmt.Rhs[i])
	}
}

// track starts a binding for a variable that this declaration
// declares. It returns nil for every other identifier, which includes
// the blank identifier, a constant, and a name that an assignment
// reuses.
func (c *collector) track(ident *ast.Ident, fn ast.Node) *binding {
	// SAFETY: a missing entry gives a nil object, which the comma-ok
	// form of the assertion reports as false.
	obj, isVar := c.pass.TypesInfo.Defs[ident].(*types.Var)
	if !isVar {
		return nil
	}
	b := &binding{name: ident.Name, declFunc: fn, varType: obj.Type()}
	c.bindings[obj] = b
	return b
}

// record adds the widening that an assigned value performs, or drops
// the binding when the value hides no type that the code knows.
//
// Complete evidence is one concrete type. A narrower interface and a
// type parameter both leave the assertion a real question, because
// neither one names the type that the variable holds.
func (c *collector) record(b *binding, value ast.Expr) {
	from := c.pass.TypesInfo.TypeOf(source(c.pass.TypesInfo, value))
	if !strictlyBroader(b.varType, from) || interfaceOf(from) != nil {
		b.dropped = true
		return
	}
	if len(b.widenings) > 0 && !types.Identical(b.widenings[0].from, from) {
		// Two types make a union, and the assertion separates them.
		b.dropped = true
		return
	}
	// The walk reads the file in source order, so the first widening
	// of the list is the first one of the function.
	b.widenings = append(b.widenings, widening{from: from, pos: value.Pos()})
}

// use drops a binding that a function literal reads. The analyzer
// reads one function at a time, so it cannot see what the literal does
// with the variable, or when the program runs the literal.
func (c *collector) use(ident *ast.Ident, fn ast.Node) {
	if b := c.bindings[c.object(ident)]; b != nil && b.declFunc != fn {
		b.dropped = true
	}
}

// drop rejects the binding that an expression names, if it names one.
func (c *collector) drop(e ast.Expr) {
	ident, isIdent := e.(*ast.Ident)
	if !isIdent {
		return
	}
	if b := c.bindings[c.object(ident)]; b != nil {
		b.dropped = true
	}
}

// assertion attaches an assertion to the binding its operand names.
func (c *collector) assertion(assert *ast.TypeAssertExpr) {
	ident, isIdent := source(c.pass.TypesInfo, assert.X).(*ast.Ident)
	if !isIdent {
		return
	}
	if b := c.bindings[c.object(ident)]; b != nil {
		b.asserts = append(b.asserts, assert)
	}
}

// object returns the object an identifier names, whether the
// identifier declares it or reads it.
func (c *collector) object(ident *ast.Ident) types.Object {
	if obj := c.pass.TypesInfo.Defs[ident]; obj != nil {
		return obj
	}
	return c.pass.TypesInfo.Uses[ident]
}

// enclosingFunc returns the function declaration or function literal
// that holds the node, and nil for a node at package level.
func enclosingFunc(stack []ast.Node) ast.Node {
	for i := len(stack) - 1; i >= 0; i-- {
		switch stack[i].(type) {
		case *ast.FuncDecl, *ast.FuncLit:
			return stack[i]
		}
	}
	return nil
}

// source follows the widening steps that an expression puts on a
// value: an explicit conversion to an interface type, and a type
// assertion to an interface that the value satisfies already. It
// returns the expression under the steps.
func source(info *types.Info, e ast.Expr) ast.Expr {
	e = ast.Unparen(e)
	switch x := e.(type) {
	case *ast.CallExpr:
		if len(x.Args) == 1 && isTypeExpr(info, x.Fun) && interfaceOf(info.TypeOf(x.Fun)) != nil {
			return source(info, x.Args[0])
		}
	case *ast.TypeAssertExpr:
		// A type switch holds the only assertion with no target type,
		// and such an assertion is the operand of nothing.
		if x.Type != nil {
			target := interfaceOf(info.TypeOf(x.Type))
			if target != nil && types.Implements(info.TypeOf(x.X), target) {
				return source(info, x.X)
			}
		}
	}
	return e
}

// isTypeExpr reports whether an expression names a type, which makes
// the call around it a conversion and not a call.
func isTypeExpr(info *types.Info, e ast.Expr) bool {
	tv, recorded := info.Types[e]
	return recorded && tv.IsType()
}

// strictlyBroader reports whether the interface "to" holds less
// evidence than the type "from". An interface that also accepts "to"
// is the same width and hides nothing. Untyped nil names no type, so
// it has no evidence to hide.
//
// Go assignability puts a value in an interface only when the value
// implements it, so the Implements test below holds on every path the
// analyzer takes. It stays as a guard on the type information.
func strictlyBroader(to, from types.Type) bool {
	target := interfaceOf(to)
	if target == nil || untyped(from) || !types.Implements(from, target) {
		return false
	}
	origin := interfaceOf(from)
	return origin == nil || !types.Implements(to, origin)
}

// interfaceOf returns the interface that a type describes, and nil for
// every other type. The underlying type of a type parameter is its
// constraint, which is the evidence the parameter carries.
func interfaceOf(t types.Type) *types.Interface {
	iface, _ := types.Unalias(t).Underlying().(*types.Interface)
	return iface
}

// untyped reports whether a type is untyped. An untyped constant takes
// its default type before it reaches an interface, so untyped nil is
// the only value that arrives here without a type of its own.
func untyped(t types.Type) bool {
	basic, isBasic := types.Unalias(t).(*types.Basic)
	return isBasic && basic.Info()&types.IsUntyped != 0
}

// name returns the type in the terms of the reader: a type of the
// analysed package prints without its path.
func name(pass *analysis.Pass, t types.Type) string {
	return types.TypeString(t, types.RelativeTo(pass.Pkg))
}

// generatedFiles returns the generated files of the pass. The rule
// states the shape of hand-written code, so a report against a file
// that a program writes has no reader who can act on it.
//
// The set holds each token.File itself, not its name. A //line
// directive changes the name that token.Position reports, so a
// comparison of names can exempt the wrong file in both directions.
func generatedFiles(pass *analysis.Pass) map[*token.File]bool {
	files := make(map[*token.File]bool)
	for _, f := range pass.Files {
		if ast.IsGenerated(f) {
			files[pass.Fset.File(f.FileStart)] = true
		}
	}
	return files
}
