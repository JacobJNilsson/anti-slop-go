# anti-slop-go

Opinionated `go/analysis` rules that reject low-evidence Go patterns.

This project is a Go companion to [dmmulroy/anti-slop](https://github.com/dmmulroy/anti-slop).
The upstream project targets TypeScript and JavaScript through Oxlint.
This project applies the same philosophy to Go.

## Status

Implementation phase. Six analyzers are available, and all six run by
default: `safetyassert` (rule G01), `nountypedmap` (G02), `noanyparam`
(G03), `noanyreturn` (G04), `nolaundering` (G05), and `noerrorassert`
(G10). Read the specification in [`docs/spec`](docs/spec):

1. [Overview](docs/spec/001-overview.md): philosophy, goals, and scope.
2. [Rules](docs/spec/002-rules.md): the rule catalogue with examples.
3. [Implementation](docs/spec/003-implementation.md): architecture, distribution, and configuration.

## The idea in one paragraph

Code generators produce code that compiles but carries no evidence.
A type assertion with no stated invariant, an `any` parameter, or a
`map[string]any` field moves a proof obligation from the author to the
reader. These rules reject such patterns. The author must decode input
at its I/O boundary, keep concrete types inside the program, and write
a `// SAFETY:` justification where an assertion is the correct tool.

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
    version: v0.2.0
```

The `import` line is necessary. The registration lives in the `plugin`
subpackage, not in the module root. The `version` line takes a tag of
this repository; `v0.2.0` is the release that this section shipped in.

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
          disable:
            - nountypedmap
```

Four points about this file:

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
- `disable` drops a rule from the default set, which holds the five
  rules that the Status section names. A configuration that disables
  every rule is legal, and the linter then reports nothing. `enable`
  turns on an opt-in rule. No rule is opt-in yet, so `enable` takes no
  name today, and a name that is on by default stops the run.

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
