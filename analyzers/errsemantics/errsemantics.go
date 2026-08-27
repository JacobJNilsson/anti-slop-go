// Package errsemantics implements rule G13 of the anti-slop rule set.
// The rule rejects a test that decides which error it got from the
// words of the message. The message is prose for a human, and no API
// promises it. errors.Is names a sentinel, and errors.As names a type,
// so both survive a reword.
package errsemantics

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/JacobJNilsson/anti-slop-go/internal/pathmatch"
	"github.com/JacobJNilsson/anti-slop-go/internal/signature"
)

const doc = `reject a test that asserts the text of an error

A test that matches the message of an error passes for another error
with the same words. It fails for the right error after a reword.
The identity of the error is the evidence: errors.Is names a sentinel,
and errors.As names a type. Rule G10 owns the type-assertion half of
this idea, and this rule owns the string half.

The rule reads test files only, and it skips generated files. A test
file is a file whose name ends in _test.go. Every file of a package
that the testpackages flag names is a test file as well. The golangci-lint plugin
takes the same patterns in the test-packages setting. A package that
serves tests and carries no _test.go name, such as a shared suite,
needs that entry.

The rule reads the static type of the operand, and reports the
predeclared error type and an alias of it. That is the narrow test of
rule G10. An interface that declares Error() string under another name
is another type.

The rule reports these forms by default:

  - an err.Error() call as an argument of strings.Contains,
    strings.HasPrefix, strings.HasSuffix, or strings.EqualFold;
  - an err.Error() call as an argument of regexp.MatchString or of the
    MatchString method of regexp.Regexp;
  - ErrorContains, ErrorContainsf, Regexp, and Regexpf of testify, on
    an argument that carries an error.

The equality setting adds two more forms, and it is off by default. A
package that tests its own message text writes them, and the Go wiki
page TestComments accepts that:

  - an err.Error() call in a == or != comparison;
  - EqualError, EqualErrorf, Equal, and Equalf of testify, on an
    err.Error() call.

An Error call that reaches a printer is no assertion. The rule reads
the direct argument of a reported call and the direct operand of a
reported comparison, so t.Errorf and fmt.Sprintf stay clean. A message
that flows through a variable stays clean as well.

A testify call carries its failure message after the values it asserts
on. The rule reads the first three arguments of such a call, and the
first two of the receiver form of Assertions. An error in the failure
message is diagnostic output, and it decides nothing.

No comment stops a report. errors.Is exists in every version of Go
that this module supports, so a marker would justify a shape that has
a fix. The escapes are the severity of the rule, the disable setting,
and the equality setting.

The rule is opt-in. The wiki page accepts one shape the forms cannot
separate: a check that a message of the package under test holds a
property, such as the name of a parameter. A project that turns the
rule on decides that its tests read no message at all. 002 holds the
measurement behind that decision.`

// The import paths the rule resolves. The rule reads the package of
// the object the type checker resolved, so a local package with one of
// these names reports nothing.
const (
	stringsPath   = "strings"
	regexpPath    = "regexp"
	testifyPrefix = "github.com/stretchr/testify/"
)

// testifyAssertArgs is the number of leading arguments of a testify
// call that carry the assertion. The package form takes the testing
// value and the two values it asserts on. The receiver form takes one
// less, because the Assertions value holds the testing value.
const testifyAssertArgs = 3

// The two messages the rule emits. Each one names errors.Is and
// errors.As, which are the fix. The equality message adds the setting,
// because a package that tests its own message text keeps that form.
const (
	textMessage = "this test reads the text of an error, and a reword of the message breaks it; " +
		"assert the identity with errors.Is and a sentinel, or the type with errors.As and a target variable"
	equalityMessage = "this test compares the text of an error; a reword of the message breaks it; " +
		"assert the identity with errors.Is and a sentinel, or the type with errors.As and a target variable; " +
		"a package that tests its own message text keeps this form and leaves " + equalityNames + " off"
)

// equalityNames names both spellings of the setting. The reader of a
// diagnostic runs one of two tools, and each tool takes its own
// spelling. A message with one name leaves the other reader without a
// working setting.
const equalityNames = "errsemantics-equality (-errsemantics.equality outside golangci-lint)"

// The functions of each reported group, by the name of the object.
//
// strings holds no other Contains. regexp holds MatchString as a
// function and as a method of Regexp. The name and the package
// therefore name the object together. regexp.Match takes bytes, so an
// error message reaches it through a conversion. That is no direct
// argument and no finding.
var (
	stringsPredicates = map[string]bool{
		"Contains":  true,
		"HasPrefix": true,
		"HasSuffix": true,
		"EqualFold": true,
	}
	regexpMatchers = map[string]bool{"MatchString": true}
	testifyText    = map[string]bool{
		"ErrorContains":  true,
		"ErrorContainsf": true,
		"Regexp":         true,
		"Regexpf":        true,
	}
	testifyEqualError = map[string]bool{"EqualError": true, "EqualErrorf": true}
	testifyEqual      = map[string]bool{"Equal": true, "Equalf": true}
)

// errorType is the predeclared error interface. The universe holds one
// object for it, and every use of the name error names that object.
var errorType = types.Universe.Lookup("error").Type()

// Analyzer is the G13 analyzer. The rule is opt-in, so the
// golangci-lint plugin runs it only when the enable setting names it.
// Consumers get it through antislop.Analyzers(), and cmd/antislop
// registers its flags.
var Analyzer = New(false, nil)

// New returns an analyzer that reads the equality setting, and that
// reads the packages the testPackages patterns name as test code. A
// programmatic consumer, such as the golangci-lint plugin, builds one
// instance for each configuration. Two runs can hold different
// settings, and the package-level value is shared.
//
// The instance carries its own flags, which write into the
// configuration of that instance. The flags are the configuration
// surface of cmd/antislop and of go vet -vettool, which read no
// settings file.
func New(equality bool, testPackages []string) *analysis.Analyzer {
	cfg := &config{equality: equality, testPackages: testPackages}
	a := &analysis.Analyzer{
		Name:     "errsemantics",
		Doc:      doc,
		URL:      "https://github.com/JacobJNilsson/anti-slop-go/blob/main/docs/spec/002-rules.md",
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Run:      cfg.run,
	}
	a.Flags.BoolVar(&cfg.equality, "equality", equality,
		"report a comparison of an error message as well; a package that tests its own message text leaves it off")
	a.Flags.Var(&cfg.testPackages, "testpackages",
		"package path patterns whose files count as test files, separated by commas; a repeated flag adds patterns")

	return a
}

// config holds the settings of one analyzer instance.
type config struct {
	equality     bool
	testPackages pathmatch.List
}

// form names the group of one reported call. The groups differ in the
// evidence they need and in the setting that turns them on.
type form int

const (
	// formNone is a call that the rule never reports.
	formNone form = iota
	// formText needs an Error call in an argument, and reports by
	// default.
	formText
	// formErrorText needs an Error call or an error value in an
	// argument, and reports by default. The text asserts of testify take
	// the error itself, so no Error call appears there.
	formErrorText
	// formEqualityValue needs an Error call or an error value, and the
	// equality setting turns it on.
	formEqualityValue
	// formEqualityText needs an Error call, and the equality setting
	// turns it on. Equal of testify takes two values of any type, and an
	// assertion on the error value itself is no text assertion.
	formEqualityText
)

// CONTRACT: analysis.Analyzer.Run fixes this signature.
func (c *config) run(pass *analysis.Pass) (any, error) {
	// SAFETY: inspect.Analyzer is in Requires, so the driver always
	// supplies its result, and that result is an *inspector.Inspector.
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	generated := signature.GeneratedFiles(pass)
	testFile := signature.TestFiles(pass, c.testPackages)
	// A test file holds the assertions of a project. Production code
	// that reads a message renders it, and it decides no test.
	reported := func(pos token.Pos) bool {
		return testFile(pos) && !generated(pos)
	}

	insp.Preorder([]ast.Node{(*ast.CallExpr)(nil), (*ast.BinaryExpr)(nil)}, func(n ast.Node) {
		if !reported(n.Pos()) {
			return
		}
		switch node := n.(type) {
		case *ast.CallExpr:
			c.checkCall(pass, node)
		case *ast.BinaryExpr:
			c.checkComparison(pass, node)
		}
	})

	return nil, nil
}

// checkCall reports one call of a reported group.
//
// The report sits at the call and never at the Error call inside it.
// The call is the predicate that decides the test, and it is the line
// the author rewrites.
func (c *config) checkCall(pass *analysis.Pass, call *ast.CallExpr) {
	fn := callee(pass, call)
	form := formOf(fn)
	args := assertArgs(fn, form, call.Args)
	switch form {
	case formText:
		if hasErrorText(pass, args) {
			pass.Reportf(call.Pos(), "%s", textMessage)
		}
	case formErrorText:
		if hasErrorText(pass, args) || hasErrorValue(pass, args) {
			pass.Reportf(call.Pos(), "%s", textMessage)
		}
	case formEqualityValue:
		if c.equality && (hasErrorText(pass, args) || hasErrorValue(pass, args)) {
			pass.Reportf(call.Pos(), "%s", equalityMessage)
		}
	case formEqualityText:
		if c.equality && hasErrorText(pass, args) {
			pass.Reportf(call.Pos(), "%s", equalityMessage)
		}
	case formNone:
	}
}

// assertArgs returns the arguments that carry the assertion.
//
// Every testify function the rule reads takes the testing value, then
// the two values it asserts on. The receiver form of Assertions drops
// the testing value. Everything after those is the failure message:
// the variadic tail, and the format string of a name that ends in "f".
// An error there is diagnostic output, and it decides nothing, so a
// report on it would name a correct line.
//
// The predicates of strings and regexp take no message, so every
// argument of such a call carries the assertion.
func assertArgs(fn *types.Func, form form, args []ast.Expr) []ast.Expr {
	if form == formNone || form == formText {
		return args
	}
	positions := testifyAssertArgs
	if fn.Signature().Recv() != nil {
		positions--
	}

	return args[:min(positions, len(args))]
}

// checkComparison reports a comparison of an error message against a
// string. The report sits at the operator, which is the assertion.
func (c *config) checkComparison(pass *analysis.Pass, cmp *ast.BinaryExpr) {
	if !c.equality || (cmp.Op != token.EQL && cmp.Op != token.NEQ) {
		return
	}
	if isErrorText(pass, cmp.X) || isErrorText(pass, cmp.Y) {
		pass.Reportf(cmp.OpPos, "%s", equalityMessage)
	}
}

// callee returns the function that a call names, and nil for a call
// that names none. A call through a function value, a conversion, and a
// call of a builtin all give nil, and the rule reads none of them.
func callee(pass *analysis.Pass, call *ast.CallExpr) *types.Func {
	var name *ast.Ident
	switch fun := ast.Unparen(call.Fun).(type) {
	case *ast.SelectorExpr:
		name = fun.Sel
	case *ast.Ident:
		name = fun
	default:
		return nil
	}
	fn, _ := pass.TypesInfo.Uses[name].(*types.Func)

	return fn
}

// formOf returns the group of one called function.
//
// The test reads the package of the object and never the name of the
// import. A renamed import and a dot import therefore follow the same
// rule. A local package named strings declares another object, and a
// library that carries the names of testify under another path does
// too. Neither one gives a group.
func formOf(fn *types.Func) form {
	if fn == nil || fn.Pkg() == nil {
		return formNone
	}
	path, name := fn.Pkg().Path(), fn.Name()
	switch {
	case path == stringsPath && stringsPredicates[name]:
		return formText
	case path == regexpPath && regexpMatchers[name]:
		return formText
	case !strings.HasPrefix(path, testifyPrefix):
		return formNone
	case testifyText[name]:
		return formErrorText
	case testifyEqualError[name]:
		return formEqualityValue
	case testifyEqual[name]:
		return formEqualityText
	default:
		return formNone
	}
}

// hasErrorText reports whether one argument of a call is an Error call
// on an error value.
func hasErrorText(pass *analysis.Pass, args []ast.Expr) bool {
	for _, arg := range args {
		if isErrorText(pass, arg) {
			return true
		}
	}

	return false
}

// hasErrorValue reports whether one argument of a call carries an
// error value. The text asserts of testify take the error itself and
// call Error inside, so the argument is the whole evidence there.
func hasErrorValue(pass *analysis.Pass, args []ast.Expr) bool {
	for _, arg := range args {
		if isError(pass.TypesInfo.TypeOf(ast.Unparen(arg))) {
			return true
		}
	}

	return false
}

// isErrorText reports whether an expression is an Error call on an
// error value. Such a call returns the message, and the message is the
// prose that the rule rejects as evidence.
//
// The test of the method name is a filter and no decision. The
// predeclared error interface declares one method, so a method call on
// a value of that type is an Error call and nothing else. The name
// stands here because it states the shape to a reader, and because it
// answers before the type lookup.
func isErrorText(pass *analysis.Pass, expr ast.Expr) bool {
	call, isCall := ast.Unparen(expr).(*ast.CallExpr)
	if !isCall || len(call.Args) != 0 {
		return false
	}
	selector, isSelector := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	if !isSelector || selector.Sel.Name != "Error" {
		return false
	}

	return isError(pass.TypesInfo.TypeOf(selector.X))
}

// isError reports whether a type is the predeclared error type.
//
// The test is deliberately narrow, and rule G10 states the same one. An
// interface that embeds error, and an interface that declares
// Error() string under another name, both hold the method set of error.
// A method set is no promise about a wrap chain, so errors.Is and
// errors.As answer no question about such a value.
//
// types.Identical unaliases its arguments, so "type E = error" is the
// same type here, and the fixture pins that.
func isError(t types.Type) bool {
	return types.Identical(t, errorType)
}
