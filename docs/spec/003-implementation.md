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
  docs/spec/
```

- `antislop.go` exports `func Analyzers() []*analysis.Analyzer`.
- `cmd/antislop` wraps the list in
  `golang.org/x/tools/go/analysis/multichecker`.
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

Configuration travels through analyzer flags, so every consumption
path shares one format. golangci-lint maps its settings block onto the
same flags.

```yaml
# .golangci.yml (module plugin form)
linters-settings:
  custom:
    antislop:
      settings:
        boundary-packages:        # G02, G06: decode boundaries
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

Semantics:

- `boundary-packages`: package path patterns where decode-time
  dynamic checks are legal. G02 allows local untyped maps here; G06
  allows type switches here.
- `reflect-allow`: package path patterns that may import `reflect`.
- `enable` / `disable`: rule toggles. Defaults follow the severity
  column in 002.

There are no inline suppression comments. The `SAFETY:` and `PANICS:`
comments are justifications with content, not switches. A finding that
a project cannot fix and cannot justify indicates a wrong rule; file an
issue instead.

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
