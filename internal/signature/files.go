// This file holds the file tests that every rule applies. A rule reads
// the shape of hand-written code, and the two tests below answer which
// file it reads. One implementation cannot drift from itself.

package signature

import (
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// GeneratedFiles returns the generated-file test for one pass. The test
// answers true for a position in a file that carries the "Code
// generated ... DO NOT EDIT." header, which go/ast recognises.
//
// Every rule needs this test, and a rule without a marker takes it
// alone. A rule states the shape of hand-written code, so a report
// against a file that a program writes has no reader who can act on it.
//
// The set behind the test holds each token.File itself, not its name. A
// //line directive changes the name that token.Position reports, so a
// comparison of names can exempt the wrong file in both directions.
func GeneratedFiles(pass *analysis.Pass) func(pos token.Pos) bool {
	files := make(map[*token.File]bool)
	for _, file := range pass.Files {
		if ast.IsGenerated(file) {
			files[pass.Fset.File(file.FileStart)] = true
		}
	}
	return func(pos token.Pos) bool { return files[pass.Fset.File(pos)] }
}

// IsTestFile reports whether a file is a test file. The name of the
// file is the whole test, which is the test the go tool applies.
//
// The name of the package answers another question. It separates the
// external test package from the test files that sit in the package
// under test. A rule that reads test code reads both.
func IsTestFile(file *token.File) bool {
	return strings.HasSuffix(file.Name(), "_test.go")
}
