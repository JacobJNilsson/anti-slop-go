package signature

import (
	"go/ast"
	"go/token"
	"os"
	"regexp"

	"golang.org/x/tools/go/analysis"
)

// markerExpr returns the expression that matches one justification
// marker. The name is the marker word, such as "SAFETY". The expression
// follows the justification comment contract of
// docs/spec/003-implementation.md, and it is the only place that states
// the contract: every rule with a marker gets its expression here.
//
// The marker must start a line of the comment text, on any line of the
// group. A word boundary alone accepted "NOT-SAFETY:" and a marker in
// the middle of a sentence. A hyphen and a space are both word
// boundaries.
//
// The input is the text of ast.CommentGroup.Text. That method removes
// the comment markers and the first space of a line comment. It keeps
// the rest of the leading text of a line. A block comment may start
// with a space, and a gutter of stars stays, so the class before the
// name accepts both.
func markerExpr(name string) *regexp.Regexp {
	return regexp.MustCompile(`(?m)^[\s*]*` + regexp.QuoteMeta(name) + `\s*:`)
}

// Justifications answers, for one analysis pass, whether the author
// wrote a justification comment above a line. One pass and one marker
// give one instance. The markers of
// docs/spec/003-implementation.md share every test below; only the
// marker itself changes, so the rules cannot drift apart.
type Justifications struct {
	pass      *analysis.Pass
	marker    *regexp.Regexp
	generated func(pos token.Pos) bool
	// comments is nil until the first justification test. Most packages
	// never need it, and building it reads every source file.
	comments *commentIndex
}

// NewJustifications prepares the marker tests for one pass. The marker
// is the marker word of the rule, such as "SAFETY". The constructor
// builds the expression, so no rule can write its own.
func NewJustifications(pass *analysis.Pass, marker string) *Justifications {
	return &Justifications{pass: pass, marker: markerExpr(marker), generated: GeneratedFiles(pass)}
}

// Generated reports whether pos sits in a generated file. It runs the
// test of GeneratedFiles, so a rule with a marker needs one value only.
func (j *Justifications) Generated(pos token.Pos) bool {
	return j.generated(pos)
}

// MarkedAbove reports whether a justification comment ends on the line
// directly above one of lines, in the file that holds pos. The comment
// must own its line: a comment beside code justifies the code beside
// it, never the line below. The analyzer cannot judge the text of the
// comment; review must.
func (j *Justifications) MarkedAbove(pos token.Pos, lines []int) bool {
	if j.comments == nil {
		j.comments = newCommentIndex(j.pass)
	}
	return j.comments.markedAbove(j.pass.Fset.File(pos), lines, j.marker)
}

// LineOf returns the physical line of a position. It ignores //line
// directives, because comments sit at physical lines.
func LineOf(fset *token.FileSet, pos token.Pos) int {
	return fset.PositionFor(pos, false).Line
}

// EnclosingStmtLines returns the lines of the statements that hold the
// node at the top of stack: the innermost statement, and the outermost
// statement below the enclosing block. A rule adds them to its
// candidate lines, so a justification above the statement covers the
// code inside it.
//
// The walk stops at a block. A comment above a block would justify
// every statement inside it, whatever the length of the body, and the
// reader of the flagged line would stand far from it.
//
// A case clause and a communication clause hold a statement list and no
// block, so the block test does not stop the walk there. The line of a
// clause sits above the first statement of the list only, and a later
// statement of the same clause needs its own comment.
//
// A node outside every statement, such as one in a package-level
// declaration, gives no line.
//
// This walk is the placement rule of the justification contract of
// docs/spec/003-implementation.md. One implementation serves every
// marker, so the rules cannot drift apart. A rule adds the candidate
// lines of its own shape, such as the line of a token that a
// multi-line operand pushes down.
func EnclosingStmtLines(fset *token.FileSet, stack []ast.Node) []int {
	var lines []int
	for _, stmt := range enclosingStmts(stack) {
		lines = append(lines, LineOf(fset, stmt.Pos()))
	}

	return lines
}

// enclosingStmts returns the innermost statement that holds the node
// and the outermost statement below the enclosing block.
func enclosingStmts(stack []ast.Node) []ast.Stmt {
	var inner, outer ast.Stmt
	for i := len(stack) - 1; i >= 0; i-- {
		if _, isBlock := stack[i].(*ast.BlockStmt); isBlock {
			break
		}
		stmt, isStmt := stack[i].(ast.Stmt)
		if !isStmt {
			continue
		}
		if body, isClause := clauseBody(stmt); isClause {
			if len(body) == 0 || body[0] != outer {
				break
			}
		}
		if inner == nil {
			inner = stmt
		}
		outer = stmt
	}
	var stmts []ast.Stmt
	if inner != nil {
		stmts = append(stmts, inner)
	}
	if outer != nil && outer != inner {
		stmts = append(stmts, outer)
	}

	return stmts
}

// clauseBody returns the statement list of a case clause or of a
// communication clause, and reports whether the statement is one.
func clauseBody(stmt ast.Stmt) ([]ast.Stmt, bool) {
	switch clause := stmt.(type) {
	case *ast.CaseClause:
		return clause.Body, true
	case *ast.CommClause:
		return clause.Body, true
	}

	return nil, false
}

// commentIndex answers the question "does a whole-line comment end on
// this line of this file?" for every comment of the package. It holds
// no marker, so the same code serves every rule. Each pass builds its
// own index.
type commentIndex struct {
	byFile  map[*token.File]map[int][]*ast.CommentGroup
	ownLine map[*ast.CommentGroup]bool
}

func newCommentIndex(pass *analysis.Pass) *commentIndex {
	read := sourceReader(pass)
	index := &commentIndex{
		byFile:  make(map[*token.File]map[int][]*ast.CommentGroup, len(pass.Files)),
		ownLine: make(map[*ast.CommentGroup]bool),
	}
	for _, file := range pass.Files {
		tokenFile := pass.Fset.File(file.FileStart)
		src, err := read(tokenFile.Name())
		if err != nil {
			// Fail open: the own-line test needs the source bytes.
			src = nil
		}
		byLine := make(map[int][]*ast.CommentGroup, len(file.Comments))
		for _, group := range file.Comments {
			end := LineOf(pass.Fset, group.End())
			byLine[end] = append(byLine[end], group)
			index.ownLine[group] = startsOwnLine(tokenFile, src, group)
		}
		index.byFile[tokenFile] = byLine
	}
	return index
}

// markedAbove reports whether a whole-line comment that carries the
// marker ends on the line directly above one of lines.
func (ci *commentIndex) markedAbove(file *token.File, lines []int, marker *regexp.Regexp) bool {
	byLine := ci.byFile[file]
	for _, line := range lines {
		for _, group := range byLine[line-1] {
			if ci.ownLine[group] && marker.MatchString(group.Text()) {
				return true
			}
		}
	}
	return false
}

// sourceReader returns the file reader of the pass. Not every driver
// sets one, so the operating system is the fallback.
func sourceReader(pass *analysis.Pass) func(name string) ([]byte, error) {
	if pass.ReadFile != nil {
		return pass.ReadFile
	}
	return os.ReadFile
}

// startsOwnLine reports whether only whitespace comes before the comment
// group on its first line. A comment that trails code justifies the code
// beside it, never the line below.
func startsOwnLine(file *token.File, src []byte, group *ast.CommentGroup) bool {
	start := file.Offset(group.Pos())
	if start > len(src) {
		return true // Fail open: the source is unreadable or stale.
	}
	lineStart := start
	for lineStart > 0 && src[lineStart-1] != '\n' {
		lineStart--
	}
	for _, b := range src[lineStart:start] {
		if b != ' ' && b != '\t' {
			return false
		}
	}
	return true
}
