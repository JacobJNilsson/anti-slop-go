# 002: Rule catalogue

Each rule has an identifier (G01-G11), an analyzer name, a default
severity, and a statement of what it rejects and why. Examples show a
rejected form and an accepted form.

Severity levels:

- **error**: on by default.
- **opt-in**: off by default; the project must enable it.

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
containing statement, and must match `SAFETY:`.
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

Exemptions, checked by signature shape:

- variadic `...any` followed by a format-string parameter before it
  (`fmt`-style helpers),
- parameters named `cause` in error-wrapping helpers (mirrors upstream),
- implementations of external interfaces that require `any`
  (`json.Marshaler` patterns, `sql/driver.Valuer` inputs).

Rejected:

```go
func Store(key string, value any) error
```

Accepted:

```go
func Store[V Storable](key string, value V) error
```

## G04 `noanyreturn`: no `any` returns (error)

An `any` return forces every caller to assert. Return the concrete
type, or a small interface the caller consumes, or a generic result.

Rejected:

```go
func Lookup(key string) (any, error)
```

Accepted:

```go
func Lookup(key string) (Record, error)
```

## G05 `nolaundering`: no widen-then-assert (error)

A value must not pass through `any` (or a broader interface) and come
back through an assertion in the same function or call chain the
analyzer can see. The widening destroys evidence the program already
had; the assertion re-creates it as a guess.

Rejected:

```go
var v any = user
u := v.(User)
```

Rejected (chained):

```go
u := any(user).(User)
```

Accepted: keep `user` typed as `User`.

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

## Rules considered and rejected

- **Structural-name bans** (upstream `no-shape-in-symbol-names`):
  the upstream rule polices a TypeScript naming habit. A Go version
  would ban words like `Data` or `Info` and would fire on idiomatic
  code. Out of scope.
- **Ignored errors**: `errcheck` owns this. Duplication creates
  configuration drift.
- **Comment and documentation tone**: out of scope; see 001 non-goals.
