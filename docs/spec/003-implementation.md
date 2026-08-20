# 003: Implementation and configuration

## Architecture

Each rule is one `golang.org/x/tools/go/analysis.Analyzer`. Analyzers
use the type information in `analysis.Pass.TypesInfo`; none of the
rules work on syntax alone. The module layout:

```
anti-slop-go/
  analyzers/
    safetyassert/
      safetyassert.go
      safetyassert_test.go
      testdata/src/...
    nountypedmap/
    ...
  antislop.go        // exported list of all analyzers
  cmd/antislop/      // multichecker binary
  plugin/            // golangci-lint module plugin entry point
  docs/spec/
```

- `antislop.go` exports `func Analyzers() []*analysis.Analyzer`.
- `cmd/antislop` wraps the list in
  `golang.org/x/tools/go/analysis/multichecker`.
- `plugin` holds the `register.Plugin` call. It stands apart, so a
  consumer of `cmd/antislop` or of `Analyzers()` never compiles the
  golangci-lint dependency.
- Tests use `analysistest` with `testdata` packages. Every rule ships
  with accepted and rejected fixtures before it merges.

## Distribution

Three consumption paths, in order of priority:

1. **golangci-lint module plugin.** Provide the
   `register.Plugin("antislop", ...)` entry point so a project adds
   the rules through `.custom-gcl.yml`. This is the primary path.
2. **Standalone binary.** `go run github.com/JacobJNilsson/anti-slop-go/cmd/antislop@latest ./...`
   for projects without golangci-lint.
3. **`go vet -vettool`.** The same binary satisfies the vet tool
   contract.

## Configuration

The settings block is the configuration surface of the plugin.
golangci-lint gives the block to `New` through
`register.DecodeSettings`, and it never sets `analysis.Analyzer.Flags`.
`BuildAnalyzers` translates the decoded settings into the analyzer set
it returns.

The standalone `cmd/antislop` binary and the `go vet -vettool` path
read no configuration file. They configure through analyzer flags, once
a rule has a flag. The plugin then sets the same flags from its
settings, so one parser serves both paths.

The decoder rejects an unknown key. A key therefore appears in the
settings only when a rule reads it, and a key that ships stays.

```yaml
# .golangci.yml (module plugin form)
version: "2"
linters:
  settings:
    custom:
      antislop:
        type: module
        settings:
          boundary-packages:        # G06: decode boundaries
            - "*/internal/ingest"
            - "*/api"
          reflect-allow:            # G07
            - "*/internal/codec"
          disable:
            - noerrorassert         # when staticcheck covers it
          enable:
            - nointerfacereturn     # opt-in rules
            - justifypanic
```

The example shows the target state. Only `enable` and `disable` exist
today; `boundary-packages` and `reflect-allow` arrive with G06 and G07.

Semantics:

- `boundary-packages`: package path patterns where decode-time
  dynamic checks are legal. G06 allows type switches here. G02 needs
  no entry, because it never flags a local variable.
- `reflect-allow`: package path patterns that may import `reflect`.
- `enable` / `disable`: rule toggles. Defaults follow the severity
  column in 002. `disable` drops a rule from the default set. `enable`
  turns on an opt-in rule, so it rejects a rule that is on by default.
  Both settings reject a name that is not a rule, and the run stops. A
  configuration that disables every rule is legal; the linter then
  reports nothing.

This project adds no inline suppression comments. The `SAFETY:` and
`PANICS:` comments are justifications with content, not switches. A
finding that a project cannot fix and cannot justify indicates a wrong
rule; file an issue instead.

One caveat, recorded after a test with golangci-lint v2.10.1: on the
golangci-lint path, `//nolint:antislop` suppresses a finding. That
mechanism belongs to golangci-lint, and a plugin cannot turn it off.
The `go vet -vettool` path has no such directive. A project that wants
a reason on every directive enables the `nolintlint` linter.

## The SAFETY comment contract

Shared by G01 and G11 (`PANICS:`), and identical to upstream:

- The comment sits directly above the flagged statement, or above the
  statement that contains it.
- The marker matches the regular expression `\bSAFETY\s*:` (or
  `\bPANICS\s*:`).
- The text after the marker must state the invariant. The analyzer
  cannot judge the text; review must.

## Milestones

1. **M1**: module skeleton, `safetyassert` (G01), `nountypedmap`
   (G02), CI with `analysistest`, first tagged release.
2. **M2**: `noanyparam` (G03), `noanyreturn` (G04), `nolaundering`
   (G05), golangci-lint plugin entry point.
3. **M3**: `noadhoctypeswitch` (G06), `noreflect` (G07),
   `nomonkeypatch` (G08), `noerrorassert` (G10).
4. **M4**: opt-in rules `nointerfacereturn` (G09) and `justifypanic`
   (G11); configuration polish; dogfood on a real project and tune
   false positives before 1.0.

## Open questions

- G05 scope: single-function analysis is cheap; cross-function
  laundering needs the SSA pass. Start single-function; measure the
  escape rate on real code.
- G06 sealed-interface guidance: should the analyzer suggest the
  marker-method pattern in its diagnostic, or stay silent on the fix?
- Generics: G03 accepts constrained type parameters. Decide whether an
  unconstrained `[T any]` parameter used only for pass-through counts
  as evidence or as laundering.
