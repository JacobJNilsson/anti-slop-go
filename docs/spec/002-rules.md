# 002: Rule catalogue

Each rule has an identifier (G01-G11), an analyzer name, a default
severity, and a statement of what it rejects and why. Examples show a
rejected form and an accepted form.

Severity levels:

- **error**: on by default.
- **opt-in**: off by default; the project must enable it.

G03 and G04 share one exemption. The section "The external contract
exemption", after G11, describes it, and both rules point at it.

## G01 `safetyassert`: require a SAFETY comment for panicking type assertions (error)

A single-result type assertion `v.(T)` panics when the assertion fails.
The author must state the invariant that makes the panic unreachable.
The comma-ok form is checked code and needs no comment.

Rejected:

```go
cfg := raw.(Config)
```

Accepted:

```go
// SAFETY: raw comes from configStore.Load, which only stores Config values.
cfg := raw.(Config)
```

Also accepted:

```go
cfg, ok := raw.(Config)
if !ok {
    return fmt.Errorf("config store returned %T", raw)
}
```

The comment must sit directly above the assertion or above its
containing statement. The marker `SAFETY:` must start a line of the
comment text, so `NOT-SAFETY:` is no justification. The full marker
contract is in 003.
This rule supersedes `forcetypeassert`.

The rule skips generated files, which `go/ast` recognises by the
`Code generated ... DO NOT EDIT.` header. Nobody edits a generated file
to add a justification, so a diagnostic there is noise the project
cannot fix.

## G02 `nountypedmap`: no untyped maps in signatures and fields (error)

`map[string]any`, `map[string]interface{}`, and maps with `any` values
describe nothing. Data with known keys belongs in a struct. Decode at
the I/O boundary with `json.Unmarshal` into a struct, not into a map.

The rule flags untyped maps in:

- exported and unexported function parameters and results,
- struct fields,
- package-level variable declarations.

The rule never flags a local variable. A decode function keeps its
untyped map inside the function body, and the configuration needs no
allowlist for it.

The rule skips generated files, which `go/ast` recognises by the
`Code generated ... DO NOT EDIT.` header. A program writes the file, so
a diagnostic there has no reader who can act on it.

Rejected:

```go
func Render(data map[string]any) ([]byte, error)
```

Accepted:

```go
type RenderInput struct {
    Title string
    Rows  []Row
}

func Render(data RenderInput) ([]byte, error)
```

## G03 `noanyparam`: no `any` parameters (error)

An `any` parameter moves parsing from the boundary into the callee.
Accept a named domain type. Generic type parameters with constraints
are fine; they carry evidence.

The rule flags a parameter of a function declaration, a method, a
function literal, an interface method, and a declared function type.
An alias, such as `type A = any`, is the same type, so the rule flags
its use sites. A defined type, such as `type Payload any`, is a domain
type, so the rule accepts it. The rule tests the written parameter type
only. It accepts a type that merely holds the empty interface, such as
`[]any`, `*any`, or `Box[any]`. A type parameter is not the empty
interface, so `[T any]` and its uses stay clean. 003 keeps the open
question about an unconstrained type parameter.

The rule accepts three shapes.

**The `fmt`-style variadic tail.** The rule accepts the last parameter
when every one of these holds:

- the parameter is variadic, and its element is the empty interface;
- an earlier parameter has the predeclared type `string`, which a
  defined string type is not;
- that parameter's name starts or ends with "format", in any case, or
  the signature has a name of more than one letter that ends in `f`.

The name is the name of the function, of the method, of the interface
method, of the function field, or of the declared function type. A
function literal carries no name, so it needs the parameter name. A
name that only holds "format" inside itself, such as
`informationText`, is another word. The exemption covers the variadic
tail: an `any` parameter beside a format string keeps its report.

**The name `cause`.** The rule accepts a parameter named `cause`,
which mirrors the upstream error-wrapping helper. The analyzer cannot
read intent, so the name is the whole contract. A group such as
`(cause, value any)` keeps the report, because only one of its names
carries the contract.

**An external contract**, which the section "The external contract
exemption" describes. `context.Context`, with `Value(key any) any`, is
the case that needs the exemption of G03 and the exemption of G04 on
one method.

Rejected:

```go
func Store(key string, value any) error
```

Accepted:

```go
func Store[V Storable](key string, value V) error
```

The rule skips generated files, which `go/ast` recognises by the
`Code generated ... DO NOT EDIT.` header. A program writes the file, so
a diagnostic there has no reader who can act on it.

## G04 `noanyreturn`: no `any` returns (error)

An `any` return forces every caller to assert. Return the concrete
type, or a small interface the caller consumes, or a generic result.

The rule flags a result of a function declaration, a method, a function
literal, an interface method, and a declared function type. Aliases,
defined types, type parameters, and types that merely hold the empty
interface follow G03 exactly.

The only exemption is an external contract, which the section "The
external contract exemption" describes. G04 needs no `cause` exemption
and no `fmt` exemption: neither shape returns a value.

Rejected:

```go
func Lookup(key string) (any, error)
```

Accepted:

```go
func Lookup(key string) (Record, error)
```

The rule skips generated files, which `go/ast` recognises by the
`Code generated ... DO NOT EDIT.` header. A program writes the file, so
a diagnostic there has no reader who can act on it.

## G05 `nolaundering`: no widen-then-assert (error)

A value must not pass through `any`, or through a broader interface,
and come back through an assertion. The widening throws away evidence
that the program had. The assertion makes a guess of that evidence
again. The rule reports two shapes.

**The chained shape.** The widening sits in the operand of the
assertion.

Rejected:

```go
u := any(user).(User)
f := reader.(any).(*os.File)
```

The rule reports the assertion when the operand widens a type that the
code knows. Such an operand is an explicit conversion to an interface,
or an assertion to an interface that the value satisfies already.
`any(v).(T)` is a report when `v` is a `T`, because the two steps
cancel. It is a report when `v` is another type too, because the
conversion manufactures the assertability: `v.(T)` alone does not
compile.

Accepted:

```go
rw := reader.(io.ReadWriter)
```

An assertion that only narrows an interface has no widening step. G05
accepts it, and G01 owns the justification.

**The binding shape.** The widening puts the value in a local
variable, and an assertion takes the type back later in the same
function.

Rejected:

```go
var v any = user
u := v.(User)
```

Accepted: keep `user` typed as `User`.

The rule tracks a local variable only when every assignment in the
function widens the same concrete type. A variable that also takes a
value which the function cannot know holds a real question, and the
rule accepts the assertion on it. Such a value has one of these
sources:

- a parameter,
- a call that returns `any`,
- a map read,
- a channel receive,
- a range clause,
- one call that fills two names.

A narrower interface, a type parameter, and a second concrete type
also stop the report. None of them names the type that the variable
holds, so the assertion still separates possibilities. This
conservatism is the guard against false positives.

The zero value of a declaration is not an assignment. A `var v any`
with one widening assignment in one branch stays a report. An
assertion on nil fails, so the other branch justifies nothing. The
advice of the message does not fit that shape. Where a widening
carries a value out of one branch, hold the value in a typed variable
beside a bool, or in a typed pointer.

The rule reports the comma-ok form as well. The widening answered the
question that it asks, so the `ok` result carries no information.

**What the rule leaves alone.** The rule reports an assertion that
takes a concrete type back. It reports a type switch too, whose cases
take concrete types back. An assertion to another interface asks
whether
the value satisfies that interface. Go states no negative answer to
that question at compile time, and the standard library tests use the
runtime form, so the rule leaves it. A version probe is the same
shape: code asserts a value against a small interface to learn whether
the build supplies a method. The compiler answers for the version in
front of it, and that answer does not settle the question that the
code asks.

A type parameter also stops a report. The rule leaves a widening
alone when a type parameter appears in the type that the widening
hides. It leaves the widening alone when a type parameter appears in
the type that the assertion names too. Go reads a value of a
parameterized type through an interface only, which `any(v).(T)`
shows. Go builds one the same way, which `any(concrete).([]T)` shows.
The test reads a type parameter in these places:

- the type arguments of a named type,
- the key and the value of a map,
- the fields of a struct type,
- the parameters and the results of a function type,
- the element of a pointer, a slice, an array, and a channel.

A generic function that launders a value of another type still gets a
report. 003 keeps the wider open question about type parameters.

`internal/signature` holds this walk, and rule G06 reads it for the
same shape in a type switch. One expression therefore gets one answer
from both rules.

These exemptions come from a scan of the whole standard library, tests
included. The scan reported 34 findings before them and 3 after, and
every one of the 3 is a widening that an assertion takes back.
`golang.org/x/tools`, `golang.org/x/exp`, and `golang.org/x/text`
report none. `golang.org/x/net` reports 2, in a vendored copy of the
`encoding/xml` test that std also reports.

A `SAFETY:` comment does not stop a report. G01 accepts the comment,
and G05 still reports, because the fix is to delete the widening and
not to justify the assertion. G05 reads no marker comment.

The message names the type that the widening hides and the line of the
widening. A variable with two widenings takes the first one. The line
is the adjusted line, so a `//line` directive moves the message and
the position of the diagnostic together.

The analyzer reads one function at a time, and it reads no order of
statements. It follows no value through a parameter, a result, a
struct field, a slice, a map, a channel, or a package-level variable. A
function literal that reads the variable drops the binding, because the
analyzer cannot see when the program runs the literal. A variable whose
address escapes drops too. 003 records what an SSA pass would add.

The rule skips generated files, which `go/ast` recognises by the
`Code generated ... DO NOT EDIT.` header. A program writes the file, so
a diagnostic there has no reader who can act on it.

## G06 `noadhoctypeswitch`: no ad hoc type switches on `any` (error)

A type switch over an `any` value re-parses data away from its
boundary. The program knew the type at an earlier line, and the switch
asks for it again. Every new case adds a shape that the reader of the
function must hold. Branch on a domain value instead: a kind field, a
sealed interface with a marker method, or one handler for each type.

Rejected:

```go
func handle(msg any) {
    switch m := msg.(type) {
    case Ping:
        ...
    case Data:
        ...
    }
}
```

Accepted:

```go
type Message interface{ isMessage() }

func handle(msg Message) { ... }
```

### What the rule reads

The rule reads the guard of a type switch and the static type of its
operand. It reports `switch x.(type)` and `switch v := x.(type)` where
that type is the empty interface. An alias, such as `type A = any`, is
the same type, so the rule reports its use sites. A defined type, such
as `type Payload any`, is a domain type of the package, so the rule
accepts it. G03 and G04 read a signature with the same test, and one
implementation serves the three rules.

The diagnostic sits at the `.(` token of the guard. A switch holds one
guard, so a switch with six cases gets one diagnostic, never six. A
switch with an init statement holds the same guard. A switch inside
another switch is a switch of its own, and it gets its own diagnostic.

Every operand of the empty interface type gets a report. The list holds
a parameter, a local variable, and a conversion such as `any(v)`. It
holds a call that returns `any`, a field of a struct, and a named
result. A local variable that a widening filled gets a report from G05
as well, and the fix there is to delete the widening. Where the value is dynamic in truth, the fix of
G06 is the domain value.

**The rule reads no single assertion.** `v.(T)` is another shape. G01
asks for the `SAFETY:` justification of it, and G05 reads the widening
in front of it. G06 reads the guard of a switch, which names no target
type, and nothing else.

**The rule reads no error.** The predeclared `error` type is an
interface with a method. It is therefore no empty interface, and no
error operand reaches this rule. G10 owns that shape, and the fix there
is `errors.As`. The division is structural, and no setting moves a
switch from one rule to the other.

**The rule reads no type parameter.** Go reads a value of a
parameterized type through an interface only. `switch any(v).(type)` is
therefore the one way to branch on the instantiation, and the widening
is no choice of the author. The rule accepts the switch where the type
of `v` holds a type parameter. G05 accepts the same widening in an
assertion, and `internal/signature` holds the one walk that both rules
read. The test reads a conversion and no call: a function that returns
`any` hides the type behind a signature, and the author wrote that
signature.

### The boundary packages

A program reads a dynamic type somewhere: at the boundary where bytes
become values. The `boundary-packages` setting of the plugin, and the
`-noadhoctypeswitch.boundary` flag of the standalone binary, take
package path patterns. Every type switch of a package whose import path
matches one pattern is clean. 003 states the pattern syntax and the two
configuration paths.

The rule drops a trailing `_test` from the import path before it reads
the patterns. One entry therefore covers a package and its external
test package. The drop works in one direction only, and it is silent.
G07 states the two consequences of that rule, and both hold here word
for word. An entry that names an external test package matches nothing.
An entry that names a production package also allows a real package
whose own path ends in `_test`.

The design puts the decoding in a boundary package, and the pattern
names that package:

```yaml
boundary-packages:
  - "example.com/app/internal/ingest"
```

**What the entry buys, and what it does not.** The entry silences G06
in that package, and nothing else. The decode function of such a
package usually takes an `any` parameter or returns one, and G03 and
G04 still report those signatures. `CONTRACT:` does not fit there. No
external API sets the signature of a decoder that the project itself
wrote, and the comment must name such an API.

A project therefore answers G03 and G04 in the boundary package in one
of two ways today. It names `noanyparam` or `noanyreturn` in the
`disable` setting, which drops the rule everywhere. Or it accepts the
findings of that one package, where the `any` is the point of the
design. Neither answer is a mechanism of this rule set, and this
specification invents none. A per-package toggle is a golangci-lint
question, and 003 records the `//nolint:antislop` directive that path
already carries.

### The evidence of an external contract

An external API can force an `any` parameter on a function of this
project. The section "The external contract exemption" states the two
kinds of evidence that G03 accepts for such a parameter. The first is
an interface of an imported package that declares the parameter. The
second is a `CONTRACT:` comment above the declaration that names the
API.

G06 accepts the same evidence for the operand of a switch. The operand
must be a parameter, and the signature that declares it must be exempt
under G03 at that position. The switch is then the legitimate
consumption of that contract. `database/sql.Scanner`, with
`Scan(src any) error`, is the case that the interface evidence covers.
An implementation of it must accept the empty interface, and it must
read the dynamic type of the value to fill its destination.

A comment carries the evidence that no interface states. A call that
sets a signature, such as the registration of a handler, is invisible
to the analyzer:

```go
// CONTRACT: bus.Subscribe sets the signature of a handler.
func onEvent(payload any) {
    switch p := payload.(type) { // clean: the comment admits the parameter
    case Ping:
        ...
    }
}

func init() { bus.Subscribe(onEvent) }
```

Where the parameter carries no such evidence, G03 reports the signature
and G06 reports the switch. The two findings have one cause, and the
signature is the one to fix. A parameter with a domain type leaves the
switch nothing to ask.

The signature that declares the parameter answers the question. A
function literal therefore answers for its own parameter, whatever the
comment above the function around it. It keeps the answer of the
function around it for a parameter that it reads from there. A literal
stored in a contracted field takes the comment above the statement that
holds it, which is the placement rule of `CONTRACT:` in 003.

**The evidence reads the parameter itself, and no copy of it.** The
rule compares the operand against the variables of the signature, so a
new variable loses the exemption:

```go
// CONTRACT: bus.Subscribe sets the signature of a handler.
func onEvent(payload any) {
    w := payload
    switch w.(type) { // reported: w is a new variable
    ...
    }
    payload = normalize(payload)
    switch payload.(type) { // clean: the parameter keeps its contract
    ...
    }
}
```

The message misdirects there. It asks for a boundary entry or a domain
value, and a `CONTRACT:` comment already sits three lines above. The
author switches on the parameter itself, or names the package as a
boundary. A test that follows a copy needs the value graph that 003
describes for G05, and this rule builds none.

The two other exemptions of G03 are no evidence here. The name `cause`
and the `fmt`-style variadic tail describe the use of a parameter, and
neither one names an API that sets the signature. A switch on such a
parameter keeps its report.

### What the rule leaves alone

- **A generated file**, which `go/ast` recognises by the
  `Code generated ... DO NOT EDIT.` header. A program writes the file,
  so a diagnostic there has no reader who can act on it.
- **A package that a boundary pattern names**, in every file of it.
- **An operand that is no empty interface**: an error, a defined type,
  and every interface with a method.
- **A conversion that widens a type parameter**, which Go demands.

### Decisions the rule states

**No comment above the switch stops a report.** The justification
belongs to the signature that admitted the `any` value, where the
reader of the API stands. A `CONTRACT:` comment above the switch
statement itself justifies nothing, and a fixture pins that. A comment
there would also justify a switch on a local variable, which no
external API can set.

**A test file gets no exemption.** A test that reads the dynamic type
of an `any` value builds the same table of shapes that production code
builds. The fix is the same domain value.

**`recover()` gets a report, and neither escape reaches it.** `recover`
returns `any`, because the language sets that signature. The operand is
a call and no parameter, so no `CONTRACT:` comment covers it. The
package that recovers is no decode boundary either. The escape is
therefore the `disable` setting, or the fix itself. The program owns
the value it panicked with. A comma-ok assertion against the sentinel
type of the package answers the question, and it needs no case for
every shape. The
known bite is the shape of `go/types`, which recovers a `bailout` value
and re-panics on every other value. The scan below holds 7 such
switches, 5 of them in tests that read a panic message.

**Two shapes give one finding and no signature to fix.** G03 accepts a
parameter named `cause`, and it accepts the variadic tail of an
`fmt`-style helper. Neither name is evidence here, so a switch on such
a parameter reports while the signature stays clean. The reader then
gets one finding with no second one to guide the repair. The answers
are the answers of this rule: a domain value in the signature, or a
boundary entry for a package that decodes.

**The message names the problem and one direction.** 003 asked whether
the diagnostic should suggest the marker-method pattern or stay silent.
The message names the problem and one direction for the repair, and it
stays inside the length of the family. The three shapes of the fix sit
in this entry, which the `URL` field of the analyzer names. The message
names both spellings of the boundary setting, because the reader of a
diagnostic runs one of two tools.

### Measurement

A scan of the whole standard library, tests included, reports 110
findings in 71 files and 46 packages. The scan ran the standalone
binary over `./...` in `GOROOT/src` with Go 1.26.2 on darwin/arm64, and
with no boundary pattern.

79 findings sit in production files of 31 packages, and 31 sit in test
files. Three packages hold 26 of the production findings.
`database/sql` gives 11, in `convert.go`, which turns a value of a
driver into the destination of a `Scan` call. `crypto/tls` gives 10.
They read `hc.cipher`, an `any` field that holds one of two cipher
shapes, and they read `crypto.PublicKey`, which the standard library
declares as `any`. `net/http` gives 5, all in `transfer.go`, which
passes a request or a response through an `any` parameter.
`database/sql/driver`, `encoding/binary`, `encoding/json`,
`encoding/xml`, `encoding/asn1`, `fmt`, `log/slog`, `text/template`,
and `html/template` hold most of the rest.

Every one of those packages is a boundary entry. They convert bytes to
values, or values to bytes, and the type switch is their work. A
project that writes such a package names it in `boundary-packages`, and
the rule then reports the switches that leaked out of it. `log/slog`
shows the other fix in the same scan. `Value.Kind` switches on the
`any` field of the value to compute a `Kind`, and a `Kind` field would
answer without the switch.

Two exemptions hold findings back in that scan. The type-parameter
walk holds 3: a run without it gives 113, and the three are
`crypto/internal/fips140/tls12`, `runtime/map_benchmark_test.go`, and
`runtime/pprof/pprof_test.go`. Each one branches on the instantiation
of a constrained parameter. The evidence of an external contract holds
1: `fakeDriverString.ConvertValue` of `database/sql` implements
`driver.ValueConverter`, and that interface declares the parameter. The
standard library writes no `CONTRACT:` comment, so the comment side of
the evidence adds nothing to these numbers.

`golang.org/x/tools` reports 28 findings in 16 packages, 23 of them in
production files. The largest groups sit in the test harness
(`internal/packagestest`), in the inliner, and in the JSON-RPC
transport, which decodes messages.

The volume of this rule is therefore a boundary question and not a code
question, as with G07. A project writes one entry for each boundary
package, and the rule reports the ad hoc switches elsewhere.

## G07 `noreflect`: no reflection outside allowlisted packages (error)

`reflect` erases every static guarantee. Serialization libraries need
it; application code does not. The configuration allowlists packages
(by path pattern) that may import `reflect`. `reflect.DeepEqual` in
test files is exempt by default.

Rejected:

```go
package service

import "reflect"

func Store(v any) error {
    if reflect.TypeOf(v).Kind() != reflect.Struct {
        return errors.New("want a struct")
    }
    ...
}
```

Accepted:

```go
package service

func Store(r Record) error { ... }
```

### What the rule reads

The rule reads the import of `reflect`, because the import is the
decision. A package that imports `reflect` reflects, whatever the
number of call sites, and one report per file names the decision at the
place a reader can remove it. The diagnostic sits at the import
specification. A file may import one package twice, under two names,
and each specification is a decision, so each one gets a report. No
production file of the standard library holds that shape, and a fixture
of the rule pins it.

The rule reads the objects that the type checker resolved, and never
the name of the import. A renamed import, such as `r "reflect"`, and a
dot import both give the same objects, so both follow the same rule.

### The allowlist

The allowlist is the whole escape. The `reflect-allow` setting of the
plugin, and the `-noreflect.allow` flag of the standalone binary, take
package path patterns. A package whose import path matches one pattern
may import `reflect` in every file. 003 states the pattern syntax and
the two configuration paths.

The rule drops a trailing `_test` from the import path before it reads
the patterns. The external test package of a package carries the path
of that package with `_test` at the end, and one entry covers both.

The drop works in one direction only, and it is silent, so an entry
names the production package path. Two consequences follow:

- An entry that names an external test package, such as
  `example.com/app/codec_test`, matches nothing. The rule drops the
  suffix from the path of the package before the comparison, so no
  package path ever ends in `_test`. Such an entry reports no error and
  allows nothing.
- An entry that names a production package also allows a real package
  whose own path ends in `_test`. A directory named `foo_test` holds
  such a package, and the pattern `example.com/app/foo` allows it.

Both come from one rule: the analyzer cannot ask the driver whether a
package is a test variant, so it reads the path.

The design puts the reflection in a boundary package, and the pattern
names that package:

```yaml
reflect-allow:
  - "example.com/app/internal/codec"
```

### The test file exemption

A test file that only calls `reflect.DeepEqual` is clean. Go gives a
test no other way to compare two values of a composite type, and the
comparison reads no type at run time that the test did not write.

The exemption covers `DeepEqual` alone. One other use of `reflect` in
the same file takes the whole import back into the rule, and the
message names that reason. The exemption is a property of the file, so
a package keeps clean test files beside a test file that reports.

The report of a test file sits at the first use that is not
`DeepEqual`, and not at the import. The import of a test file is no
error by itself, and the use is the line the author must change. A
production file keeps its report at the import, which is the line the
author removes there. A test file therefore gets one report, whatever
the number of its imports of `reflect`.

A blank import, `_ "reflect"`, uses no object of the package. A test
file with one stays clean under the rule above. A production file with
one reports at the import, because every import of a production file
reports and the file needs no reflection at all.

A file that is no test file always reports, `DeepEqual` included. A
production comparison of two values belongs to the types themselves,
through an `Equal` method or a comparison of fields.

### What the rule leaves alone

- **A generated file**, which `go/ast` recognises by the
  `Code generated ... DO NOT EDIT.` header. A program writes the file,
  so a diagnostic there has no reader who can act on it.
- **A package that a pattern allows**, in every file of it.

No comment stops a report. `SAFETY:` states an invariant and
`CONTRACT:` names an API that sets a signature, and neither fits an
import. The allowlist is the escape, and it sits in the configuration
of the project, where a review can read the whole list at once.

### Measurement

A scan of the whole standard library, tests included, reports 132
findings in 132 files and 52 packages. The scan ran the standalone
binary over `./...` in `GOROOT/src` with Go 1.26.2 on darwin/arm64.

48 findings sit in production files of 26 packages. Those 26 packages
hold the codecs (`encoding/gob`, `encoding/json`, `encoding/xml`,
`encoding/asn1`, `encoding/binary`), the formatters (`fmt`,
`internal/fmtsort`, `text/template`, `html/template`, `log/slog`), and
the frameworks that accept a value of any type (`database/sql`,
`net/rpc`, `testing`, `testing/quick`, `internal/fuzz`, `flag`). A
project allows such a package, or it never writes one.

84 findings sit in 84 test files, one for each file. 172 test files of
the standard library import `reflect`, so the `DeepEqual` exemption
clears 88 of them. That denominator comes from the file lists of the go
tool for the same platform:

```sh
go list -f '{{$d := .Dir}}{{range .TestGoFiles}}{{$d}}/{{.}}
{{end}}{{range .XTestGoFiles}}{{$d}}/{{.}}
{{end}}' std
```

The files of that list that hold an import line of `reflect` are the
172. A build constraint keeps some test files out of the list, and
another platform therefore gives another number. State the platform
with the count.

Four x repositories give: `golang.org/x/tools` 52 findings, 9 of them
in test files; `golang.org/x/text` 24 findings, 11 in test files;
`golang.org/x/net` 11 findings, 7 in test files; `golang.org/x/exp` 5
findings, 1 in a test file. The production findings of `x/tools` sit in
the syntax-tree machinery and in the fact encoding of the analysis
framework, which read types while they run. The findings of `x/text`
sit in the table generators and in the CLDR decoder, and the findings
of `x/net` sit in an XML codec and in `http2`.

The volume of this rule is therefore an allowlist question and not a
code question: a project writes one entry for each boundary package,
and the rule then reports the reflection that leaked out of them.

## G08 `nomonkeypatch`: no monkey patching in tests (error)

A test must not rewire production code through mutable globals.
The seam belongs in the design: accept an interface or a function
value as a dependency.

Rejected:

```go
// prod.go
var now = time.Now

// prod_test.go
now = func() time.Time { return fixed }
```

Accepted:

```go
type Clock interface{ Now() time.Time }

func NewServer(c Clock) *Server
```

### What the rule reads

The rule reads test files. A test file is a file whose name ends in
`_test.go`. The name of the file is the whole test. The name of the
package answers another question. The rule reads the external test
package `a_test` and the test files that sit in package `a` the same
way.

The rule reports three shapes in a test file.

**An assignment that rewires behaviour of a package-level variable.**
Behaviour is a function value or an interface. A variable of an
interface type holds an implementation, and an assignment gives the
package another one. A defined type, such as `type Hook func()`,
carries the behaviour of the type it names. A variable of another
type, such as an integer field, holds data and gets no report.

The target reaches the variable through a field, an index, or a
pointer. `Options.Now`, `Registry["boot"]`, `Chain[0]`, and
`*Fallback` are reports where the name at the root of the target is a
package-level variable. The root carries the ownership, and the target
carries the type.

The variable belongs to the package under test, or to any package that
the test imports. Both are reports. An assignment to a variable of
another package is worse, because the owner of that code cannot see
the change.

The rule reads the assignment statement, which is `*ast.AssignStmt` in
the syntax tree. Each target of a multiple assignment gets its own
report. The rule reads a range clause that assigns with `=` as well,
because such a clause writes to a variable that another statement
declares. A clause with `:=` declares its own variables, so it stays
clean.

**An import of a runtime patching library.** The rule reports the
import itself, because the import is the decision. The list of
libraries is fixed in the analyzer: `bou.ke/monkey` and
`github.com/agiledragon/gomonkey`. An entry matches the path itself
and every directory under it, so a version path such as
`github.com/agiledragon/gomonkey/v2` matches. A path that only starts
with the same letters is another library. The list takes no
configuration yet. A project that needs another entry files an issue,
and every project then gets the entry.

**A `//go:linkname` directive.** A directive carries no space between
the slashes and the name, and a space or a tab follows the name. The
rule reports the comment. A comment with another name, such as
`//go:linknamex`, is another directive.

### What the rule leaves alone

- **A variable that a test file declares.** The rule reads the file of
  the declaration, not the file of the assignment. Such a variable is
  test infrastructure, and the test files of the package own it, its
  entries included. Another test file of the same package may assign
  to it, and an external test package may assign to an exported one.
  Only a variable that no test file declares carries production
  behaviour.
- **An assignment in a file that is no test file.** Production code
  that rewires its own package-level variable is another question, and
  this rule does not answer it.
- **A field, an entry, or an element of a value that the test owns.**
  A local struct, a map that the test builds, and a slice that the
  test builds are all seams that the design offers. A test that fills
  one of them uses the design, which is the fix this rule asks for.
  The rule reads the root of the target, so a field of a
  package-level struct is production state and gets a report.
- **A local variable**, which includes a short declaration that hides
  a package-level name. The variable belongs to the test.
- **A target under a call**, such as `*addressOf(&now) = f`. The rule
  follows no call, so such a target reaches no variable it can name.
  A helper of the test that takes a pointer and assigns through it
  hides the patch from this rule in the same way.
- **A `//go:linkname` directive in a file that is no test file.** Such
  a directive is systems programming, such as a runtime shim or a cgo
  shim. The rule cannot judge it, and the blast radius of a wrong
  report there is large. The narrow rule reports the test files, where
  the directive is a patch.
- **Generated files**, which `go/ast` recognises by the
  `Code generated ... DO NOT EDIT.` header. A program writes the file,
  so a diagnostic there has no reader who can act on it.

### Decisions the rule states

The rule reads the target of an assignment and never the value. A
method value, a function literal, and a named function all rewire the
same seam.

A restore does not stop a report. A test that saves the old value and
puts it back with `t.Cleanup` still rewires the production code while
it runs. The design still holds no seam.

No comment justifies a report. `SAFETY:` states an invariant and
`CONTRACT:` names an API that sets a signature, and neither fits here.
The fix is an injected dependency: the author changes the design, and
the report goes away.

The rule reads a statement that assigns. It follows no pointer that a
call answers with, so `patch(&now)` stays clean. It reads no call of a
helper in another package that assigns for the test.

The rule reports a variable that its own package documents as an
override point, such as a test hook of a library. The rule cannot read
that intent. Where a report is wrong, 003 documents the two escapes:
`//nolint:antislop` on the golangci-lint path, and the `disable`
setting that drops the rule from the set. A test that sets
`flag.Usage` in `TestMain` is the known case, and the standard library
holds one of them. Neither escape belongs in the analyzer, and both
keep the build green while the project decides.

### Measurement

A scan of the whole standard library, tests included, reports 152
findings in 38 files. 139 findings are assignments to 43 variables,
and 85 of those name a variable with `hook` or `testingOnly` in its
name, such as `testHookLookupIP`. Such a variable is the shape this
rule rejects: the package keeps the seam in a global, where a
parameter or a field belongs. 13 findings are `//go:linkname`
directives in test files of `runtime`, `runtime/metrics`,
`runtime/pprof`, `sync`, `time`, and `unique`. No package of the
standard library imports a runtime patching library.

The interface shape and the root of the target add 13 findings to that
scan. `crypto/rand` patches its own `Reader`, and
`testing/cryptotest` patches `rand.Reader` from another package, which
is the shape this rule calls worse. `net/http` fills `testHookMu`,
`internal/buildcfg` resets its `Error`, and `flag` writes
`CommandLine.Usage` through a package-level pointer. Every one of the
13 is the shape the rule states.

Two shapes in that scan show the cost of a rule that reads no intent.
A test that sets `flag.Usage` configures the test binary itself, and
it rewires no code under test. A test of `runtime` reaches the runtime
through the linker, and no injected dependency replaces that. Both
carry the shape that the rule states, so both get a report.

## G09 `nointerfacereturn`: return concrete types for known values (opt-in)

A function that always builds one concrete type must return that type,
not an interface. The caller keeps the evidence and the methods.
`error` returns are exempt. This overlaps with `ireturn`; enable one.

Rejected:

```go
func NewStore() Storage { return &fileStore{} }
```

Accepted:

```go
func NewStore() *FileStore { return &FileStore{} }
```

## G10 `noerrorassert`: no type assertions on errors (error)

A wrapped error defeats a direct assertion and a direct type switch.
The wrapper carries its own dynamic type, so the test misses the error
that the code looks for. `errors.As` and `errors.Is` walk the wrap
chain, so they answer the question the code asks.

Rejected:

```go
if pe, ok := err.(*fs.PathError); ok { ... }
```

Accepted:

```go
var pe *fs.PathError
if errors.As(err, &pe) { ... }
```

**What the rule reads.** The rule reads the static type of the operand.
It reports the operand whose type is the predeclared `error` type, and
an alias of that type, such as `type E = error`. It reports both forms
of the assertion, `err.(T)` and `pe, ok := err.(T)`. The comma-ok form
carries the same bug: the wrapper answers `false`.

**Only the `error` type.** The rule matches the literal predeclared
type. An interface that embeds `error` is another type, and the rule
leaves it alone. An interface that declares `Error() string` under
another name is another type too. Both hold the method set of `error`.
A method set is no promise about a wrap chain. Only the `error`
contract promises one, through `Unwrap` and the `%w` verb of
`fmt.Errorf`. A report on a wider interface would name code that
`errors.As` cannot always replace.

A defined rename escapes the rule as well. `type MyError error` has the
predeclared type as its underlying type, and the rule reports no
assertion on a `MyError` value. That shape is not the method-set case:
its values come from the same `%w` machinery, and `errors.As` accepts
them unchanged. A scan of the standard library measured the wider test
with `types.Implements` and gave the same 307 findings, so the narrow
test loses nothing there. The rule keeps the narrow test, and a project
that renames `error` gets no report.

**A type switch gets one diagnostic.** The guard of a type switch holds
one assertion, and the rule reports there. A switch with six cases gets
one diagnostic, never six. The rule reads `switch err.(type)`,
`switch e := err.(type)`, and the form with an init statement.

Rejected:

```go
switch e := err.(type) {
case *fs.PathError:
    ...
}
```

**An interface target is a report too.** `err.(interface{ Timeout() bool })`
is the idiom that the `net` package used before `errors.As` existed.
Wrapping defeats it the same way, so the rule reports it. `errors.As`
accepts a pointer to an interface variable, so the fix holds:

```go
var timeout interface{ Timeout() bool }
if errors.As(err, &timeout) && timeout.Timeout() { ... }
```

A type parameter target, such as `err.(T)` in a generic function, names
one type at each instantiation. The rule reports it with the message
for a concrete target.

**No comment stops a report.** G01 asks for a `SAFETY:` comment above a
single-result assertion. G10 still reports the same assertion, like
G05, because the fix is `errors.As` and not a justification. The rule
reads no marker comment. One shape has no `errors.As` form: code that
must read exactly one level of the chain and must not walk it. That
shape is very rare. Its author can call `errors.Is` or `errors.As` on
an unwrapped copy of the error, or disable the rule for the project.
The messages for the type switch and for the interface target name that
escape, because those two shapes hold most of the deliberate
single-level code.

**The rule holds no exemption list.** A function that walks the wrap
chain itself asserts on an error, such as a copy of `errors.Unwrap`.
The rule reports it. A list of exempt shapes would grow with every
guess, and each entry would hide real findings elsewhere. A package
that implements the chain itself disables the rule.

**Positions and files.** The diagnostic sits at the `.(` token of the
assertion or of the switch guard. The rule skips generated files, which
`go/ast` recognises by the `Code generated ... DO NOT EDIT.` header. A
program writes the file, so a diagnostic there has no reader who can
act on it.

**A test file gets no exemption.** Test code reads wrapped errors too.
The fix costs more there than in production code. A test that asserts
an exact dynamic type asks a narrower question than `errors.As` answers:
`errors.As` matches a wrapped value that the test was written to
reject. A mechanical rewrite of such a test weakens it. The comparator
of `archive/tar/writer_test.go` and the test of `errors.As` itself in
`errors/wrap_test.go` both hold that shape. The author must decide
which question the test asks, and rewrite it or disable the rule for
the package.

**The volume this rule adds.** A scan of the standard library gives 307
findings. 86 sit in production files, and 221 sit in test files. The
test ruling above therefore drives 72 percent of the volume. The
numbers stay in family with the rules that shipped before: `safetyassert`
gives 1220 findings on the same scan, and `noanyparam` gives 601. G10
ranks third. The default severity of `error` rests on these numbers,
and a project with a large legacy error surface uses the `disable`
setting.

`staticcheck` covers parts of this; the rule exists so the set stays
complete when a project runs this project alone. A project that runs
`staticcheck` with the equivalent checks names `noerrorassert` in the
`disable` setting of the plugin, which 003 describes.

## G11 `justifypanic`: require a PANICS comment for panic in library code (opt-in)

`panic` outside `main`, `init`, and test helpers is an API decision.
The author must state why the process cannot continue, in a `// PANICS:`
comment directly above the call. `log.Fatal` and `os.Exit` in library
code get the same treatment.

Rejected:

```go
func MustParse(s string) Config {
    c, err := Parse(s)
    if err != nil {
        panic(err)
    }
    return c
}
```

Accepted:

```go
func MustParse(s string) Config {
    c, err := Parse(s)
    if err != nil {
        // PANICS: MustParse is documented for package-level defaults only;
        // a bad literal is a programmer error, not runtime input.
        panic(err)
    }
    return c
}
```

## The external contract exemption, shared by G03 and G04

An external API can set the shape of a signature. The author cannot
change such a signature and keep the program working, so a report there
names code the project cannot repair. G03 and G04 accept two kinds of
evidence for it.

### Evidence the analyzer reads: an interface method

A method is exempt when all of this holds:

- the receiver, or the pointer to the receiver, implements an exported
  interface of a package that the analysed package imports;
- that interface declares a method of the same name;
- that method has the empty interface at the same position.

```go
func (c *store) Value(key any) any { ... } // implements context.Context
```

This evidence needs no comment, and it covers the common case. It has
two limits. The analysed package must import the package that declares
the interface. A structural implementer that names no such import gets
nothing here. And `types.Implements` answers false when a type
parameter of the receiver takes part in the match. A generic type that
implements the interface for one instantiation only then loses the
exemption. A generic receiver whose type parameters take no part keeps
it. Both limits lose the exemption; neither invents one. The comment
below is the remedy.

### Evidence the author writes: a CONTRACT comment

Every other signature-setting API needs a justification comment,
because the analyzer cannot see what sets the shape:

```go
// CONTRACT: sync.Pool.New sets this signature.
func newBuffer() any { return new(bytes.Buffer) }
```

The comment follows the justification contract of 003: it owns its
line, and the marker `CONTRACT:` starts a line of its text. It ends on
the line directly above the line where the signature starts. A function
declaration, a method, an interface method, and a field start on that
line. Where the signature sits inside an expression that spans lines,
the comment sits above the statement or the variable declaration.

One comment can silence more than one signature, so place it with
care:

- It covers every empty interface parameter and result on the line it
  sits above. A comment above `func F(cb func(v any))` covers `F` and
  the callback type.
- It covers every signature inside the statement or the variable
  declaration it sits above. A comment above a handler table covers
  every entry of that table.
- It covers nothing inside a `var ( ... )` block or a type
  declaration. The comment there sits above the block, not above an
  entry.

The text must name the API that sets the signature. The analyzer
cannot judge the text; review must.

A comment, not a table of shapes, is the escape for a reason. A call
that sets a signature can take any form: a struct field, a call
argument, a conversion, or a method value. A test of one form accepts
that form silently and elsewhere, and it hides the evidence from the
reader of the declaration. The comment sits where the reader stands, it
names the API, and a review can reject it.

This repository writes the comment at every entry point it owns. The
`Run` field of `analysis.Analyzer` has the type
`func(*analysis.Pass) (any, error)`. `register.NewPlugin` sets
`New(conf any)`. So the analyzers of this module and the golangci-lint
plugin justify themselves.

## Rules considered and rejected

- **Structural-name bans** (upstream `no-shape-in-symbol-names`):
  the upstream rule polices a TypeScript naming habit. A Go version
  would ban words like `Data` or `Info` and would fire on idiomatic
  code. Out of scope.
- **Ignored errors**: `errcheck` owns this. Duplication creates
  configuration drift.
- **Comment and documentation tone**: out of scope; see 001 non-goals.
