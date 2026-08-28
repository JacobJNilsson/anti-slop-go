// Package a holds the fixtures of rule G11 for library code. The
// package is no main package, and this file is no test file, so every
// call that stops the process needs a justification comment here.
package a

import (
	"log"
	"os"
	"runtime"
)

// Config is the domain value of the Must fixtures below.
type Config struct{ Name string }

func parse(s string) (Config, error) { return Config{Name: s}, nil }

func Bare() {
	panic("the store lost its connection") // want "panic in library code has no justification comment; state why the process cannot continue in a comment directly above it, or return an error to the caller"
}

func Justified() {
	// PANICS: the loader runs before the program serves a request.
	panic("the store lost its connection")
}

// A name is no evidence. "Must" tells the reader that the function
// panics, and the comment tells the reader why, so the rule reads the
// comment alone.
func MustParse(s string) Config {
	c, err := parse(s)
	if err != nil {
		panic(err) // want "panic in library code has no justification comment"
	}

	return c
}

func MustParseJustified(s string) Config {
	c, err := parse(s)
	if err != nil {
		// PANICS: MustParse takes a literal of the program, so a bad
		// value is a fault of the programmer and no runtime input.
		panic(err)
	}

	return c
}

// The body of an if is a block. A comment above the if would justify
// every call of that body, so the candidate lines stop at the block.
func MarkerAboveTheEnclosingIf(err error) {
	// PANICS: this comment sits above the if, not above the panic.
	if err != nil {
		panic(err) // want "panic in library code has no justification comment"
	}
}

// A case clause holds a statement list and no block, so the clause line
// is a candidate for the first statement of the list. A later statement
// of the same clause needs its own comment.
func MarkerAboveACaseClause(kind int) {
	switch kind {
	// PANICS: the caller reads the kind from a table of this package.
	case 1:
		panic("unknown kind")
	}
}

func MarkerAboveAMultiStatementCaseClause(kind int) {
	switch kind {
	// PANICS: the caller reads the kind from a table of this package.
	case 1:
		log.Println("the first statement of the clause is justified")
		panic("unknown kind") // want "panic in library code has no justification comment"
	}
}

// A method of the universe scope, such as Error of the predeclared
// error type, belongs to no package. The rule reads such a call and
// leaves it alone.
func ErrorText(err error) string { return err.Error() }

func Fatal() {
	log.Fatal("the disk holds no space") // want "log.Fatal in library code has no justification comment"
}

// A communication clause holds a statement list too, so the line of the
// clause is a candidate for the first statement of it.
func MarkerAboveACommClause(ch chan int) {
	select {
	// PANICS: the producer closes ch only when the table is broken.
	case <-ch:
		panic("the table is broken")
	}
}

func Fatalf(path string, err error) {
	log.Fatalf("cannot read %s: %v", path, err) // want "log.Fatalf in library code has no justification comment"
}

func Fatalln() {
	log.Fatalln("the disk holds no space") // want "log.Fatalln in library code has no justification comment"
}

func FatalJustified(path string, err error) {
	// PANICS: the daemon holds no state that the caller can repair.
	log.Fatalf("cannot read %s: %v", path, err)
}

// log.Panicf panics after it writes the line, so it is the shape of the
// first sentence of the rule.
func LogPanicf(err error) {
	log.Panicf("cannot continue: %v", err) // want "log.Panicf in library code has no justification comment"
}

func Exit(code int) {
	os.Exit(code) // want "os.Exit in library code has no justification comment"
}

// The report sits at the call and not at the statement. A deferred call
// starts six columns after its statement starts.
func DeferredExit(code int) {
	defer os.Exit(code) // want "os.Exit in library code has no justification comment"
}

func ExitJustified() {
	// PANICS: the signal handler already stopped every worker.
	os.Exit(1)
}

// A method of log.Logger calls os.Exit as the package function does,
// and the type checker resolves both to package log.
func LoggerMethod(l *log.Logger, err error) {
	l.Fatalf("cannot continue: %v", err) // want "log.Fatalf in library code has no justification comment"
}

// Service embeds the logger, so Go promotes the method and the object
// stays the one of package log.
type Service struct{ *log.Logger }

func (s Service) Stop() {
	s.Fatal("stop") // want "log.Fatal in library code has no justification comment"
}

// Reporter is a type of this project. Its Fatalf stops nothing, and the
// name alone is no evidence, so the rule reads no call of it.
type Reporter interface {
	Fatalf(format string, args ...any)
}

func CustomReporter(r Reporter) { r.Fatalf("the report continues") }

// The rule reads the object that the type checker resolved. A local
// value named os is no package, so its Exit field stops nothing.
func LocalShadow(code int) {
	os := struct{ Exit func(int) }{Exit: func(int) {}}
	os.Exit(code)
}

// The rule reads the call and follows no variable, so a function value
// that holds os.Exit stays clean.
var exit = os.Exit

func IndirectExit(code int) { exit(code) }

// runtime.Goexit ends one goroutine and no process, so the rule leaves
// it alone.
func Goexit() { runtime.Goexit() }

// A rethrow preserves the value that another function panicked with. It
// decides nothing, so it needs no justification.
func Rethrow() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("the caller gets the value back")
			panic(r)
		}
	}()
}

func RethrowDirect() {
	defer func() { panic(recover()) }()
}

func RethrowFromAVarSpec() {
	defer func() {
		var r = recover()
		if r != nil {
			panic(r)
		}
	}()
}

func RethrowFromAnAssignment() {
	defer func() {
		var r any
		count := 1
		r = recover()
		if r != nil && count == 1 {
			panic(r)
		}
	}()
}

// The exemption reads a variable. A recover call that fills a field
// names none, so the panic below keeps its report and its author writes
// the comment.
func RecoverIntoAField() {
	var state struct{ value any }
	defer func() {
		state.value = recover()
		if message, ok := state.value.(string); ok {
			panic(message) // want "panic in library code has no justification comment"
		}
	}()
}

// A recover call that drops its value fills no variable, so the panic
// beside it raises a value of this function.
func DiscardRecover() {
	defer func() {
		_ = recover()
		panic("the worker cannot continue") // want "panic in library code has no justification comment"
	}()
}

// A new value is a new decision, whatever the recover above it.
func PanicAfterRecover() {
	defer func() {
		if r := recover(); r != nil {
			panic("the wrapper loses the value") // want "panic in library code has no justification comment"
		}
	}()
}

// The exemption reads one function and no literal inside it. A variable
// that another function filled is no rethrow of this one.
func RethrowAcrossFunctions() {
	var r any
	defer func() { r = recover() }()
	panic(r) // want "panic in library code has no justification comment"
}

// A package-level value is no result of a recover call.
var sentinel = "the caller cannot repair this"

func PanicWithASentinel() {
	panic(sentinel) // want "panic in library code has no justification comment"
}

func PlainCommentAboveTheCall() {
	// The store cannot roll back the partial write above. The text
	// carries no marker word, and the rule accepts it.
	panic("the store lost its connection")
}

func MarkerOnTheSecondLineOfAGroup() {
	// The store cannot roll back the partial write above.
	// PANICS: the process must stop before it writes more.
	panic("the store lost its connection")
}

func MarkerInABlockComment() {
	/*
	 * PANICS: the gutter of stars is part of the contract.
	 */
	panic("the store lost its connection")
}

func MarkerBesideTheCode() {
	panic("x") // PANICS: a trailing comment is not above the call. // want "panic in library code has no justification comment"
}

func BlankLineBelowTheMarker() {
	// PANICS: a blank line breaks the link to the call.

	panic("the store lost its connection") // want "panic in library code has no justification comment"
}

// A comment above a go statement justifies the statement, and not the
// body of the literal.
func PanicInsideAFuncLiteral() {
	// PANICS: this comment justifies the go statement.
	go func() {
		panic("the worker cannot start") // want "panic in library code has no justification comment"
	}()
}

// A literal at package level sits in no function declaration, so the
// rule reads it as library code. The blank line keeps this comment from
// justifying the call.

var boom = func() { panic("the table holds no entry") } // want "panic in library code has no justification comment"

// An init function of any package runs before the program works, and
// 002 exempts it.
func init() {
	panic("the build carries no table")
}

type starter struct{}

// A method named init is no init function. The runtime calls no method,
// so this one is library code.
func (starter) init() {
	panic("the starter holds no table") // want "panic in library code has no justification comment"
}
