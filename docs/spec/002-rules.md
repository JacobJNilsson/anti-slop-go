# 002: Rule catalogue

Each rule has an identifier (G01-G13), an analyzer name, a default
severity, and a statement of what it rejects and why. Examples show a
rejected form and an accepted form.

Severity levels:

- **error**: on by default.
- **opt-in**: off by default; the project must enable it.

G03 and G04 share one exemption. The section "The external contract
exemption", after the last rule, describes it, and both rules point at
it.

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

The rule skips generated files. 003 states the header forms the module
accepts.

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

The rule skips generated files. 003 states the header forms the module
accepts.

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

## G03 `noanyparam`: no `any` parameters (opt-in)

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

The rule skips generated files. 003 states the header forms the module
accepts.

### Why the rule is opt-in

Go limits the reach of an `any` parameter. The callee cannot read such
a value until it asserts a type, and three rules of this set report
that assertion. G01 asks for a `SAFETY:` comment above a panicking
assertion. G05 reports a value that passes through `any` and comes back
through an assertion. G06 reports a type switch on an `any` value
outside a boundary package. A widened parameter therefore leaves
evidence at the use site, and the use site keeps its report.

The escape that the rule documents has a cost for the reader. A
defined type, such as `type Payload any`, satisfies the rule, and it
carries the same values as `any`. An adoption review read that escape
as ceremony. The declaration states a domain name, and no field and no
method stands under it. A project turns on such a rule deliberately,
because a reviewer rejects the escape it asks for.

The rule exempts no test file. The private service reports 43
findings, and at least 15 of them sit in test code. A test helper that
takes an `any` value gives the same report as an exported API. Those 15
findings carry no API decision, so a first run hides the exported
signatures behind them.

A project that leaves the rule off gets no report on a new API
signature. The `enable` setting restores the report, and 003 states the
setting. The opt-in severity belongs to the golangci-lint plugin, which
is the path that reads a configuration file. `cmd/antislop` and `go vet
-vettool` read none, so they run this rule by default, and
`-noanyparam=false` turns it off there. 003 records the split.

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

The rule skips generated files. 003 states the header forms the module
accepts.

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

The rule skips generated files. 003 states the header forms the module
accepts.

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

- **A generated file.** 003 states the header forms the module accepts.
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

A test file is a file whose name ends in `_test.go`. Every file of a
package that the `test-packages` setting names is a test file as well.
The section "The test-packages setting" states which packages count.

### What the rule leaves alone

- **A generated file.** 003 states the header forms the module accepts.
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
`_test.go`. Every file of a package that the `test-packages` setting
names is a test file as well. The section "The test-packages setting"
states which packages count. The name of the package answers another
question. The rule reads the external test package `a_test` and the
test files that sit in package `a` the same way.

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

- **A variable of a package that the `test-packages` setting names.**
  Such a package is test infrastructure, so the rule reads its
  package-level variables the way the next entry states.
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
- **Generated files.** 003 states the header forms the module accepts.

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
not an interface. The caller keeps the evidence and the methods that
come with the type. `error` results are exempt.

Rejected:

```go
func NewStore() Storage { return &FileStore{} }
```

Accepted:

```go
func NewStore() *FileStore { return &FileStore{} }
```

Also accepted, because two concrete types make the interface honest:

```go
func NewStore(mem bool) Storage {
    if mem {
        return &MemStore{}
    }

    return &FileStore{}
}
```

### What the rule reads

The rule reads the body, and it reports a result only when the body
proves the conclusion. It reads every return statement of the function
and takes the static type of the expression at that result position.
One return that gives no concrete type ends the proof, because the
author cannot narrow a result the body does not build. Four shapes give
no concrete type:

- A return of an expression whose type is already the interface. The
  value comes from somewhere else, such as a parameter or a call, and
  the evidence sits there.
- A naked return. It reads the result variable, whose type is the
  interface, and an assignment somewhere in the body set it. The rule
  reads no flow of values, so it stops there.
- A conversion, such as `return Storage(&fileStore{})`. The static type
  of a conversion is the type it names. The author states the widening,
  and the rule reads the statement.
- A body with no return statement at all, and a declaration with no
  body, such as an assembly stub.

Two shapes carry the proof. A return of a value builds the type of that
value. The expression can be a composite literal, a variable, or a call
with one result. A return of a call with several results carries the
type of each result in its tuple, so `return openFile(name)` proves the
first result.

A return of `nil` beside one concrete type keeps the conclusion. The
caller of such a function sees that one type or nothing, so the
concrete result describes the API. The constructor shape `return nil,
err` is therefore a report and not an exemption. The concrete result
also removes the trap of a nil pointer inside a non-nil interface,
which a caller of the interface form cannot see.

One shape in that group needs a reader who knows the package. A method
can return the interface to give the caller a nil interface value, and
`(*net.TCPAddr).opAddr` is such a method. The concrete result would
give a typed nil there, which is the trap above with the sign
reversed. Some of the findings with `or nil` in the message hold that
shape, and the author decides.

**Two of those shapes are silent escapes.** A conversion, and a named
result that an assignment fills before a naked return, both silence the
rule and leave no marker for review. Every other escape in this rule
set carries one: `CONTRACT:` states the API that sets the signature,
and `disable` sits in the configuration file. A project that enables
G09 buys that gap. The rule keeps both shapes, because it reads static
types, and because the author of a conversion states the widening on
purpose. Neither shape is the recommended escape. `CONTRACT:` is the
reviewable one, and it stays the answer for a signature the project
must keep.

### Which results the rule judges

The rule judges each result position on its own. A signature with an
interface at position one and a concrete type at position two gets one
report. The report sits at the type of the result it names.

Go groups names, so `(first, second Storage)` is one type and two
results. Every report of that group sits at the one type, so two
results that build the same type give one message. Two results that
build two types give two messages, because the reader needs both.

Three result types are out of scope:

- The predeclared `error`. Go returns errors through that interface,
  and a concrete error result breaks the caller that compares the
  value with `nil`. The test is narrow, as in G10: an interface that
  embeds `error` is another type, and the rule judges it.

  **A recorded risk.** A domain interface that embeds `error`, or that
  holds `Error() string` under another name, gets a report. The advice
  there re-creates the trap that the `error` exemption avoids. A
  function with a concrete error result gives a non-nil value for a nil
  pointer. The caller that tests the result then reads the wrong
  answer. A wider test, such as an exemption for every method set that
  holds `Error() string`, would remove that risk. The rule keeps the
  narrow test, because G10 states the same narrow test for the same
  reason. One project cannot get two answers about `error` from one
  rule set. A project that meets the shape uses `CONTRACT:` or
  `disable`.
- The empty interface. G04 reports `any` and `interface{}`, and it
  gives the same advice. A defined interface with no method, such as
  `type Empty interface{}`, is a domain type, and this rule judges it.
- A result type that mentions a type parameter. Go reads and builds
  such a value through the interface. Every other result of a generic
  function follows the rule, because a concrete instantiated type, such
  as `sorter[T]`, is a legal result.

The rule reads a function declaration and a method. Exported and
unexported declarations follow the same rule, and a test file gets no
exemption. A function literal takes its signature from the call or the
variable that holds it, so the rule leaves a literal alone. The returns
of a literal belong to the literal, and never to the function around
it.

### The exemptions

A method that an interface declares with the same result is exempt at
that result. The receiver must satisfy the whole interface. An
interface of an imported package therefore fixes the signature, and the
author cannot change it.

An interface of the package under analysis fixes the result too, for
another reason: the advice would stop the package from compiling. A
method that narrows its result no longer satisfies the interface, and
the compiler rejects the file. The rule cannot state advice that does
not build, so such a method is no finding. The scan reads an unexported
local interface too, because the compiler answers the same way for it.

This is the test that G03 and G04 share, and `internal/signature` holds
it. G09 takes a second entry point of the same machinery, and the two
stances differ on purpose. G03 and G06 report a parameter. The author
who owns the local interface can widen the parameter of that interface
too, so the local interface is no contract there. G09 reports a result,
and the same author has no such repair: the concrete type and the
interface disagree in one direction only. `internal/signature` gives
the two rules two constructors, and no rule reads the stance of the
other.

A method whose receiver implements no interface keeps its report. A
scan of the standard library measured the difference between the two
stances: the local interfaces hold back 101 findings of 378, and they
add none.

Every other external contract needs a justification: a `CONTRACT:`
comment directly above the declaration. Such a comment covers every
result of the signature. 003 states the comment contract. The analyzer
cannot judge the text; review must.

The rule skips generated files. 003 states the header forms the module
accepts.

### Why the rule is opt-in

The rule states a discipline, and the discipline costs something at two
points. A package that returns an unexported concrete type must export
that type to comply. A package that plans a second implementation holds
an interface that only one type satisfies today.

`context` shows both costs in one package. `Background`, `WithCancel`,
and `WithValue` each build one unexported type, and the interface is
the whole API of the package. A report there names no defect. A project
that wants the discipline elsewhere enables the rule and reads such a
package as the exception.

The opt-in severity belongs to the golangci-lint plugin, which is the
path that reads a configuration file. `cmd/antislop` and `go vet
-vettool` read none, so they run this rule by default, and
`-nointerfacereturn=false` turns it off there. 003 records the split.

`disable` and `enable` are the switches of the plugin, and
`//nolint:antislop` works on that path too. 003 states both.

### The difference from `ireturn`

`ireturn` reports an interface result by its type, and an allowlist
holds the types a project accepts. It asks no question about the body.
G09 reports a result only where the body proves that one concrete type
reaches every path. A function such as `func NewStore(mem bool) Storage`
with two implementations stays clean under G09, and it reports under
`ireturn`. G09 is the narrower rule, and it needs no allowlist. Enable
the one that fits the project; 001 records the overlap.

### Measurement

A scan of the whole standard library, tests included, reports 277
findings in 138 files and 75 packages. The scan ran the standalone
binary over `./...` in `GOROOT/src` with Go 1.26.2 on darwin/arm64, and
with `-nointerfacereturn` to select this rule alone. 38 findings sit in
test files. 64 of the 277 report a concrete type beside `nil`.

The largest groups are `crypto/hpke` with 21, `net/http` with 18, `net`
with 14, `image` with 12, and `text/template/parse` with 10. 162
findings name an unexported concrete type, and 115 name an exported
one.

The local-interface exemption holds 101 findings back, and it adds
none. A run without it gives 378. Four groups hold most of the
difference. `text/template/parse` gives 23 `Node` methods, and
`log/slog` gives 14 `slog.Handler` methods. `image` gives 14 methods
that build a `color.Color` or a `color.Model`. `net` gives 14 methods
above `Addr`, `Conn`, and its own `sockaddr`. Every one of them would
stop the package from compiling.

The exported half holds the findings a project can act on with no other
change. `io.LimitReader` returns `Reader` and always builds
`*io.LimitedReader`, which the package exports and documents.
`debug/elf`, `debug/macho`, `debug/pe`, and `archive/zip` return
`io.ReaderAt` or `io.ReadSeeker` and always build `*io.SectionReader`.
Each caller of those functions loses the methods of the concrete type
at the signature.

The unexported half holds the cost of the rule. `context` gives 7, and
`crypto/hpke` builds one `*aead` behind `AEAD`. The repair for such a
package is to export the type, which is a design decision and no lint
fix. The severity of the rule follows from that number: 162 findings of
277 ask for an API change.

`golang.org/x/tools` v0.38.0 reports 114 findings in 37 files and 20
packages, 4 of them in test files. `go/ssa/interp` holds 58, where an
interpreter passes values through an interface. `internal/mcp` and
`internal/jsonrpc2_v2` hold 19 between them.

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
assertion or of the switch guard. The rule skips generated files. 003
states the header forms the module accepts.

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

A `panic` outside `main`, outside `init`, and outside a test file is an
API decision. The author of a library stops the process of somebody
else. The author must state why the process cannot continue, in a
`// PANICS:` comment directly above the call. `log.Fatal` and `os.Exit`
in library code get the same treatment.

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

The severity is opt-in. The measurement below states the volume, and
the `enable` setting of the plugin turns the rule on, which 003
describes. The standalone binary and `go vet -vettool` read no
configuration file, so both apply every rule of the module. A reader of
those two paths turns this rule off with `-justifypanic=false`.

### What the rule reads

The rule reads the object that the type checker resolved, and never a
name. It reports three groups of calls:

- the builtin `panic`;
- `os.Exit`;
- `Fatal`, `Fatalf`, `Fatalln`, `Panic`, `Panicf`, and `Panicln` of
  package `log`.

Every `Fatal` of package `log` calls `os.Exit`, and every `Panic` of it
calls the builtin. The rule therefore reads the package function and
the method of `log.Logger` under one name. `log.Fatalf` and
`log.Default().Fatalf` are one decision, and a rule that reads the
package function alone would leave the other one open. An embedded
`*log.Logger` promotes the method, and the object stays the one of
package `log`, so a call through the embedding reports as well.

The `Panic` group belongs to the first sentence of the rule. Such a
call raises a panic, and the name of the package is no reason to read
it another way.

A logger of the project is another type. An interface that declares
`Fatalf`, and a value of a type that the project declares, both give
another object. The rule reads no call of them, because a name is no
evidence that the process stops.

The message names the function of package `log` in both forms, so a
call of `l.Fatalf` reports as `log.Fatalf`. The message names the
function that stops the process, and the position of the diagnostic
names the call itself.

`runtime.Goexit` gets no report. It ends one goroutine and leaves the
process running, so it is control flow and no termination.

The rule reads the call and follows no variable. A `var exit = os.Exit`
with a later `exit(1)` therefore stays clean. A local value that carries
the name of a package resolves to another object as well. A variable
named `os` with an `Exit` field therefore stops nothing.

The diagnostic sits at the call and not at the statement. A deferred
call starts later than the statement that holds it, and the call is the
line the author justifies.

### The justification

The comment follows the justification contract of 003. It owns its
line, and the marker `PANICS:` starts a line of its text, so
`NOT-PANICS:` is no justification. The comment ends on the line
directly above one of these lines:

- the line of the call;
- the line of the innermost statement that holds the call;
- the line of the outermost statement below the enclosing block.

One implementation gives that set to G01 and to G11, which
`internal/signature` holds, so the two rules cannot drift apart. G01
adds one candidate of its own, the line of the `.(` token. A call needs
no such line, because a call starts where it starts. The rule of a case
clause holds here word for word. The line of a clause sits above the
first statement of the clause only. A later statement of the same clause
needs its own comment.

The candidate lines stop at a block. A comment above an `if` therefore
justifies no panic inside the body of that `if`. Such a comment would
cover every statement of a body of any length, and the reader of the
panic would stand far from it. The accepted example above shows the
placement the rule asks for, which is the line directly above the call.

### What the rule leaves alone

- **The function `main` of a main package.** It is the program itself,
  and nobody stands behind it.
- **An init function of any package.** It runs before the program
  works, and it reports a broken build to the one who starts it.
- **Every test file.** A panic there stops one test binary, and the
  author of the test reads the stack trace at once. The exemption
  covers the whole file, so a helper of a test needs no comment. A test
  file is a file whose name ends in `_test.go`. Every file of a package
  that the `test-packages` setting names is a test file as well. The
  section "The test-packages setting" states which packages count.
- **A generated file.** 003 states the header forms the module accepts.
- **A rethrow of a recovered value**, which the next section states.

### Decisions the rule states

**The exemption reads the function and not the package.** A function
beside `main` in a main package is library code. The reader of that
function still stands behind the call, and the package name says
nothing about it. A function literal inside `main` is `main`, because
the declaration that holds the call decides. A call outside every
declaration sits in a package-level literal, which is library code.

**A method named `init` is no init function.** The runtime calls no
method, so such a method reports like any other one.

**A name is no evidence.** A `MustXxx` function gets no exemption from
its name. The name states that the function panics, and the rule asks
why. The comment is the whole evidence, and a review can reject it.

**A rethrow preserves and decides nothing.** The exemption covers
`panic(v)` where `v` is a `recover()` call, or a variable that a
`recover()` call of the same function filled. That shape re-raises the
panic of another function. A new value in a new variable keeps its
report, and `panic("wrapped")` beside a recover is such a value.

The rule reads one function, and it reads no order of statements inside
it. Three limits follow:

- A `recover()` in a nested function literal fills no variable of the
  function around it. A panic outside that literal reports.
- A `recover()` that fills a field or an entry names no variable the
  rule can follow. A panic on that field reports.
- A variable that a `recover()` filled keeps the exemption for the
  whole function. A `panic(r)` above the line that fills `r` is
  therefore exempt. A `r = fmt.Errorf(...)` before the panic does not
  take the exemption back either.

The first two limits report a shape that is a rethrow, and their author
writes the comment. The third one accepts a panic that raises another
value. All three shapes are rare. A value graph would answer better,
and 003 records that work for G05, which asks the same question.

**No setting silences the rule.** The comment is the configuration. A
project that cannot write one for a call disables the rule, which is
the escape that 003 records for every rule.

### Measurement

A scan of the whole standard library, tests included, reports 1538
findings in 361 files and 163 packages. The scan ran the standalone
binary with `-justifypanic` over `./...` in `GOROOT/src` with Go 1.26.2
on darwin/arm64.

1499 of the findings are the builtin `panic`, 35 are `os.Exit`, and 4
are calls of package `log`. The four are `log.Fatalf` in
`net/http/transport.go`, `log.Fatal` in
`internal/runtime/gc/internal/gen`, `log.Panic` in `net/rpc/client.go`,
and `log.Panicln` in `expvar`. No production file of the standard
library calls a `Fatal` method of a `log.Logger` value. The method
ruling therefore adds nothing to this number, and it closes the hole
anyway. `reflect` gives 234 findings, `runtime` 110, `math/big` 75,
`testing` 66, and `crypto/cipher` 45. 21 of the 35 `os.Exit` findings
sit in `testing`, which stops the test binary, and 6 sit in package
`log` itself.

The three exemptions hold findings back in that scan, and a run without
each one gives the number:

- The test-file exemption holds back 1162 findings. A run without it
  gives 2700, so test code writes 43 percent of the panics of the
  standard library.
- The rethrow exemption holds back 18 findings. Every one of them
  re-raises a value that a deferred `recover` caught. Most sit in a
  package that turns a panic into an error, such as `encoding/gob`,
  `text/template`, `fmt`, `go/types`, and `regexp/syntax`.
  `sync.WaitGroup` re-raises for another reason: it keeps its own
  `Done` call out of the way of the panic.
- The main and init exemption holds back 11 findings, and every one of
  them sits in an init function. They are the self-check of
  `crypto/internal/fips140`, the type registration of `encoding/gob`,
  and two of `runtime/mgcstack`.

`golang.org/x/tools` v0.41.0 reports 598 findings in 168 files with the
same binary. 104 of them, 17 percent, sit in files of a package main.
`cmd/toolstash` gives 30, `cmd/go-contrib-init` 15, and `cmd/stringer`
7. A repository with a `cmd` tree therefore pays a cost that the
standard library does not show. The rule exempts `main` and `init` and
no other function, because a library function beside `main` still takes
an API decision. A `cmd` package that grows a library inside it is the
case this rule reports.

The volume is the reason for the opt-in severity. Most Go code takes the
decision to panic without a word. A project that turns this rule on
therefore accepts a large first run. The number is information for that
decision and no measure of the rule. The standard library is a library
of libraries, and `reflect`, `runtime`, and `math/big` panic on a
programmer error by design.

A first run under golangci-lint arrives truncated. The findings of this
rule share one text, and `max-same-issues` keeps 3 of them by default.
`max-issues-per-linter` keeps 50 findings of the whole `antislop`
linter. Set both settings to 0 to read the list, which v2.10.1
documents in `golangci-lint run --help`. Such a project writes
the comment where the panic is the contract, as `bytes.Buffer.Truncate`
documents its own panic today.

## G12 `fullstructcomp`: compare the whole value, not field after field (opt-in)

A test can assert several fields of one value, one assertion for each
field. Such a test states one claim for each field it names. It states
no claim about the rest. The field that the code adds next joins the
value, and the test stays silent about it. A failing `require` call
also hides the fields behind it, because that call stops the test. One
comparison of the whole value against a want value states the whole
expectation. It reports every wrong field at once.

The name of the rule comes from the Google Go Style Guide. Its section
"Full structure comparisons" states the same advice.

Rejected:

```go
func TestCreate(t *testing.T) {
    got := Create("boot", 3)
    require.Equal(t, "boot", got.Name)
    require.Equal(t, 3, got.Count)
    require.Equal(t, "new", got.State)
}
```

Accepted:

```go
func TestCreate(t *testing.T) {
    got := Create("boot", 3)
    want := Item{Name: "boot", Count: 3, State: "new"}
    if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(Item{}, "ID")); diff != "" {
        t.Errorf("Create() = unexpected value (-want +got):\n%s", diff)
    }
}
```

The `IgnoreFields` call takes one name for each field that the test
cannot predict, and almost every fix needs one name. The rule counts
the names that one comparison would need, and that count decides a
report. A scan with the analyzer over the standard library and over
`golang.org/x/tools` v0.49.0 shows how the counts spread. The table
counts every group that the rule finds at two fields, before the gate
stops any of them.

| ignore names | standard library | `x/tools` |
| --- | --- | --- |
| 0 | 34 | 2 |
| 1 to 2 | 22 | 3 |
| 3 to 5 | 25 | 9 |
| 6 to 10 | 17 | 1 |
| 11 to 20 | 16 | 0 |
| above 20 | 20 | 8 |

81 of the 134 standard library groups need 5 names or fewer. Of the
rest, 20 need above 20 names. The roundtrip branch keeps 4 more groups,
which gives the 85 that the default reports. Every message of this rule
names the option, unless cmp answers the comparison with an `Equal`
method of the value. No option of cmp changes such a comparison.

### What the rule reads

**The files.** The rule reads test files. A test file is a file whose
name ends in `_test.go`. Every file of a package that the
`test-packages` setting names is a test file as well. The section "The
test-packages setting" states which packages count, and an entry there
adds findings for this rule. It reads the test files of the package
under test and the external test package the same way. It skips
generated files. 003 states the header forms the module accepts.

**The unit.** One function declaration. A function literal inside a
declaration belongs to that declaration, because a subtest closure is
the same test. A helper is a declaration of its own, and its base is a
parameter. `func compare(t *testing.T, want, got *User)` is such a
helper. Three of the clearest findings of the standard library live in
one: `os/user.compare`, `go/token.checkPos`, and
`go/scanner.checkPos`.

**The assertion sites.** Two forms.

- A call of the equality family of the testify module: `Equal`,
  `Equalf`, `EqualValues`, `EqualValuesf`, `Exactly`, and `Exactlyf`.
  The rule resolves the import path of the call. It reads the package
  name of a qualified call, and it reads the type of a receiver. A
  package with the same name and another path counts for nothing. The
  receiver form, which `r := require.New(t)` builds, counts like the
  package form.
- An `if` statement whose condition holds `==` or `!=`, and whose body
  calls an `Error` or `Fatal` method on a value of the `testing`
  package. The name of the method is not evidence by itself. A type of
  the project with an `Errorf` method therefore stops no report.

The failure call must be a direct statement of the body of the `if`.
A call inside a nested block, such as a second `if` or a loop, states
a condition that the rule cannot read. A walk of the whole body adds 4
findings to the standard library scan, and each one holds such a nested
condition.

**The base and the field.** The rule walks the chain of an operand
through a selector, an index, a pointer, and parentheses. It stops at
the name at the root. That name must be a variable, so a chain that
starts with the name of a package names no base. A call stops the walk,
because the result of a method is not a field of the receiver. The
field is the whole path under the base. So `got.A.B` names the field
`A.B`, which is another field than `A`. The rule groups by the declared
variable, and never by its name. Two values of one name, in two subtest
closures, therefore stay apart.

The rule resolves each step of the chain through the selections of the
type checker. A promoted field then gives the path of the embedded
structure that declares it. `got.Name` and `got.Embedded.Name` name one
field through one path, and the rule reads one field. The paragraph
"The cut" below needs the real path, because that count walks the type
of the value downwards.

**The counting.** A site that names exactly one field of a base
contributes that field. A site that names two or more fields of one
base contributes nothing, because the author compared those fields in
one statement already. The rule counts distinct fields, so one field
asserted twice is one field. It reports a base when the count reaches
the number that the `min` setting sets.

That measure matters. A count of field mentions, instead of
single-field sites, reports 37 standard library functions at four
fields. The count of single-field sites reports 20. The 17 others hold
a combined condition that needs no change.

A table-driven loop needs no rule of its own. The syntax tree holds one
site inside the body of the loop, so a table of 40 cases counts once.

The run of one base is a group, and one group gives one finding.

**The cost of the fix.** A count of fields is not enough. Two
assertions that reach one end field inside two large subtrees give a
fix whose ignore list states far more than the test does. An end field
is a field with no structure under it. The rule reads a
second condition for that reason, and it reports a base only when both
of these hold:

1. The count of distinct fields reaches the `min` setting.
2. The group is a roundtrip, or the fix needs no more ignore names than
   the `maxignore` setting allows.

A value that answers a comparison with an `Equal` method takes another
route, and the paragraph "The type at the anchor" states it.

**The anchor.** The anchor of a group is the deepest field path that
holds every asserted path. One comparison there states every claim of
the group, so the message names the anchor, as in
`compare got.Pagination as a whole`. The anchor is the base itself when
the assertions sit in two subtrees or more. A subtree anchor is rare in
the two public corpora: 3 of the 85 kept groups of the standard library
carry one, and none of `x/tools` does. The Measurement section
describes a probe of a first design of the gate. That probe met the
form more often in the private service, where a checklist reads one
part of a large response.

**The cut.** The cut counts the `cmpopts.IgnoreFields` names that one
comparison at the anchor needs. It reads the type from the anchor
downwards:

- A path that the test asserts costs nothing.
- A subtree that holds no asserted path costs one name, because one
  name covers the whole of it.
- A subtree that holds one asserted path pays for each field beside it.

The number that the cut gives is the cost of the group.

The walk follows `cmp`. It opens the structure under a pointer, a
slice, an array and a map. It stops at a type with an `Equal` method
that cmp can call. Such a type costs one name, because cmp calls the
method and reads no field under it. An unexported field, an interface field
and a type parameter each cost one name as well. The walk descends only where an
asserted path leads, so the length of that path bounds it. The rule
also walks a type to remove its containers, and that walk carries a
bound. A type can hold itself, as `type L []L` does, and such a type
never reaches a structure. The bound is 12 steps, and it is a guard. An
ordinary type reaches its structure in two or three steps, so the
number decides nothing.

**The type at the anchor.** The `Equal` stop holds below the anchor and
not at it. cmp calls the method of the value under comparison and reads
no field of it, so `cmpopts.IgnoreFields` skips nothing there. The rule
therefore opens the type at the anchor, whatever methods it carries,
and it reports such a group only at a cost of zero. A cost of zero
means the test names every field, and one plain `cmp.Diff` against a
want value states the same claims. The message of such a group names no
option, because no option applies. The route holds for a roundtrip as
well, and for the same reason.

The standard library holds two such groups, and both stay silent. Both
sit in `crypto/x509`. One asserts 2 fields of an `rsa.PrivateKey`,
which costs 7 names, and one asserts 10 fields of a `Certificate`,
which costs 62.

**The roundtrip.** A group is a roundtrip when `min` sites or more pair
the base with one other base. Each such site compares one field path of
the base against the same path of the other base. The other base must
hold a value of the same type. The rule removes a pointer, a slice, an
array and a map from both types before it compares them. Such a test
holds a want value already. The ignore list of its fix covers the
unstable fields alone, which the analyzer cannot predict, so the cost
gate reads no cost there.

The other base of such a site must name exactly one field itself. A
site that reads two fields of the other value states two claims about
it. The rule counts no field of that base there, so it reads no
roundtrip either. The roundtrip branch keeps 4 groups of the standard
library and 1 of `x/tools` that the cost alone drops.

Both parts of that definition are necessary, and the probe of the first
design measured that. 100 groups of the private service read a field of
another value, and only 17 of them read a value of the same type. The
other 83 compare a field of an unrelated record, such as
`require.Equal(t, order.ID, line.OrderID)`. No want value of the right
type stands there, so the fix writes one by hand and pays for it. Only
34 percent of the same 100 groups pair a path with the same path. The
standard library gives 72 percent, because its compare helpers walk the
same paths on both sides.

### The settings

`-fullstructcomp.min` sets the number of distinct fields that a
report needs. The `fullstructcomp-min` setting of the plugin takes the
same number. The default is **2**.

The default has a history, and both steps stand here. The first
measurement read the standard library and recommended **4**. At 4 the
standard library reports 15 of its 11169 test functions. A reading of
all 15 gave 5 clear improvements, 3 that improve with work, and 7 wrong
reports.

A second measurement then read the findings of a private Go service at
exactly 3 fields and at exactly 2 fields, and it set the default at 2.
That service holds 838 test functions and uses testify. Two forms fill
both groups of findings. The first creates a value and asserts the
fields it passed in. The second stores a value, reads it back, and
asserts the fields of the result. One such roundtrip asserts 3 fields
of a value that holds 29. Each form holds a want value that the test
can write, so each one is the form this rule asks for.

The mid-flow checkpoint is the counter-class of that reading. Such a
test runs a scenario in steps, and it asserts the one field that each
step changed. The section "Why the rule is opt-in" states it. A project
that meets that form raises the number to 3 or to 4. The flag is the
first escape such a project reaches for.

The second reading supersedes the first. The recommendation of 4 came
from the standard library, which writes no checklist. The default of 2
comes from the kind of codebase that this rule is for.

A number of 1, and every number below it, gives no useful rule. The
rule then reports every base with one single-field assertion, and a Go
test needs the spot check. A scan of the standard library at one field
reports 1056 test functions, which is 9.5 percent of them. Almost every
one of those assertions is correct as it stands.
`require.Equal(t, http.StatusOK, resp.StatusCode)` states the whole
claim of its test. A diff of the whole response would need a longer
ignore list than the assertion it replaces.

`-fullstructcomp.maxignore` sets the number of `cmpopts.IgnoreFields`
names that the comparison of a report may need. The
`fullstructcomp-maxignore` setting of the plugin takes the same number.
The default is **5**.

The number comes from a reading of the groups, and the table of the
section above shows the spread. Every group that a reader judged good
sits at 5 names or below. Three of them stand here. A three-field type
with two asserted fields costs 1 name. A response of 79 end fields with
four asserted pagination fields costs 2 names. A create-and-echo test
costs 5 names.

The group of the private service that gave the gate its shape needs 57
names for its 2 assertions. It is the counter-example of the class.

Keep rates of the two public corpora, from a scan with the analyzer:

| gate | standard library | `x/tools` |
| --- | --- | --- |
| no gate | 134 | 23 |
| 5 names | 85 | 15 |
| 5 names or the field count, whichever is higher | 85 | 15 |
| 8 names | 93 | 16 |
| one name for each asserted field | 63 | 7 |

The plain cap and the higher-of-two form keep the same groups in both
corpora. The plain cap is therefore the choice, because a reader
predicts one number more easily than two.

A ratio of asserted fields to the end fields of the type was the first
candidate, and the probe of the first design rejected it. The
counter-example asserts 2.2 percent of its end fields, which is 2 of
90. One roundtrip that a reader judged good asserts 6.8 percent, which
is 3 of 44. That roundtrip stores a value in the private service, reads
it back, and asserts three fields of the result.

Any global ratio that silences the first and keeps the second sits in
that window of 4 points. One added field moves a type across it. Inside
the window the filter is also blunt. At 5 percent it drops 42 groups of
the private service for no stated reason. At 7 percent it drops the
good roundtrip. A ratio for each subtree gives the same window, because
the counter-example asserts 2.5 percent of each of its two subtrees.
The setting is called `maxignore` for that reason. It counts names, and
it states no fraction.

A value of 0 reports a group whose fix carries no ignore name at all,
which means the assertions name every field of the value. A high value
gives the report volume of the rule before the gate. A high value still
keeps one class silent, which the paragraph "The type at the anchor"
states. No number reaches that class, because no ignore name reaches it
either.

### What the rule leaves alone

- **A base that a range clause of the function fills.** Such a base
  holds a table case, and not a produced value. Its fields meet values
  that the case carries, and no want value stands beside it. The
  exclusion removes 10 of 30 standard library groups at four fields,
  and 1 of 41 groups of the private service. A clause with `:=` and a
  clause with `=` both fill the variable, so both stop the reports of
  it.
- **A group whose fix needs more ignore names than the setting
  allows.** The class is two assertions that reach one end field inside
  two large subtrees. One such group of the private service asserts one
  total of each of two subtrees of a value of 90 end fields. The two
  assertions name one field pair through two paths. One comparison of
  the whole value needs 57 ignore names, which is 28 names for each
  assertion. The rule states nothing there, because the fix it knows is
  worse than the test it reads. The checklist needs no other advice, so
  the rule gains no message for the class.
- **A checklist on a type that carries its own `Equal` method**, where
  the assertions leave a field out. cmp calls the method and reads no
  field of the value, so the ignore list is not available. The test
  names a field that the rewrite cannot skip, and the rule has no fix
  to offer. The same checklist reports when it names every field,
  because one plain `cmp.Diff` then states the same claims.
- **Two `cmp.Diff` calls that read one base.** A merge of the two is
  not the advice. The private service holds 79 such function and base
  pairs, and the other corpora hold none. 55 of the 79 compare the same
  value before and after an action, which is a checkpoint and no
  duplicate. 6 more compare a "before" half and an "after" half against
  two different want values, which one comparison cannot state. Those
  two shapes cover the clear cases, and the other 18 pairs hold neither
  of them. A merge is correct only where the cost at the common parent
  is small. The shape that suggested the report has a cost of 57
  there.
- **A site that names two fields of one base**, such as
  `if a.X != w.X || a.Y != w.Y`. That statement is one compare.
- **A call outside the equality family.** `True`, `Len`, `Nil`,
  `Empty`, and `Contains` state another claim. `True(got.A == x)`
  states an equality in another form. `testifylint` already asks the
  author to write it as `Equal`, and this rule reads it after that
  change. `InDelta` states an equality with a tolerance, and the
  tolerance sits in the call, so the rule leaves it out of the count.
  `cmpopts.EquateApprox` carries the same tolerance into the fix.
- **A test harness that holds the produced value.** A test that writes
  `s.got.Name` and `s.got.Count` names two fields of the base `s`. Each
  site names one of them, so the two sites report. A test that writes
  `require.Equal(t, s.want.Name, s.got.Name)` names two fields of `s`
  in one site, so that site contributes nothing. Such a harness is
  therefore exempt in silence. The base is the root of the chain, and a
  narrower answer needs another unit of grouping.
- **A method of a testify suite.** `s.Equal(...)` inside a suite method
  names a type of the project, so the rule reads no assertion there.
  The private service imports the suite package nowhere, so this hole
  distorts no measurement above. A wider test needs the
  base-granularity answer of the paragraph above first.
- **A generated file**, and every file that is not a test file.

### Decisions the rule states

**A site that names one field of two bases counts for both.** A helper
compares `want.Name` against `got.Name`, and `want.Count` against
`got.Count`. Such a helper gets two reports: one for `want`, and one
for `got`. One `cmp.Diff` answers both. One report for each function is
the alternative. It would hide the second base of a function with two
produced values. The measurement of the codebases counts groups for the
same reason. A measurement of the duplicate positions gives 6 to 10
percent of the reports.

**An argument of the message counts like any other operand.** The call
`require.Equalf(t, want.A, got.A, "the name is %s", got.Name)` names
two fields of `got`, so it contributes nothing for that base. It still
names one field of `want`, so it counts for `want`. The rule reads the
operands of the call and judges no position in it. A run that reads the
value arguments alone gives the private service the same 156 findings
at two fields, and the same 40 at four. No file of the standard library
calls testify, so no such form reaches that scan. The decision changes
no count in either codebase today.

**The rule reports a base whose type holds an unexported field.** There
is no exemption for such a type. The message carries one more sentence
for it. `cmp.Diff` panics at run time, and not at build time, on an
unexported field anywhere in the type graph. The author needs
`cmp.AllowUnexported` there, and would not guess it. Without that
sentence the author writes a panic into a test that passes. The walk
starts at the anchor, because the comparison that the message asks for
reads the type there.

The private service settles the question. Not one of its 75 measured
groups needs the option, and not one of its 651 asserted fields is
unexported. Its tests read exported domain types from other packages.
The standard library gives the opposite result. 53 percent of its
asserted fields are unexported, because its tests sit in the package
under test. The rule reports both, and it names the option where the
option is needed.

The test of the type graph follows `cmp`. It walks a pointer, a slice,
an array, a map, and the fields of a structure. It stops at a type that
carries a usable `Equal` method. Three things make a method usable. The
method set of the type must hold it. The value must fit the parameter.
The result must be a boolean. `time.Time` carries such a method, and it
needs no option. Three forms carry the name and give no comparison, and
a run with go-cmp v0.7.0 panics on each one:

- a value field whose `Equal` method has a pointer receiver, because
  the method set of a value holds no such method;
- a value field whose `Equal` method takes a pointer, because the value
  does not fit the parameter;
- a field whose `Equal` method takes another type, such as
  `func (w W) Equal(n int) bool`.

A pointer field with `func (t *T) Equal(*T) bool` meets all three
requirements. The method set of the pointer holds the method, and
the pointer fits the parameter, so cmp calls it and the walk stops.

The walk reads the element of a map and never the key. cmp compares the
keys of a map as keys and reads no field under them. A run with go-cmp
v0.7.0 over a map with an unexported field in its key type returned a
diff and no panic.

**The note is a lower bound.** A missing note is not proof that the fix
needs no option. An interface field, such as the `PublicKey any` of
`crypto/x509.Certificate`, carries a value that the analyzer cannot
name. That value can hold unexported fields, and `cmp.Diff` then panics
where the message promised nothing. The note names what the types
state, and the author reads the panic for the rest.

**No comment stops a report.** `SAFETY:`, `PANICS:`, and `CONTRACT:`
state an invariant, a reason, or an API that the analyzer cannot read.
Here there is no invariant to state. There is only a judgement about
which form of a test reads better, so no marker fits. Five things stop
a report: the opt-in severity, the `disable` setting, the `min`
setting, the `maxignore` setting, and `//nolint:antislop` on the
golangci-lint path. 003 states all five.

### Why the rule is opt-in

Roughly half of the reports of the standard library at any number are
wrong. The rule fires on idiomatic Go.

The known false report is the mid-flow checkpoint. Its fields belong to
one value, and its assertions sit apart in the function. The rule
counts a group where the author wrote a sequence. At 3 or at 4 the form
mostly falls away, and the setting is the first thing such a project
changes.

The standard library holds three more forms that no number reaches.
`crypto/tls.checkConnectionState` asserts a non-zero cipher suite, a
length, and two fields that only TLS 1.3 fills. Its type also holds
unexported fields, so no want value exists. `runtime.TestMemStats`
asserts ranges and not equality. `crypto/x509` reads a `Certificate` of
52 fields with a `PublicKey any` and raw bytes. The analyzer cannot
separate two questions. One is "assert five fields of one produced
value". The other is "assert five separate properties of a value that
no expectation describes". That limit is a property of syntax, and not
a fault of the number.

The report volume is low at every number. At the default settings the
rule reports 85 findings in the standard library and 15 in `x/tools`.
The private service reports 66, in the scan that the Measurement
section dates. Without the cost gate the counts are 134, 23 and 156.
Those 134 sit in 110 of the 11169 test
functions of the standard library, which is 1.0 percent. The 156 sit in
108 of the 838 test functions of the service, which is 12.9 percent. At
a minimum of 4 fields the two shares are 0.13 percent and 4.2 percent.
A testify codebase writes checklists at every number, and the standard
library does not.

The cost gate carries limits of its own, and they belong here.

- The gate decides a group near the cap with one number, and it is
  sometimes wrong in both directions. Two fields of a value of 55 end
  fields sit at 4 or 5 names, which is inside the cap. A reader can
  judge either way.
- A report disappears when a type gains a field and the cost of the fix
  grows past the cap. The number is correct, and the change surprises
  the author.
- The roundtrip branch pays no cost at all. It trusts that a want value
  of the type of the base exists. The fix of such a test can still need
  many ignore names for the fields that a server sets.
- The roundtrip branch also keeps the checkpoint alive. A test that
  compares two snapshots of one type, one before an action and one
  after it, reads as a roundtrip to this rule. The 4 standard library
  groups that the branch keeps are two such pairs: two connection states
  in `crypto/tls`, and two memory statistics values in `runtime`. Each
  pair reports both of its bases, which makes 4 groups. The advice
  inverts the intent of such a test, which asks about the difference
  between the two snapshots, and not about their equality.
- The gate does not target the mid-flow checkpoint, which is the known
  false report of the standard library. Roughly half of the kept
  standard library reports are still wrong. The `min` setting stays the
  first escape for such a project.

The fix also takes a dependency. `github.com/google/go-cmp` is a direct
requirement of the private service and of `golang.org/x/tools`. The
standard library takes no external dependency at all. A project decides
that for itself, which is what an opt-in rule asks it to do.

The opt-in severity belongs to the golangci-lint plugin, which is the
path that reads a configuration file. `cmd/antislop` and `go vet
-vettool` read none, so they run this rule by default.
`-fullstructcomp=false` turns it off there. 003 records the split.

### Related work

No Go linter covers this.

`testifylint` v1.6.4 holds 25 checkers, and every one of them judges
one call alone. None counts assertions across statements, and none
names go-cmp. It composes with this rule and overlaps it nowhere.
`bool-compare` turns `True(a == b)` into `Equal`, and this rule then
counts the result. `gocritic` holds the nearest cousin,
`typeAssertChain`, which folds repeated type assertions into one type
switch. That is the same form of advice for another target.
`staticcheck`, `revive`, `exhaustruct`, `funlen`, `gocognit`, and
`dupl` all answer other questions. A group of ten assertions trips
none of them. Outside Go, `jest/max-expects` caps the number of
assertions of a test at 5. It never asks whether they read one value.

The advice is canonical, and this rule mechanises it:

- The Google Go Style Guide, "Full structure comparisons"
  (`google.github.io/styleguide/go/decisions.html#full-structure-comparisons`):
  "avoid writing test code that performs a hand-coded field-by-field
  comparison of the struct. Instead, construct the data that you're
  expecting your function to return, and compare directly using a deep
  comparison."
- The Go wiki, `go.dev/wiki/TestComments`, "Compare Full Structures".
  It states the same guidance, and it prefers `cmp` to
  `reflect.DeepEqual`.

The rule respects one boundary that the guide states itself. The guide
compares separate return values one by one. This rule counts the fields
of one base only, so two results of one call stay apart.

### Measurement

A scan of the whole standard library, tests included, reports 134
findings at two fields and 20 at four, with no cost gate. The scan ran
the standalone binary with `-fullstructcomp` over `./...` in
`GOROOT/src` with Go 1.26.2 on darwin/arm64.

A scan of the private service reports 156 findings at two fields and 40
at four, with no cost gate. The two scans hold the separation that the
severity rests on.

A scan with the cost gate at the default of 5 names reports 85 findings
in the standard library, which is 49 fewer. The same scan reports 15 of
the 23 findings of `golang.org/x/tools` v0.49.0. A scan of the private
service reports 66 of its 156. The exemption for a type with an `Equal`
method takes groups away and adds none. At the default it takes none,
because the cost gate already stops the two `crypto/x509` groups above.
It takes both at a high `maxignore`.

A first design of the gate came from a separate probe, which predicted
86 for the standard library. The probe built a field path from the text
of a selector, so it resolved a promoted field wrongly. Three groups
differ between the probe and the analyzer, and all three hold a
promoted field. The analyzer alone keeps one group of `compress/gzip`,
where the resolved paths sit under one embedded header and cost 3
names. The probe alone keeps two groups of `archive/zip`, where the
resolved paths cost 15 and 11 names. The analyzer is the reference
here, because it is the code that runs.

The `cmp.AllowUnexported` sentence follows the codebase in the same
way. 76 of the 134 standard library findings carry it. None of the 156
findings of the private service carries it. The decision to exempt no
type rests on that split.

A hand-written script measured the same two codebases before the
analyzer existed, and the two counts agree. The private service matches
exactly: 156 groups at two fields, in 108 functions, and 40 groups at
four, in 35 functions. The standard library gives 134 findings against
the 133 groups of that script. The one added finding sits in
`encoding/gob/codec_test.go`, where `(***i.A)[0] != 11` names one field
of `i` through parentheses. The analyzer reads parentheses, and the
script did not.

## G13 `errsemantics`: assert the identity of an error, not its text (opt-in)

A test that reads the message of an error decides from prose. No API
promises the words. Such a test passes for another error with the same
words, and it fails for the right error after a reword. The identity of
the error is the evidence: `errors.Is` names a sentinel, and
`errors.As` names a type. Both survive a reword.

The source is the Go wiki page TestComments, section "Test Error
Semantics"
(<https://go.dev/wiki/TestComments#test-error-semantics>): "don't use
string comparison to check what type of error your function returns."

Rejected:

```go
if !strings.Contains(err.Error(), "run id must be non-empty") {
    t.Errorf("Seed() error = %v, want a run id error", err)
}
```

Accepted:

```go
if !errors.Is(err, ErrEmptyRunID) {
    t.Errorf("Seed() error = %v, want ErrEmptyRunID", err)
}
```

G10 and G13 are one idea in two halves. G10 owns the type assertion on
an error, and G13 owns the string. Evidence about an error is its
identity, and never its prose.

### Where the rule reads

The rule reads test files only. A test file is a file whose name ends
in `_test.go`. Every file of a package that the `test-packages` setting
names is a test file as well. The section "The test-packages setting"
states which packages count, and an entry there adds findings for this
rule. Production
code that reads a message renders it for a person, and it decides no
test. The rule skips generated files. 003 states the header forms the
module accepts.

The rule reads the static type of the operand. It reports the
predeclared `error` type and an alias of it, such as `type E = error`.
That is the narrow test of G10, word for word. A type that declares
`Error() string` under another name is another type, and `errors.Is`
answers no question about it.

### The forms the rule reports

The rule resolves every callee to the package that declares it. A
local package named `strings` therefore reports nothing, and a library
that carries the names of testify under another import path reports
nothing either. The rule reports:

- an `err.Error()` call in an argument of `strings.Contains`,
  `strings.HasPrefix`, `strings.HasSuffix`, or `strings.EqualFold`;
- an `err.Error()` call in an argument of `regexp.MatchString` or of
  the `MatchString` method of `regexp.Regexp`;
- `ErrorContains`, `ErrorContainsf`, `Regexp`, and `Regexpf` of
  testify, where an argument carries an error or its message. The text
  asserts of testify take the error itself, so no `Error` call appears
  there. The rule reads every package under
  `github.com/stretchr/testify/`, so `assert`, `require`, and the
  receiver form of `Assertions` all report.

`regexp.Match` takes bytes. An error message reaches it through a
conversion, which is no direct argument, so that function is out. A
method of the project that carries the name `MatchString` is another
object, and the rule leaves it alone.

**The rule never reads the failure message of a testify call**. Every
testify function the rule reads takes the testing value first. The two
values it asserts on follow. Everything after those is the failure
message: the variadic `msgAndArgs` tail, and the format string of a
name that ends in `f`.
The rule therefore reads the first three arguments of such a call, and
the first two of the receiver form of `Assertions`. An error in the
failure message is diagnostic output, and
`assert.Regexp(t, "x", s, "read failed: %v", err)` stays clean. A
report there would name a correct line and would offer `errors.Is` as
the repair of a printed message.

### The equality setting

Two more forms stay off until the `errsemantics-equality` setting is
true. `-errsemantics.equality` is the same setting outside
golangci-lint:

- an `err.Error()` call in a `==` or `!=` comparison;
- `EqualError`, `EqualErrorf`, `Equal`, and `Equalf` of testify.
  `EqualError` takes the error itself. `Equal` takes two values of any
  type, so it reports only where an argument is an `Error` call. An
  assertion on the error value itself is no text assertion, and
  `testifylint` owns that ground.

A package that tests its own message text writes the equality form, and
the wiki page accepts that. The standard library holds 119 equality
findings and 163 default findings, so the setting holds back 119 of its
282 findings, which is 42 percent. The three private corpora of the
Measurement section hold 5 equality findings against 1093 findings in
all, so the setting costs those projects almost nothing.

### What stays clean

- **Diagnostic output.** `t.Errorf`, `t.Logf`, `t.Fatalf`, and
  `fmt.Sprintf` print a message and decide nothing. An `Error` call
  counts where it sits in an argument of a reported call, or in an
  operand of a reported comparison, and nowhere else.
- **A message that flows through a variable.** The rule reads the
  direct argument, so `msg := err.Error()` and a later
  `strings.Contains(msg, ...)` stay clean. This is a gap and no ruling.
- **A conversion of the message**, such as `[]byte(err.Error())`, for
  the same reason.
- **A message that `fmt.Sprintf` carries.** A `%v` of an error inside a
  format string launders it past the rule. This is the second gap, and
  a value graph would answer both. 003 records that work for G05.
- **A value that is no error.** A type that declares `Error() string`
  under another name is one such value. A plain string variable that
  holds a message is another. The rule reads types and no history.
- **A call through a function value**, such as `var c = strings.Contains`
  and a later `c(err.Error(), ...)`. The call names a variable, and the
  rule reads the object.
- **Every production file, and every generated file.** A helper
  package that is no test file is production code by this test. An
  `internal/testutil` that holds the assertions of a project is such a
  package, unless the `test-packages` setting names it. The rule reads
  a package that the setting names. A project that names no package
  keeps the file name test, which is the test the go tool applies.

### Positions and no marker

The diagnostic sits at the call, and at the operator of a comparison.
That expression is the predicate, and it is the line the author
rewrites.

No comment stops a report. `errors.Is` and `errors.As` exist in every
version of Go this module supports, so a marker would justify a shape
that has a fix. G10 states the same, and both rules keep the family of
markers small. The escapes are the opt-in severity, the `disable`
setting, and the equality setting.

### Measurement

The scan ran the standalone binary with `-errsemantics` over `./...` in
each corpus, with Go 1.26.2 on darwin/arm64. Three corpora hold
AI-assisted Go, and two hold idiomatic Go for comparison.

| Corpus | Test files | Default forms | Per test file | Equality forms |
| --- | --- | --- | --- | --- |
| GOROOT `src` | 1678 | 163 | 0.10 | 119 |
| `golang.org/x/tools` v0.41.0 | 270 | 24 | 0.09 | 7 |
| Private A | 139 | 31 | 0.22 | 1 |
| Private B | 86 | 130 | 1.51 | 0 |
| Private C | 404 | 927 | 2.29 | 4 |

Two rows name a public artifact, and a reader repeats those two scans.
The other three are private Go repositories of the author. Private A is
a service, Private B is a research tool, and Private C is an ingestion
pipeline. Those clones move with every commit. The scan ran on
2026-08-22, against Private A at `03ef4044`, Private B at `0faf8fc`,
and Private C at `3b102677`, which was `origin/main` of that repository
on that day. A later scan of the same corpora gives other numbers.

Private A, B, and C give 2.3, 15.6, and 23.6 times the findings per
test file of the standard library. No other rule of this project
separates the two kinds of corpus that far. Private A sits lowest,
because that project settled on `errors.Is` early.

### The severity decision

The severity is opt-in, and the volume is not the reason. The reason is
one sentence of the same wiki section: "It's OK to use string
comparisons to check that error messages coming from the package under
test satisfy some property, for example, that it includes the parameter
name."

The wiki page therefore permits a whole class inside the default forms,
and no form separates it. A reading of 28 findings measured the class.
The sample took 12 findings of GOROOT, 6 of Private C, 4 of Private B,
3 of Private A, and 3 of `x/tools`, at random from each list.

- All 28 read the message of an error that the package under test
  produced. The rule cannot see that, because the origin of an error
  value is no static fact.
- 7 decide which error came back, from a table field such as
  `test.err`, `tc.wantErrText`, or `tc.want`. That shape decides which
  error came back from the words alone, which the wiki page rejects,
  and it is the finding this rule wants.
- The other 21 check a property of a message the package owns. Private
  B asserts that a failure names its input file. Private C asserts that
  a message names the pipeline step that failed. `crypto/x509` asserts
  the words " but have public key of type ". Each one is the parameter
  check the wiki page names.
- 2 of the 3 Private A findings sit one line below an identity
  assertion on the same error, which is `require.ErrorIs` or
  `errors.Is`. The test already asks the question the rule asks, and
  the text check adds a property of the message. A report there names
  correct code.

The permitted class covers less than those 21 findings suggest. One
Private C finding of the sample matches the text of an error that the
test itself injected through a stub. The test owns that value, so no
message of the package under test is under examination. `errors.Is`
against the injected sentinel is a repair of one line. A review of a
larger sample would move findings from the permitted class into the
class the rule wants. A second reading of 20 more findings reached the
same share, so the opt-in severity stands. A project that wants this
rule at error severity re-measures that share first.

An opt-in rule that a project turns on deliberately is the honest
answer. Such a project decides that its tests must read no message at
all, and it accepts the property checks with them. A default severity
would report 163 findings against the standard library, and the wiki
page permits most of them.

The project considered one refinement and rejected it. The rule could
skip a text check that sits beside an identity assertion on the same
error. That heuristic reads a second statement and guesses at the
intention. It also gives nothing to the standard library, where the
sample held no identity assertion at all. 001 asks for the smaller
check, and the opt-in severity is smaller than the heuristic.

### Prior art

No linter reads this ground. Both of the linters that come near stop at
the error value:

- `errorlint` reports `err == target`, `err.(*T)`, and a `%v` verb
  where `%w` belongs. Its comparison check reads the error value. This
  rule reads the string that `Error()` returns.
- `testifylint` `error-is-as` reads the error value as well. In v1.6.4
  it reports `assert.Error` and `assert.NoError` with a sentinel in the
  `msgAndArgs` tail. It reports `assert.True` and `assert.False` around
  an `errors.Is` or `errors.As` call, and `assert.IsType` on an error. It
  names `EqualError` and `ErrorContains` under `require-error` only,
  which asks for `require` in the place of `assert`. No checker of that
  linter reads the text of a message.

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

## The test-packages setting, shared by G07, G08, G11, G12, and G13

Five rules must decide whether a file is a test file or a production
file. The `test-packages` setting states the answer for a whole
package, and the five rules read one implementation of it.

**The test.** A file is a test file when its name ends in `_test.go`.
It is a test file as well when the import path of its package matches
a pattern of the setting. The patterns take the syntax that every
other package-naming setting takes, and 003 states it.
`boundary-packages` and `reflect-allow` take the same syntax. The test
drops a trailing `_test` from the path, so one entry covers a package
and its external test package. An empty list names no package, which
is the default of all five rules.

**Why the file name is not enough.** A project holds packages that
serve tests and carry no `_test.go` name. A shared suite that a
`TestMain` function starts is such a package. The go tool builds it
like production code, and the file name test therefore reads it as
production code. The five rules then judge it by the wrong standard.

**The evidence.** An adoption review of the private service measured
the cost. G11 reports 23 findings there, and 12 of them sit in two
test-helper files. The reviewer read those 12 as noise. A panic in a
test helper stops one test binary, and its author reads the stack
trace at once. That is the reason the rule exempts a `_test.go` file
already.

The entry that removes those 12 findings also adds 2 findings of G08.
A helper there assigns a package-level variable of a third-party web
library, and it restores the value in a cleanup. Both directions of
the setting therefore show in one repository.

**What each rule does with the answer.** G07 gives such a package the
`reflect.DeepEqual` allowance of a test file. G08 reads the
assignments of such a package, and it treats a package-level variable
that the package declares as test infrastructure. G11 asks for no
`PANICS:` comment there.

G12 and G13 read no such package today, and this change makes them
read it. An entry therefore adds findings for those two rules, and it
removes findings for the other three. Both directions are the point. A
helper that asserts field after field, or that reads the text of an
error, is a test assertion wherever it sits.

**One entry names one kind of package.** A package that serves tests
and holds production code as well loses reports of G07, G08, and G11.
It loses them for its production code too. Name a package that serves
tests alone.

The golangci-lint plugin reads the key `test-packages`. The standalone
paths read `-noreflect.testpackages`, `-nomonkeypatch.testpackages`,
`-justifypanic.testpackages`, `-fullstructcomp.testpackages`, and
`-errsemantics.testpackages`. Each flag takes the patterns as a
comma-separated list, and a repeated flag adds patterns. 003 states
the setting with the other keys.

## Rules considered and rejected

- **Structural-name bans** (upstream `no-shape-in-symbol-names`):
  the upstream rule polices a TypeScript naming habit. A Go version
  would ban words like `Data` or `Info` and would fire on idiomatic
  code. Out of scope.
- **Ignored errors**: `errcheck` owns this. Duplication creates
  configuration drift.
- **Comment and documentation tone**: out of scope; see 001 non-goals.
- **Narrow a parameter that the body downcasts**: a function that
  asserts a parameter to a narrower type looks like a function with a
  weak signature. Such a function cannot proceed when the assertion
  fails. A scan of the standard library, `golang.org/x/tools`, and
  three project corpora found 168 such assertions in the two idiomatic
  corpora. It found 3 in the project corpora, and all 3 sit in test
  files. In 85 percent of them the caller holds only the wide type, so
  no signature change is available. 57 percent assert on a sealed
  interface, where the type switch is a sum type and `gochecksumtype`
  is the right tool. G01 already owns the single-result form. Out of
  scope.
