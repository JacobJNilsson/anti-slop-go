# anti-slop-go

Opinionated `go/analysis` rules that reject low-evidence Go patterns.

## The idea in one paragraph

Code generators produce code that compiles but carries no evidence.
A type assertion with no stated invariant, an `any` parameter, or a
`map[string]any` field moves a proof obligation from the author to the
reader. These rules reject such patterns. The author must decode input
at its I/O boundary, keep concrete types inside the program, and write
a justification comment where an assertion is the correct tool.

## The rules

Eight rules run by default.

| ID | Rule | Reports |
| --- | --- | --- |
| G01 | `safetyassert` | A panicking type assertion without a justification comment above it. |
| G02 | `nountypedmap` | A map with an `any` value type in a signature, a struct field, or a package variable. |
| G04 | `noanyreturn` | An `any` result. |
| G05 | `nolaundering` | A value that passes through `any` and comes back through an assertion. |
| G06 | `noadhoctypeswitch` | A type switch on an `any` value outside a package that decodes input. |
| G07 | `noreflect` | An import of `reflect` outside an allowed package. A test file that only calls `reflect.DeepEqual` stays clean. |
| G08 | `nomonkeypatch` | A test that rewires production code: an assignment to a package-level variable, an import of a runtime patching library, or a `//go:linkname` directive. |
| G10 | `noerrorassert` | A type assertion or a type switch on an `error` value, where `errors.As` answers the question. |

Six rules are opt-in. The `enable` setting of the golangci-lint plugin
turns one on.

| ID | Rule | Reports |
| --- | --- | --- |
| G03 | `noanyparam` | An `any` parameter outside the exemptions that the specification states. |
| G09 | `nointerfacereturn` | An interface result where every return statement builds the same concrete type. |
| G11 | `justifypanic` | A `panic`, an `os.Exit`, or a `log.Fatal` call outside `main`, `init`, and test files, with no justification comment above it. |
| G12 | `fullstructcomp` | A test that asserts a value field by field instead of one `cmp.Diff`. |
| G13 | `errsemantics` | A test that reads the text of an error instead of its identity. |
| G14 | `separategotwant` | A test that calls a helper taking the testing value inside an assertion argument, instead of binding got and want first. |

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
    version: vX.Y.Z
```

The `import` line is necessary. The registration lives in the `plugin`
subpackage, not in the module root. Replace `vX.Y.Z` with a tag of
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
          test-packages:
            - example.com/app/internal/suite
          enable:
            - noanyparam
            - nointerfacereturn
            - justifypanic
            - fullstructcomp
            - errsemantics
            - separategotwant
          disable:
            - nountypedmap
```

Run the new binary with `./custom-gcl run ./...`. The plugin is verified
against golangci-lint v2.10.1. Other v2 releases are untested.

### Select the rules

`type: module` is necessary. Without it, golangci-lint looks for a
shared object file.

All rules arrive as one linter named `antislop`. It joins the standard
group of linters, so the default `linters.default: standard` runs it
without a `linters.enable` entry. A configuration that sets
`linters.default: none` needs the entry.

The plugin selects the individual rules with its own `enable` and
`disable` settings, not with `linters.enable`:

- `disable` drops a rule from the default set, which is the first table
  above. A configuration that disables every rule is legal, and the
  linter then reports nothing.
- `enable` turns on an opt-in rule from the second table. A rule that is
  on by default stops the run, because `enable` would do nothing for it.
- An unknown rule name in either setting stops the run.

### Settings

An unknown settings key stops the run. The standalone binary and
`go vet -vettool` read none of this file. They take each setting as a
flag instead.

| Setting | Rule | Default | Standalone flag |
| --- | --- | --- | --- |
| `boundary-packages` | G06 `noadhoctypeswitch` | none | `-noadhoctypeswitch.boundary` |
| `reflect-allow` | G07 `noreflect` | none | `-noreflect.allow` |
| `fullstructcomp-min` | G12 `fullstructcomp` | 2 | `-fullstructcomp.min` |
| `fullstructcomp-maxignore` | G12 `fullstructcomp` | 5 | `-fullstructcomp.maxignore` |
| `errsemantics-equality` | G13 `errsemantics` | false | `-errsemantics.equality` |
| `test-packages` | G07, G08, G11, G12, G13, G14 | none | `-<rule>.testpackages`, one flag per rule |

`boundary-packages` names the packages that decode input. A type switch
on an `any` value is the work of such a package, so G06 accepts every
one of them there.

`reflect-allow` names the packages that may import `reflect`.

`fullstructcomp-min` is the number of distinct fields of one value that
a G12 report needs. A project that meets the mid-flow checkpoint shape,
where each step of a scenario asserts the one field it changed, raises
the number.

`fullstructcomp-maxignore` is the number of `cmpopts.IgnoreFields`
names that a G12 fix may need. G12 reports no group above the setting,
because such a fix states more than the assertions it replaces. A
project that wants every checklist reported sets a high number.

`errsemantics-equality` adds a G13 report for a comparison of an error
message against a string, such as `err.Error() == "..."` and the
`EqualError` assertion of testify. It is off by default, because a
package that tests its own message text writes that form.

`test-packages` names the packages that serve tests and hold no file
whose name ends in `_test.go`. A shared suite that a `TestMain` function
starts is such a package. Six rules must decide whether a file is a test
file, and this key answers for all six:

- G07 gives the `reflect.DeepEqual` allowance of a test file.
- G08 reads the assignments as test code.
- G11 asks for no justification comment.
- G12, G13, and G14 read no such package today, so an entry adds
  findings for those three.

### Package path patterns

`boundary-packages`, `reflect-allow`, and `test-packages` name packages
by path pattern. A pattern matches the whole import path:

- `*` matches inside one path segment.
- `...` crosses a slash.
- A pattern that ends in `/...` also names the package above it.

A standalone flag takes the patterns as a comma-separated list or as a
repeated flag.

## Development

Run `make setup` once per clone; it installs the tracked git hooks.
`make check` is the definition of green: tidy check, vet, lint, the
coverage-gate self-test, race-enabled tests behind a statement coverage
gate (`COVERAGE_MIN`, default 90%), and the build. Read
[AGENTS.md](AGENTS.md) before your first commit and
[REVIEW.md](REVIEW.md) before your first pull request.

## Related project

This project is a Go companion to [dmmulroy/anti-slop](https://github.com/dmmulroy/anti-slop).
The upstream project targets TypeScript and JavaScript through Oxlint.
This project applies the same philosophy to Go.

## License

MIT. See [LICENSE](LICENSE).
