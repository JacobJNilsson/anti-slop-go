// Package fullstructcomp implements rule G12 of the anti-slop rule set.
// The rule rejects a test that asserts field after field of one value.
// Such a test states one claim for each field it names, and no claim
// about the fields it does not name. One comparison of the whole
// value states the whole expectation.
package fullstructcomp

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"

	"github.com/JacobJNilsson/anti-slop-go/internal/signature"
)

const doc = `require one full structure comparison in a test

A test can assert several fields of one value, one assertion for each
field. Such a test states one claim for each field, and no claim about
the rest. One comparison of the whole value against a want value states
the whole expectation. It reports every wrong field at once.

The analyzer reads test files, which are the files whose name ends in
"_test.go". It reads one function declaration at a time. A function
literal inside a declaration belongs to that declaration, because a
subtest closure is the same test.

It reads two forms of assertion. The first is a call of the equality
family of the testify module: Equal, Equalf, EqualValues, EqualValuesf,
Exactly, and Exactlyf. The analyzer resolves the import path. Another
package with the name testify therefore gives no assertion. The second
is an if statement that compares with "==" or "!=" and calls an Error
or Fatal method of a testing value. The call must be a statement of the
if body itself. A call inside a nested block does not count.

The base of an assertion is the variable at the root of the selector
chain. The field is the whole path under it, so "got.A.B" names the
field "A.B" of the base "got". The analyzer resolves the path through
the type checker, so a promoted field gives the path of the embedded
structure that declares it. A call stops the walk, because the result
of a method is not a field. The analyzer groups by the declared
variable and never by its name, so two values with one name stay apart.

A site that names exactly one field of a base contributes that field. A
site that names two fields of one base contributes nothing, because the
author wrote one compare there already. The analyzer counts distinct
fields, and it reports a base at the number of fields the min setting
sets. The report sits at the first single-field site of the group.

A report also states the cost of the fix. The anchor of a group is the
deepest field path that holds every asserted path, and one comparison
there states every claim of the group. The analyzer counts the
cmpopts.IgnoreFields names that the comparison needs: a subtree with no
asserted field costs one name, and a subtree with one asserted field
pays for each field beside it. A group above the maxignore setting gets
no report, because the ignore list would state more than the assertions
it replaces.

A group that compares one field path against the same path of another
value of the same type pays no such cost. Such a test holds a want
value already, and the ignore list of its fix names the unstable fields
alone, which the analyzer cannot predict.

A type with a usable Equal method answers the comparison itself, and no
option of cmp changes that answer. The analyzer opens such a type at
the anchor and reports the group only at a cost of zero, which means
the test names every field. The message of such a group names no
option.

The analyzer skips a base that a range clause of the function declares.
Such a base holds a table case, and not a produced value. No want value
stands beside it.

No comment stops a report. Five things stop one: the opt-in severity of
the rule, the disable setting, the min setting, the maxignore setting,
and //nolint:antislop on the golangci-lint path. The analyzer skips
generated files.`

// DefaultMin is the number of distinct fields a report needs.
// docs/spec/002-rules.md states the evidence: a checklist of two
// fields is the most frequent form in the measured codebases.
const DefaultMin = 2

// DefaultMaxIgnore is the number of cmpopts.IgnoreFields names that the
// comparison of a report may need. docs/spec/002-rules.md states the
// evidence: most groups of the measured codebases need five names or
// fewer, and every group that a reader judged good sits at five names
// or below.
const DefaultMaxIgnore = 5

// maxDepth bounds the walk that removes the containers of a type. A
// type can hold itself through a slice, as "type L []L" does, and the
// walk of such a type never meets a structure. The cut needs no such
// bound, because it descends only where an asserted path leads.
const maxDepth = 12

// testifyPath is the module the equality family comes from. The
// analyzer resolves this path and never the name of an import.
const testifyPath = "github.com/stretchr/testify"

// testingPath is the package that declares the failure methods of the
// standard library form.
const testingPath = "testing"

// equalMethod is the method that cmp reads instead of the fields of a
// type. A type that declares it needs no option for its own unexported
// fields.
const equalMethod = "Equal"

// The three parts of the message. The second part names the option that
// skips a field the test cannot predict. It stays away where cmp calls
// an Equal method, because no option of cmp changes such a comparison.
// The third part joins the message where cmp.Diff panics on the value.
// That failure arrives at run time, in a test that passes, and the
// author cannot guess the option.
const (
	runMessage = "assertions name %s of %s one at a time; " +
		"compare %s as a whole with cmp.Diff against a want value"
	ignoreMessage     = "; cmpopts.IgnoreFields skips a field the test cannot predict"
	unexportedMessage = "; cmp.Diff panics on the unexported field %s, " +
		"so the comparison needs cmp.AllowUnexported"
)

// equalFamily holds the testify assertions that state the equality of
// two values. A call outside the family states another claim: True
// takes a condition, and Len and Contains read one property. testifylint
// asks the author to write such a claim as Equal, and this rule reads
// it after that change.
var equalFamily = map[string]bool{
	"Equal":        true,
	"Equalf":       true,
	"EqualValues":  true,
	"EqualValuesf": true,
	"Exactly":      true,
	"Exactlyf":     true,
}

// Analyzer is the G12 analyzer. Consumers get it through
// antislop.Analyzers(), and cmd/antislop registers its flags.
var Analyzer = New(DefaultMin, DefaultMaxIgnore)

// New returns an analyzer that reports a base at min distinct fields,
// and at a fix that needs maxIgnore ignore names or fewer. A
// programmatic consumer, such as the golangci-lint plugin, builds one
// instance for each configuration, because two runs can hold different
// numbers and the package-level value is shared.
//
// The instance carries its own flags, which write into the
// configuration of that instance. The flags are the configuration
// surface of cmd/antislop and of go vet -vettool, which read no
// settings file.
func New(min, maxIgnore int) *analysis.Analyzer {
	cfg := &config{}
	a := &analysis.Analyzer{
		Name: "fullstructcomp",
		Doc:  doc,
		URL:  "https://github.com/JacobJNilsson/anti-slop-go/blob/main/docs/spec/002-rules.md",
		Run:  cfg.run,
	}
	a.Flags.IntVar(&cfg.min, "min", min,
		"the number of distinct fields of one value that a report needs")
	a.Flags.IntVar(&cfg.maxIgnore, "maxignore", maxIgnore,
		"the number of cmpopts.IgnoreFields names the whole value comparison may need")

	return a
}

// config holds the settings of one analyzer instance.
type config struct {
	min       int
	maxIgnore int
}

// CONTRACT: analysis.Analyzer.Run fixes this signature.
func (c *config) run(pass *analysis.Pass) (any, error) {
	generated := signature.GeneratedFiles(pass)

	for _, file := range pass.Files {
		if !signature.IsTestFile(pass.Fset.File(file.FileStart)) || generated(file.FileStart) {
			continue
		}
		for _, decl := range file.Decls {
			if fn, isFunc := decl.(*ast.FuncDecl); isFunc {
				c.reportRuns(pass, fn)
			}
		}
	}

	return nil, nil
}

// reportRuns reports the bases of one function declaration that reach
// the setting. A helper such as "func compare(t *testing.T, want, got
// Item)" is a declaration of its own, and its base is a parameter, so
// the rule reads it like a test function.
func (c *config) reportRuns(pass *analysis.Pass, fn *ast.FuncDecl) {
	info := pass.TypesInfo
	runs := &runs{byBase: make(map[*types.Var]*run)}

	ast.Inspect(fn, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			if isEqualCall(info, node) {
				runs.addSite(info, node.Pos(), node.Args)
			}
		case *ast.IfStmt:
			if failsTest(info, node.Body) {
				runs.addSite(info, node.Pos(), comparedOperands(node.Cond))
			}
		}

		return true
	})

	cases := rangeVars(info, fn)
	for _, base := range runs.order {
		found := runs.byBase[base]
		if cases[base] || len(found.fields) < c.min {
			continue
		}
		anchor, at := found.anchor(base)
		if !c.reports(found, at, cut(at, anchor, found.fields, true)) {
			continue
		}
		pass.Reportf(found.first, "%s", message(pass.Pkg, base, found, anchor, at))
	}
}

// reports states the second condition of a report, which reads the cost
// of the fix. cost is the number of cmpopts.IgnoreFields names that one
// comparison at the anchor needs.
//
// A type with a usable Equal method answers the comparison itself, and
// no option of cmp changes that answer. The fix states the same claims
// there only when the test names every field of the type, which is a
// cost of zero. A roundtrip needs no ignore name that the analyzer can
// count, so it passes at any other type.
func (c *config) reports(found *run, at types.Type, cost int) bool {
	if equalStop(at) {
		return cost == 0
	}

	return found.roundtrip >= c.min || cost <= c.maxIgnore
}

// run holds the fields that assertions of one function name of one
// base, one field at a time.
//
// roundtrip counts the sites that compare one field path of the base
// against the same path of another value of the same type. A run of
// such sites carries a want value already, so the cost gate asks
// nothing of it.
type run struct {
	fields    map[string]*fieldPath
	first     token.Pos
	roundtrip int
}

// anchor returns the deepest path that holds every asserted path of the
// run, and the type at that path. One comparison there states every
// claim of the run. The empty path names the base itself.
func (r *run) anchor(base *types.Var) (string, types.Type) {
	var names []string
	at := base.Type()
	first := true
	for _, path := range r.fields {
		parent := path.names[:len(path.names)-1]
		if first {
			names, first = parent, false
		} else {
			names = names[:commonLen(names, parent)]
		}
		if len(names) == 0 {
			at = base.Type()

			continue
		}
		at = path.types[len(names)-1]
	}

	return strings.Join(names, "."), at
}

// commonLen returns the number of leading names that two paths share.
func commonLen(a, b []string) int {
	total := 0
	for total < len(a) && total < len(b) && a[total] == b[total] {
		total++
	}

	return total
}

// fieldPath is one asserted path under a base. names holds the fields
// of the path, and types holds the type of each one. A promoted field
// gives the path of the embedded structure that declares it, so
// "got.Name" and "got.Embedded.Name" name one field through one path.
type fieldPath struct {
	names []string
	types []types.Type
}

// key returns the path as one string, which is the identity of a field
// inside a run.
func (p *fieldPath) key() string { return strings.Join(p.names, ".") }

// runs holds the run of each base of one function, in the order the
// bases first appear. The order makes the reports of a function stable,
// which a map alone does not.
type runs struct {
	byBase map[*types.Var]*run
	order  []*types.Var
}

// addSite records the fields that one assertion site names. A site that
// names exactly one field of a base contributes that field. A site that
// names two or more fields of one base contributes nothing: the author
// compared those fields in one statement already.
func (r *runs) addSite(info *types.Info, pos token.Pos, operands []ast.Expr) {
	named := make(map[*types.Var]map[string]*fieldPath)
	var order []*types.Var
	for _, operand := range operands {
		base, path := baseField(info, operand)
		if base == nil {
			continue
		}
		paths, seen := named[base]
		if !seen {
			paths = make(map[string]*fieldPath)
			named[base] = paths
			order = append(order, base)
		}
		paths[path.key()] = path
	}

	for _, base := range order {
		if len(named[base]) != 1 {
			continue
		}
		for key, path := range named[base] {
			r.add(base, path, pos, roundtrip(named, base, key))
		}
	}
}

// roundtrip reports whether another base of one site names the same
// field path of a value of the same type. Such a site compares a
// produced value against a want value that holds every field of it. The
// ignore list of the fix then names the unstable fields alone, which
// the analyzer cannot predict, so the cost gate reads no cost there.
//
// Both parts of the definition are necessary. A site that reads a field
// of an unrelated value, such as an identifier of another record, states
// no roundtrip, and the fix of it still writes a want value by hand.
func roundtrip(named map[*types.Var]map[string]*fieldPath, base *types.Var, key string) bool {
	for other, paths := range named {
		if other == base || len(paths) != 1 {
			continue
		}
		if _, same := paths[key]; !same {
			continue
		}
		if sameShape(base.Type(), other.Type()) {
			return true
		}
	}

	return false
}

// add records one field of one base. The position of the first
// single-field site of a base is the position of its report, and the
// walk meets the sites in the order of the source.
func (r *runs) add(base *types.Var, path *fieldPath, pos token.Pos, paired bool) {
	found, seen := r.byBase[base]
	if !seen {
		found = &run{fields: make(map[string]*fieldPath), first: pos}
		r.byBase[base] = found
		r.order = append(r.order, base)
	}
	found.fields[path.key()] = path
	if paired {
		found.roundtrip++
	}
}

// message states the finding and the fix. It names the count, the base,
// the value that one comparison reads, and the two options the fix
// needs. The type at the anchor answers the option, because the
// comparison the message asks for reads that type.
func message(home *types.Package, base *types.Var, found *run, anchor string, at types.Type) string {
	text := fmt.Sprintf(runMessage, fieldCount(len(found.fields)), base.Name(), whole(base, anchor))
	if equalStop(at) {
		return text
	}
	text += ignoreMessage
	if field, holds := unexportedField(at, types.RelativeTo(home)); holds {
		text += fmt.Sprintf(unexportedMessage, field)
	}

	return text
}

// whole names the value that one comparison reads. It is the base, or
// the field of the base that holds every asserted path.
func whole(base *types.Var, anchor string) string {
	if anchor == "" {
		return base.Name()
	}

	return base.Name() + "." + anchor
}

// fieldCount writes a field count with the singular form or the plural
// form.
func fieldCount(fields int) string {
	if fields == 1 {
		return "1 field"
	}

	return fmt.Sprintf("%d fields", fields)
}

// baseField returns the variable at the root of a selector chain and
// the path of fields that the chain names under it. It returns nil for
// an expression that names no field of a variable.
//
// The walk stops at a call, because the result of a method is not a
// field of the receiver. It stops at a root that is not a variable,
// such as the name of an imported package.
func baseField(info *types.Info, expr ast.Expr) (*types.Var, *fieldPath) {
	var steps []*fieldPath
	for {
		switch node := ast.Unparen(expr).(type) {
		case *ast.SelectorExpr:
			steps = append(steps, selectorPath(info, node))
			expr = node.X
		case *ast.IndexExpr:
			expr = node.X
		case *ast.StarExpr:
			expr = node.X
		case *ast.Ident:
			base, isVar := info.Uses[node].(*types.Var)
			if !isVar || len(steps) == 0 {
				return nil, nil
			}
			slices.Reverse(steps)

			return base, joinPaths(steps)
		default:
			return nil, nil
		}
	}
}

// selectorPath returns the fields that one selector names. A selector
// of a promoted field names the embedded structures above it as well,
// and the type checker holds the depth of the field, so the search
// reads no level below it.
//
// A selector that names no field gives its own name. Such a selector
// stands under the name of a package, and the walk of the chain rejects
// that root.
func selectorPath(info *types.Info, sel *ast.SelectorExpr) *fieldPath {
	selection, holds := info.Selections[sel]
	if holds && selection.Kind() == types.FieldVal {
		if path, found := promotedPath(selection.Recv(), selection.Obj(), len(selection.Index())); found {
			return path
		}
	}

	return &fieldPath{names: []string{sel.Sel.Name}, types: []types.Type{info.TypeOf(sel)}}
}

// promotedPath returns the path of fields from a type down to one
// field. limit is the depth of the field in the type, so the search
// ends where the field must sit.
func promotedPath(t types.Type, field types.Object, limit int) (*fieldPath, bool) {
	if limit == 0 {
		return nil, false
	}
	for _, candidate := range fieldsOf(t) {
		if candidate == field {
			return &fieldPath{
				names: []string{candidate.Name()},
				types: []types.Type{candidate.Type()},
			}, true
		}
		if !candidate.Embedded() {
			continue
		}
		deeper, found := promotedPath(candidate.Type(), field, limit-1)
		if !found {
			continue
		}

		return &fieldPath{
			names: append([]string{candidate.Name()}, deeper.names...),
			types: append([]types.Type{candidate.Type()}, deeper.types...),
		}, true
	}

	return nil, false
}

// fieldsOf returns the fields that a promoted path passes through. Such
// a path crosses an embedded structure or a pointer to one. A type with
// no structure under it carries no field, and a structure can embed
// such a type.
func fieldsOf(t types.Type) []*types.Var {
	if pointer, isPointer := types.Unalias(t).(*types.Pointer); isPointer {
		t = pointer.Elem()
	}
	structure, isStruct := types.Unalias(t).Underlying().(*types.Struct)
	if !isStruct {
		return nil
	}

	return slices.Collect(structure.Fields())
}

// joinPaths returns the steps of one chain as one path.
func joinPaths(steps []*fieldPath) *fieldPath {
	path := &fieldPath{}
	for _, step := range steps {
		path.names = append(path.names, step.names...)
		path.types = append(path.types, step.types...)
	}

	return path
}

// cut returns the number of cmpopts.IgnoreFields names that one
// comparison at a prefix needs. A path the test asserts costs nothing.
// A subtree that holds no asserted path costs one name, because one
// name covers the whole of it. A subtree that holds one pays for each
// field beside it.
//
// atAnchor marks the type of the comparison itself. The walk opens that
// type whatever methods it carries, because the count states the work
// of a fix that reads every field of it. A field below the anchor keeps
// the Equal stop of cmp.
//
// The walk descends only where an asserted path leads, so the length of
// that path bounds it.
func cut(t types.Type, prefix string, paths map[string]*fieldPath, atAnchor bool) int {
	if _, asserted := paths[prefix]; asserted {
		return 0
	}
	structure := structUnder(t, atAnchor)
	if structure == nil || !under(paths, prefix) {
		return 1
	}

	total := 0
	for field := range structure.Fields() {
		total += cut(field.Type(), join(prefix, field.Name()), paths, false)
	}

	return total
}

// under reports whether an asserted path sits below a prefix. The empty
// prefix names the anchor itself, and every path sits below it.
func under(paths map[string]*fieldPath, prefix string) bool {
	if prefix == "" {
		return len(paths) > 0
	}
	for path := range paths {
		if strings.HasPrefix(path, prefix+".") {
			return true
		}
	}

	return false
}

// join adds one field name to a path.
func join(prefix, name string) string {
	if prefix == "" {
		return name
	}

	return prefix + "." + name
}

// structUnder returns the structure that cmp opens under a type. cmp
// reads the element of a pointer, a slice, an array and a map, and it
// stops at a type with a usable Equal method. The result is nil where
// cmp reads no field, and such a type costs one ignore name.
//
// open ignores the Equal method, and the walk of the anchor takes that
// route. maxDepth bounds the walk, because a type can hold itself
// through a container and never reach a structure.
func structUnder(t types.Type, open bool) *types.Struct {
	for step := 0; step < maxDepth && (open || !hasEqualMethod(t)); step++ {
		switch node := types.Unalias(t).(type) {
		case *types.Named:
			t = node.Underlying()
		case *types.Struct:
			return node
		default:
			elem, isContainer := containerElem(t)
			if !isContainer {
				return nil
			}
			t = elem
		}
	}

	return nil
}

// equalStop reports whether cmp calls an Equal method of a type instead
// of reading the fields of it. cmp reads the element of a container, so
// the method of the element answers for the container as well. The walk
// stops at such a type, and the same walk opens it.
func equalStop(t types.Type) bool {
	return structUnder(t, false) == nil && structUnder(t, true) != nil
}

// sameShape reports whether two bases hold values of one type. cmp
// compares the element of a container, so a pointer, a slice, an array
// and a map of the type of the base carry the same want value.
func sameShape(a, b types.Type) bool {
	return types.Identical(stripped(a), stripped(b))
}

// stripped removes the containers of a type and returns the type under
// them.
func stripped(t types.Type) types.Type {
	for step := 0; step < maxDepth; step++ {
		elem, isContainer := containerElem(t)
		if !isContainer {
			break
		}
		t = elem
	}

	return t
}

// containerElem returns the element type of a container.
func containerElem(t types.Type) (types.Type, bool) {
	switch node := types.Unalias(t).(type) {
	case *types.Pointer:
		return node.Elem(), true
	case *types.Slice:
		return node.Elem(), true
	case *types.Array:
		return node.Elem(), true
	case *types.Map:
		// The walk reads the element and never the key, as the walk of
		// the unexported fields does.
		return node.Elem(), true
	}

	return nil, false
}

// isEqualCall reports whether a call is an assertion of the equality
// family of the testify module.
func isEqualCall(info *types.Info, call *ast.CallExpr) bool {
	sel, isSelector := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	if !isSelector || !equalFamily[sel.Sel.Name] {
		return false
	}

	return fromTestify(info, sel.X)
}

// fromTestify reports whether the value before the dot of an assertion
// belongs to the testify module. The package form names an imported
// package of the module. The receiver form, which "r := require.New(t)"
// builds, holds a value of a type the module declares.
func fromTestify(info *types.Info, receiver ast.Expr) bool {
	if ident, isIdent := ast.Unparen(receiver).(*ast.Ident); isIdent {
		if pkg, isPackage := info.Uses[ident].(*types.PkgName); isPackage {
			return underTestify(pkg.Imported().Path())
		}
	}

	return underTestify(typePackage(info.TypeOf(receiver)))
}

// underTestify reports whether an import path names the testify module
// or a package of it. A path that only starts with the same letters is
// another module.
func underTestify(path string) bool {
	return path == testifyPath || strings.HasPrefix(path, testifyPath+"/")
}

// typePackage returns the import path of the package that declares a
// type. It returns the empty string for a type that no package
// declares, such as an interface written in place.
func typePackage(t types.Type) string {
	if pointer, isPointer := types.Unalias(t).(*types.Pointer); isPointer {
		t = pointer.Elem()
	}
	named, isNamed := types.Unalias(t).(*types.Named)
	if !isNamed || named.Obj().Pkg() == nil {
		return ""
	}

	return named.Obj().Pkg().Path()
}

// failsTest reports whether the body of an if statement calls a method
// that fails the test. The name of the method starts with Error or with
// Fatal, and the value before the dot has a type of the testing
// package. A type of the project can declare a method with the same
// name, so the name alone is not evidence.
func failsTest(info *types.Info, body *ast.BlockStmt) bool {
	for _, stmt := range body.List {
		expr, isExpr := stmt.(*ast.ExprStmt)
		if !isExpr {
			continue
		}
		call, isCall := ast.Unparen(expr.X).(*ast.CallExpr)
		if !isCall {
			continue
		}
		sel, isSelector := ast.Unparen(call.Fun).(*ast.SelectorExpr)
		if !isSelector {
			continue
		}
		if !strings.HasPrefix(sel.Sel.Name, "Error") && !strings.HasPrefix(sel.Sel.Name, "Fatal") {
			continue
		}
		if typePackage(info.TypeOf(sel.X)) == testingPath {
			return true
		}
	}

	return false
}

// comparedOperands returns the operands of every equality test of a
// condition. One condition can hold several tests, such as
// "a.X != w.X || a.Y != w.Y", and the rule reads all of them. Such a
// site names two fields of one base, so it contributes nothing.
func comparedOperands(cond ast.Expr) []ast.Expr {
	var operands []ast.Expr
	ast.Inspect(cond, func(n ast.Node) bool {
		binary, isBinary := n.(*ast.BinaryExpr)
		if !isBinary || (binary.Op != token.EQL && binary.Op != token.NEQ) {
			return true
		}
		operands = append(operands, binary.X, binary.Y)

		return true
	})

	return operands
}

// rangeVars returns the variables that a range clause of a function
// fills. Such a variable holds a table case, and not a produced value.
// Its fields meet values that the case itself carries, and no want
// value stands beside it.
func rangeVars(info *types.Info, fn *ast.FuncDecl) map[*types.Var]bool {
	cases := make(map[*types.Var]bool)
	ast.Inspect(fn, func(n ast.Node) bool {
		stmt, isRange := n.(*ast.RangeStmt)
		if !isRange {
			return true
		}
		for _, expr := range []ast.Expr{stmt.Key, stmt.Value} {
			ident, isIdent := expr.(*ast.Ident)
			if !isIdent {
				continue
			}
			// A clause with ":=" declares the variable, and the type
			// checker holds it in Defs. A clause with "=" writes to a
			// variable of an outer statement, which sits in Uses. The
			// blank identifier declares nothing and sits in neither.
			if variable, isVar := info.Defs[ident].(*types.Var); isVar {
				cases[variable] = true

				continue
			}
			if variable, isVar := info.Uses[ident].(*types.Var); isVar {
				cases[variable] = true
			}
		}

		return true
	})

	return cases
}

// unexportedField returns the first unexported field that cmp meets in
// the graph of a type. cmp.Diff panics on such a field at run time, and
// no compiler reports it, so the message of the rule must name the
// option that the fix needs.
//
// The walk stops at a type that declares an Equal method, as cmp does:
// cmp calls the method and reads no field under it. time.Time is such a
// type.
func unexportedField(t types.Type, qual types.Qualifier) (string, bool) {
	return walkType(t, qual, make(map[types.Type]bool))
}

// walkType walks one type of the graph. The set of the types it read
// already ends the walk of a type that holds itself.
func walkType(t types.Type, qual types.Qualifier, seen map[types.Type]bool) (string, bool) {
	t = types.Unalias(t)
	if seen[t] {
		return "", false
	}
	seen[t] = true

	switch node := t.(type) {
	case *types.Pointer:
		// A pointer type holds the methods of both receivers, and a
		// value type holds the methods of its value receivers only. So a
		// method with a pointer receiver counts here and not below.
		if hasEqualMethod(t) {
			return "", false
		}

		return walkType(node.Elem(), qual, seen)
	case *types.Slice:
		return walkType(node.Elem(), qual, seen)
	case *types.Array:
		return walkType(node.Elem(), qual, seen)
	case *types.Map:
		// The walk reads the element and never the key. cmp compares the
		// keys of a map as keys, and it reads no field under them, so an
		// unexported field of a key type panics for nothing. A run with
		// go-cmp v0.7.0 over a map key with an unexported field returned
		// a diff and no panic.
		return walkType(node.Elem(), qual, seen)
	case *types.Named:
		if hasEqualMethod(node) {
			return "", false
		}

		return walkFields(node.Underlying(), types.TypeString(node, qual), qual, seen)
	case *types.Struct:
		return walkFields(node, types.TypeString(node, qual), qual, seen)
	}

	return "", false
}

// walkFields returns the first unexported field of a struct type, or
// the first one in the graph under its fields. The name is the name the
// message gives the type that holds the field.
func walkFields(t types.Type, name string, qual types.Qualifier, seen map[types.Type]bool) (string, bool) {
	structure, isStruct := t.(*types.Struct)
	if !isStruct {
		return "", false
	}
	for field := range structure.Fields() {
		if !field.Exported() {
			return name + "." + field.Name(), true
		}
		if found, holds := walkType(field.Type(), qual, seen); holds {
			return found, true
		}
	}

	return "", false
}

// hasEqualMethod reports whether a type carries the Equal method that
// cmp calls instead of the fields of the type. cmp calls a method of
// the method set of the value, which takes the value itself, and which
// returns a boolean. A field with that name is not a method, and it
// returns no comparison.
//
// The lookup treats the value as not addressable, because cmp reads a
// value and takes no address of it. A method with a pointer receiver
// therefore counts for the pointer type alone.
func hasEqualMethod(t types.Type) bool {
	obj, _, _ := types.LookupFieldOrMethod(t, false, nil, equalMethod)
	method, isMethod := obj.(*types.Func)
	if !isMethod {
		return false
	}
	shape := method.Signature()

	// The value must fit the parameter. "func (w W) Equal(n int) bool"
	// carries the name and compares another type, so cmp reads the
	// fields of a W and panics on an unexported one.
	return shape.Params().Len() == 1 && shape.Results().Len() == 1 &&
		types.AssignableTo(t, shape.Params().At(0).Type()) &&
		isBoolean(shape.Results().At(0).Type())
}

// isBoolean reports whether a type is a boolean type.
func isBoolean(t types.Type) bool {
	basic, isBasic := types.Unalias(t).Underlying().(*types.Basic)

	return isBasic && basic.Info()&types.IsBoolean != 0
}
