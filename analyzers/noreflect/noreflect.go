// Package noreflect implements rule G07 of the anti-slop rule set. The
// rule rejects an import of reflect in a package that the configuration
// does not allow. Reflection drops the types the compiler checked, and
// the program then carries the proof obligation to run time.
package noreflect

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"

	"github.com/JacobJNilsson/anti-slop-go/internal/pathmatch"
	"github.com/JacobJNilsson/anti-slop-go/internal/signature"
)

const doc = `reject an import of reflect outside the allowed packages

The reflect package reads and writes values without the types the
compiler checked. A serialization library needs it. Application code
does not: it decodes at its boundary and keeps concrete types inside.

The rule reports the import of reflect, because the import is the
decision. Every file of a package that no allow pattern names gets a
report, and the report sits at the import specification. A file that
imports reflect twice, under two names, gets one report for each
specification.

The allowlist is the only escape. The allow flag takes package path
patterns, and the golangci-lint plugin takes the same patterns in the
reflect-allow setting. A pattern matches the whole import path. "*"
matches inside one segment, and "..." matches any run of characters. A
pattern that ends in "/..." names the package above it too. The rule
drops a trailing "_test" from the path of a package, so one entry
covers a package and its external test package.

A test file that only calls reflect.DeepEqual is clean. Go gives a test
no other way to compare two values of a composite type. One other use
of reflect in the same file takes the import back into the rule. The
report sits at that use and not at the import, because the use is the
line the author must change. The rule reads the objects the file uses,
so a renamed import and a dot import follow the same test. A blank
import uses no object, so a test file with one stays clean.

A test file is a file whose name ends in "_test.go". Every file of a
package that the testpackages flag names is a test file as well. The golangci-lint plugin
takes the same patterns in the test-packages setting. A package that
serves tests and carries no "_test.go" name, such as a shared suite,
needs that entry.

No comment stops a report. The fix is a boundary package that holds the
reflection, and the allow pattern that names it. The rule skips
generated files.`

// The path of the package the rule reads, and the one function of it a
// test file may call.
const (
	reflectPath   = "reflect"
	deepEqualName = "DeepEqual"
)

// The two messages the rule emits. Each one names the fix that fits the
// file it reports.
const (
	importMessage = "this package imports reflect; reflect drops the types the compiler checked; " +
		"decode at the boundary into a concrete type, or allow this package with " + allowNames
	testMessage = "this test file uses reflect beyond DeepEqual; reflect drops the types the compiler checked; " +
		"compare concrete values, or allow this package with " + allowNames
)

// allowNames names both spellings of the allowlist. The reader of a
// diagnostic runs one of two tools, and each tool takes its own
// spelling, so a message with one name leaves the other reader without
// a working setting.
const allowNames = "reflect-allow (-noreflect.allow outside golangci-lint)"

// Analyzer is the G07 analyzer. Consumers get it through
// antislop.Analyzers(), and cmd/antislop registers its flags.
var Analyzer = New(nil, nil)

// New returns an analyzer that allows the packages the allow patterns
// name, and reads the packages the testPackages patterns name as test
// code. A programmatic consumer, such as the golangci-lint plugin,
// builds one instance for each configuration, because two runs can hold
// different patterns and the package-level value is shared.
//
// The instance carries its own flags, which write into the
// configuration of that instance. The flags are the configuration
// surface of cmd/antislop and of go vet -vettool, which read no
// settings file.
func New(allow, testPackages []string) *analysis.Analyzer {
	cfg := &config{allow: allow, testPackages: testPackages}
	a := &analysis.Analyzer{
		Name: "noreflect",
		Doc:  doc,
		URL:  "https://github.com/JacobJNilsson/anti-slop-go/blob/main/docs/spec/002-rules.md",
		Run:  cfg.run,
	}
	a.Flags.Var(&cfg.allow, "allow",
		"package path patterns that may import reflect, separated by commas; a repeated flag adds patterns")
	a.Flags.Var(&cfg.testPackages, "testpackages",
		"package path patterns whose files count as test files, separated by commas; a repeated flag adds patterns")

	return a
}

// config holds the settings of one analyzer instance.
type config struct {
	allow        pathmatch.List
	testPackages pathmatch.List
}

// CONTRACT: analysis.Analyzer.Run fixes this signature.
func (c *config) run(pass *analysis.Pass) (any, error) {
	if pathmatch.Any(c.allow, packagePath(pass)) {
		return nil, nil
	}

	generated := signature.GeneratedFiles(pass)
	testFile := signature.TestFiles(pass, c.testPackages)

	// tests holds the test files that import reflect. A production file
	// reports here, at each specification of the import.
	var tests []*ast.File
	for _, file := range pass.Files {
		if generated(file.FileStart) {
			continue
		}
		specs := reflectImports(file)
		if len(specs) == 0 {
			continue
		}
		if testFile(file.FileStart) {
			tests = append(tests, file)
			continue
		}
		for _, spec := range specs {
			pass.Reportf(spec.Pos(), "%s", importMessage)
		}
	}

	// The walk below reads every identifier of the package, and only a
	// test file that imports reflect needs the answer.
	if len(tests) == 0 {
		return nil, nil
	}
	beyond := firstUseBeyondDeepEqual(pass)
	for _, file := range tests {
		if pos, isBeyond := beyond[pass.Fset.File(file.FileStart)]; isBeyond {
			pass.Reportf(pos, "%s", testMessage)
		}
	}

	return nil, nil
}

// reflectImports returns the specifications that import reflect. A file
// may import one package twice, under two names, and each
// specification is a decision of its own.
func reflectImports(file *ast.File) []*ast.ImportSpec {
	var specs []*ast.ImportSpec
	for _, spec := range file.Imports {
		// The parser accepts a quoted string only, so the path always
		// unquotes. A broken literal gives the empty path, which is no
		// import of reflect.
		path, _ := strconv.Unquote(spec.Path.Value)
		if path == reflectPath {
			specs = append(specs, spec)
		}
	}

	return specs
}

// packagePath returns the import path the rule matches against the
// allow patterns.
//
// The external test package of a package carries the path of that
// package with "_test" at the end. Both hold the code of one package,
// and a project that allows the package allows its test code with it,
// so the rule drops the suffix.
func packagePath(pass *analysis.Pass) string {
	return strings.TrimSuffix(pass.Pkg.Path(), "_test")
}

// firstUseBeyondDeepEqual returns, for each file of the package, the
// position of the first use of an object of reflect that is not
// DeepEqual. A file with no such use holds no entry.
//
// The test reads the objects the type checker resolved, and never the
// name of the import. A renamed import, such as r "reflect", and a dot
// import both give the same objects, so both follow the same rule. A
// blank import gives no object at all.
//
// The qualifier of a selector, such as the name reflect in
// reflect.DeepEqual, resolves to a package name that the importing
// package declares. Its package is therefore the importing package, and
// this test skips it.
func firstUseBeyondDeepEqual(pass *analysis.Pass) map[*token.File]token.Pos {
	first := make(map[*token.File]token.Pos)
	for ident, object := range pass.TypesInfo.Uses {
		pkg := object.Pkg()
		if pkg == nil || pkg.Path() != reflectPath || object.Name() == deepEqualName {
			continue
		}
		// The map of a pass holds no order, so the earliest position of
		// the file wins. The diagnostic then sits at one line, whatever
		// the order the driver walked.
		file := pass.Fset.File(ident.Pos())
		if pos, found := first[file]; !found || ident.Pos() < pos {
			first[file] = ident.Pos()
		}
	}

	return first
}
