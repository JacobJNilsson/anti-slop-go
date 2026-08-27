# anti-slop-go

Opinionated `go/analysis` rules that reject low-evidence Go patterns.

This project is a Go companion to [dmmulroy/anti-slop](https://github.com/dmmulroy/anti-slop).
The upstream project targets TypeScript and JavaScript through Oxlint.
This project applies the same philosophy to Go.

## The idea in one paragraph

Code generators produce code that compiles but carries no evidence.
A type assertion with no stated invariant, an `any` parameter, or a
`map[string]any` field moves a proof obligation from the author to the
reader. These rules reject such patterns. The author must decode input
at its I/O boundary, keep concrete types inside the program, and write
a `// SAFETY:` justification where an assertion is the correct tool.

## The rules

Nine rules run by default.

| ID | Rule | Reports |
| --- | --- | --- |
| G01 | `safetyassert` | A panicking type assertion without a `SAFETY:` comment above it. |
| G02 | `nountypedmap` | A map with an `any` value type in a signature, a struct field, or a package variable. |
| G03 | `noanyparam` | An `any` parameter outside the exemptions that the specification states. |
| G04 | `noanyreturn` | An `any` result. |
| G05 | `nolaundering` | A value that passes through `any` and comes back through an assertion. |
| G06 | `noadhoctypeswitch` | A type switch on an `any` value outside a package that decodes input. |
| G07 | `noreflect` | An import of `reflect` outside an allowed package. A test file that only calls `reflect.DeepEqual` stays clean. |
| G08 | `nomonkeypatch` | A test that rewires production code: an assignment to a package-level variable, an import of a runtime patching library, or a `//go:linkname` directive. |
| G10 | `noerrorassert` | A type assertion or a type switch on an `error` value, where `errors.As` answers the question. |

Four rules are opt-in. The `enable` setting of the golangci-lint plugin
turns one on.

| ID | Rule | Reports |
| --- | --- | --- |
| G09 | `nointerfacereturn` | An interface result where every return statement builds the same concrete type. |
| G11 | `justifypanic` | A `panic`, an `os.Exit`, or a `log.Fatal` call outside `main`, `init`, and test files, with no `PANICS:` comment above it. |
| G12 | `fullstructcomp` | A test that asserts a value field by field instead of one `cmp.Diff`. |
| G13 | `errsemantics` | A test that reads the text of an error instead of its identity. |

The IDs come from the specification, where the rules stand in the order
of their writing. Each table therefore skips the IDs of the other.

The specification carries the full contract of each rule, with examples
and the measurements behind every decision:

1. [Overview](docs/spec/001-overview.md): philosophy, goals, and scope.
2. [Rules](docs/spec/002-rules.md): the rule catalogue.
3. [Implementation](docs/spec/003-implementation.md): architecture, distribution, and configuration.

## A first run

The standalone binary needs no setup:

```sh
go run github.com/JacobJNilsson/anti-slop-go/cmd/antislop@latest ./...
```

This path reads no configuration file, so it runs every rule, the
opt-in ones included. `-justifypanic=false` turns that one rule off.
`-errsemantics` alone runs that one rule and no other. The same
command satisfies the `go vet -vettool` contract: install it with
`go install github.com/JacobJNilsson/anti-slop-go/cmd/antislop@latest`,
then give `-vettool` the path of the installed binary.

## Use with golangci-lint

golangci-lint loads these rules as a module plugin. A module plugin is
Go code, so you build a golangci-lint binary that contains it. Put a
`.custom-gcl.yml` in the root of your project:

```yaml
version: v2.10.1
name: custom-gcl
destination: .
plugins:
  - module: github.com/JacobJNilsson/anti-slop-go
    import: github.com/JacobJNilsson/anti-slop-go/plugin
    version: v1.1.0
```

The `import` line is necessary. The registration lives in the `plugin`
subpackage, not in the module root. The `version` line takes a tag of
this repository. Take the newest one from the
[tag list](https://github.com/JacobJNilsson/anti-slop-go/tags).

Run `golangci-lint custom` in that directory. The command clones
golangci-lint, adds this module, and writes a `custom-gcl` binary. It
needs network access, `git`, and a Go toolchain that satisfies the `go`
directive of this module (see [go.mod](go.mod); Go 1.26 today).

Then configure the linter in `.golangci.yml`:

```yaml
version: "2"
linters:
  enable:
    - antislop
  settings:
    custom:
      antislop:
        type: module
        description: Rejects low-evidence Go patterns.
        original-url: github.com/JacobJNilsson/anti-slop-go
        settings:
          boundary-packages:
            - example.com/app/internal/ingest
          reflect-allow:
            - example.com/app/internal/codec
          fullstructcomp-min: 3
          fullstructcomp-maxignore: 5
          errsemantics-equality: true
          enable:
            - nointerfacereturn
            - justifypanic
            - fullstructcomp
            - errsemantics
          disable:
            - nountypedmap
```

Eleven points about this file:

- `type: module` is necessary. Without it, golangci-lint looks for a
  shared object file.
- `antislop` joins the standard group of linters, so a configuration
  that keeps the default `linters.default: standard` runs it without
  the `linters.enable` entry. The entry becomes necessary when the
  configuration sets `linters.default: none`. The example keeps it,
  because it states the intention.
- All rules arrive as one linter named `antislop`. You select the
  individual rules with the plugin's own `enable` and `disable`
  settings, not with `linters.enable`. An unknown rule name in either
  plugin setting stops the run.
- `disable` drops a rule from the default set, which holds the nine
  rules of the first table above. A configuration that disables every
  rule is legal, and the linter then reports nothing. `enable` turns on
  an opt-in rule from the second table. A name that is on by default
  stops the run, because `enable` would do nothing for it.
- The standalone binary and `go vet -vettool` read none of this file.
  The section "A first run" states what they read instead.
- Two settings name packages by path pattern. A pattern matches the
  whole import path: `*` holds inside one segment, `...` crosses a
  slash, and a pattern that ends in `/...` names the package above it
  as well. The standalone binary takes the same patterns in a flag, as
  a comma-separated list or a repeated flag. An unknown settings key
  stops the run, so this file names only the keys that a rule reads
  today.
- `boundary-packages` names the packages that decode input, which rule
  `noadhoctypeswitch` (G06) reads. A type switch on an `any` value is
  the work of such a package, so the rule accepts every one of them
  there. The standalone flag is `-noadhoctypeswitch.boundary`.
- `reflect-allow` names the packages that may import `reflect`, which
  rule `noreflect` (G07) reads. The standalone flag is
  `-noreflect.allow`.
- `fullstructcomp-min` names the number of distinct fields of one value
  that a report of rule `fullstructcomp` (G12) needs. The default is 2.
  A project that meets the mid-flow checkpoint shape, where each step
  of a scenario asserts the one field it changed, raises the number.
  The standalone flag is `-fullstructcomp.min`.
- `fullstructcomp-maxignore` sets the number of `cmpopts.IgnoreFields`
  names that a comparison of rule `fullstructcomp` (G12) may need. The
  rule counts those names. It reports no group above the setting,
  because such a fix states more than the assertions it replaces. The
  default is 5. A project that wants every checklist reported sets a
  high number. The standalone flag is `-fullstructcomp.maxignore`.
- `errsemantics-equality` is a boolean, and rule `errsemantics` (G13)
  reads it. It adds a report for a comparison of an error message
  against a string, such as `err.Error() == "..."` and the
  `EqualError` assertion of testify. The default is false, because a
  package that tests its own message text writes that form. The
  standalone flag is `-errsemantics.equality`.

Run the new binary with `./custom-gcl run ./...`.

Supported golangci-lint versions: the plugin is verified against
v2.10.1. The v2.9 line shares the same plugin register API and the same
`golang.org/x/tools` requirement, so it works too. Earlier v2 releases
are untested.

## Development

Run `make setup` once per clone; it installs the tracked git hooks.
`make check` is the definition of green: tidy check, vet, lint, the
coverage-gate self-test, race-enabled tests behind a statement coverage
gate (`COVERAGE_MIN`, default 90%), and the build. Read
[AGENTS.md](AGENTS.md) before your first commit and
[REVIEW.md](REVIEW.md) before your first pull request.

## License

MIT. See [LICENSE](LICENSE).
