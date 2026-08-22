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
field "A.B" of the base "got". A call stops the walk, because the
result of a method is not a field. The analyzer groups by the declared
variable and never by its name, so two values with one name stay apart.

A site that names exactly one field of a base contributes that field. A
site that names two fields of one base contributes nothing, because the
author wrote one compare there already. The analyzer counts distinct
fields, and it reports a base at the number of fields the min setting
sets. The report sits at the first single-field site of the group.

The analyzer skips a base that a range clause of the function declares.
Such a base holds a table case, and not a produced value. No want value
stands beside it.

No comment stops a report. Four things stop one: the opt-in severity of
the rule, the disable setting, the min setting, and //nolint:antislop
on the golangci-lint path. The analyzer skips generated files.`

// DefaultMin is the number of distinct fields a report needs.
// docs/spec/002-rules.md states the evidence: a checklist of two
// fields is the most frequent form in the measured codebases.
const DefaultMin = 2

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

// The two parts of the message. The second part joins the first where
// cmp.Diff panics on the value. That failure arrives at run time, in a
// test that passes, and the author cannot guess the option.
const (
	runMessage = "assertions name %s of %s one at a time; " +
		"compare the whole value with cmp.Diff against a want value; " +
		"cmpopts.IgnoreFields skips a field the test cannot predict"
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
// antislop.Analyzers(), and cmd/antislop registers its min flag.
var Analyzer = New(DefaultMin)

// New returns an analyzer that reports a base at min distinct fields. A
// programmatic consumer, such as the golangci-lint plugin, builds one
// instance for each configuration, because two runs can hold different
// numbers and the package-level value is shared.
//
// The instance carries its own min flag, which writes into the
// configuration of that instance. The flag is the configuration surface
// of cmd/antislop and of go vet -vettool, which read no settings file.
func New(min int) *analysis.Analyzer {
	cfg := &config{}
	a := &analysis.Analyzer{
		Name: "fullstructcomp",
		Doc:  doc,
		URL:  "https://github.com/JacobJNilsson/anti-slop-go/blob/main/docs/spec/002-rules.md",
		Run:  cfg.run,
	}
	a.Flags.IntVar(&cfg.min, "min", min,
		"the number of distinct fields of one value that a report needs")

	return a
}

// config holds the settings of one analyzer instance.
type config struct {
	min int
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
		pass.Reportf(found.first, "%s", message(pass.Pkg, base, found))
	}
}

// run holds the fields that assertions of one function name of one
// base, one field at a time.
type run struct {
	fields map[string]bool
	first  token.Pos
}

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
	named := make(map[*types.Var]map[string]bool)
	var order []*types.Var
	for _, operand := range operands {
		base, field := baseField(info, operand)
		if base == nil {
			continue
		}
		fields, seen := named[base]
		if !seen {
			fields = make(map[string]bool)
			named[base] = fields
			order = append(order, base)
		}
		fields[field] = true
	}

	for _, base := range order {
		if len(named[base]) != 1 {
			continue
		}
		for field := range named[base] {
			r.add(base, field, pos)
		}
	}
}

// add records one field of one base. The position of the first
// single-field site of a base is the position of its report, and the
// walk meets the sites in the order of the source.
func (r *runs) add(base *types.Var, field string, pos token.Pos) {
	found, seen := r.byBase[base]
	if !seen {
		found = &run{fields: make(map[string]bool), first: pos}
		r.byBase[base] = found
		r.order = append(r.order, base)
	}
	found.fields[field] = true
}

// message states the finding and the fix. It names the count, the base,
// and the two options the fix needs.
func message(home *types.Package, base *types.Var, found *run) string {
	text := fmt.Sprintf(runMessage, fieldCount(len(found.fields)), base.Name())
	if field, holds := unexportedField(base.Type(), types.RelativeTo(home)); holds {
		text += fmt.Sprintf(unexportedMessage, field)
	}

	return text
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
func baseField(info *types.Info, expr ast.Expr) (*types.Var, string) {
	var path []string
	for {
		switch node := ast.Unparen(expr).(type) {
		case *ast.SelectorExpr:
			path = append(path, node.Sel.Name)
			expr = node.X
		case *ast.IndexExpr:
			expr = node.X
		case *ast.StarExpr:
			expr = node.X
		case *ast.Ident:
			base, isVar := info.Uses[node].(*types.Var)
			if !isVar || len(path) == 0 {
				return nil, ""
			}
			slices.Reverse(path)

			return base, strings.Join(path, ".")
		default:
			return nil, ""
		}
	}
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
