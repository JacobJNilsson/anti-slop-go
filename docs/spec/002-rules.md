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
boundary. Branch on a domain value instead: a kind field, a sealed
interface with a marker method, or distinct handler functions.

Allowed positions, because Go idiom requires them there:

- inside functions the configuration marks as decode boundaries,
- `error` values handled with `errors.As` / `errors.Is` (see G10),
- implementations of `fmt.Formatter`, marshalers, and similar
  external contracts.

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

## G07 `noreflect`: no reflection outside allowlisted packages (error)

`reflect` erases every static guarantee. Serialization libraries need
it; application code does not. The configuration allowlists packages
(by path pattern) that may import `reflect`. `reflect.DeepEqual` in
test files is exempt by default.

## G08 `nomonkeypatch`: no monkey patching in tests (error)

A test must not rewire production code through mutable globals.
The seam belongs in the design: accept an interface or a function
value as a dependency.

The rule flags, in `_test.go` files:

- assignment to a package-level function variable declared outside the
  test file,
- imports of runtime-patching libraries (`bou.ke/monkey` and
  equivalents),
- `//go:linkname` directives.

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

Wrapped errors defeat direct assertions and direct type switches.
Use `errors.As` and `errors.Is`, which walk the wrap chain.

Rejected:

```go
if pe, ok := err.(*fs.PathError); ok { ... }
```

Accepted:

```go
var pe *fs.PathError
if errors.As(err, &pe) { ... }
```

`staticcheck` covers parts of this; the rule exists so the set stays
complete when a project runs this project alone. The configuration can
turn it off when `staticcheck` runs with the equivalent checks.

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
