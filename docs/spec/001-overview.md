# 001: Overview

## Problem

AI code generators produce Go code that compiles but carries no evidence.
The generator does not know an invariant, so it launders the type instead.
Typical moves:

- A panicking type assertion `v.(T)` with no stated reason.
- An `any` parameter where a domain type exists.
- A `map[string]any` where a struct describes the data.
- A pass through `reflect` where a method call works.
- A test that swaps a package-level function variable instead of a seam.

Each move transfers a proof obligation from the author to the reader.
The reader cannot tell a checked invariant from a guess.

## Philosophy

The rules enforce one discipline: **evidence at the boundary, concrete
types inside**.

1. Decode external input at its I/O boundary into named domain types.
2. Do not widen a known type and then narrow it again.
3. Where an assertion is the correct tool, state the invariant in a
   `// SAFETY:` comment directly above it.
4. Where a dynamic check is the correct tool, put it in one named
   function with a clear contract.

The rules do not measure style, comment tone, or commit hygiene.
Other tools cover that ground. See "Related work" below.

## Goals

- A small set of high-signal rules. Every finding must be actionable.
- Zero duplication of existing Go linters. This project composes with
  them; it does not replace them.
- Standard tooling: each rule is a `go/analysis` analyzer.
- One-line adoption through `golangci-lint`.

## Non-goals

- Detection of AI authorship. The rules judge the code, not its author.
- Surface-pattern checks (narrative comments, hedge words, TODO tone).
- Formatting, naming conventions beyond the specified rules, or metrics.
- A general strictness mode for `go vet`-class correctness bugs.

## Relation to the upstream TypeScript project

[dmmulroy/anti-slop](https://github.com/dmmulroy/anti-slop) defines the
philosophy and the `SAFETY:` convention. This project keeps both.
The rule catalogue differs because the type systems differ:

| Upstream rule | Go translation |
| --- | --- |
| `require-safety-comment-for-type-assertion` | Direct translation. See rule G01. |
| `no-unsafe-dictionary-type` | `map[string]any` and friends. See rule G02. |
| `no-unknown-parameters`, `no-object-parameters` | `any` parameters. See rule G03. |
| `no-unknown-returns` | `any` returns. See rule G04. |
| `no-widen-then-assert`, `no-chained-type-assertions` | Interface laundering. See rule G05. |
| `no-runtime-typeof` | Ad hoc type switches on `any`. See rule G06. |
| `no-reflect-apply`, `no-reflect-get` | The `reflect` package. See rule G07. |
| `no-module-mocking` | Monkey patching in tests. See rule G08. |
| `no-known-value-widening` | Interface-typed declarations of known values. See rule G09. |
| `no-shape-in-symbol-names` | Not translated. The pattern is TypeScript-specific. |
| `no-conditional-empty-object-spread` | Not translated. Go has no spread operator. |
| `no-unknown-type-aliases` | Not translated. Folded into G02-G04. |

Go also needs rules with no upstream parent. Errors are values in Go,
panics are a Go-specific escape hatch, and the Go style guides ask a
test to compare a whole structure. See rules G10, G11, and G12.

## Related work

Enable these existing linters next to this project. Do not re-implement
them here:

- `errcheck`: ignored error returns.
- `forcetypeassert`: type assertions without the comma-ok form. Rule
  G01 supersedes it with the `SAFETY:` requirement; use one, not both.
- `ireturn`: "accept interfaces, return concrete types". `ireturn`
  reports an interface result by its type, and an allowlist holds the
  types a project accepts. Rule G09 asks a narrower question: it
  reports a result only where the body builds one concrete type through
  every path. A function with two implementations behind one interface
  stays clean under G09 and reports under `ireturn`. Enable the one
  that fits the project. G09 is opt-in and needs no allowlist.
- `testifylint`: 25 checkers over the assertions of the testify
  module. Every one of them judges one call alone, so it composes with
  rule G12 and overlaps it nowhere. `bool-compare` rewrites
  `require.True(a == b)` as `require.Equal`, and G12 then counts the
  result. No checker of it counts assertions across statements, and
  none names go-cmp.
- `iface`: its `opaque` analyzer states the rule of G09 almost word for
  word. It reports "functions that return interfaces, but the actual
  returned value is always a single concrete implementation". It is off
  by default, and it is one of the five analyzers of that linter. The
  other four report other kinds of interface pollution. G09 carries two
  things that `opaque` does not. The first is the exemption for an
  interface that the package under analysis declares, which a scan
  measured at 101 findings of 378. The second is the `CONTRACT:` escape
  for a signature that an external API fixes. Enable one, not both.
- `staticcheck`, `govet`, `revive`: general correctness and idiom.
- `depguard`: a deny list of import paths. It covers the plain half of
  rule G07, which is "no package imports `reflect`". G07 keeps its own
  analyzer for the other half: an allowlist of package path patterns,
  and the exemption for `reflect.DeepEqual` in a test file. A deny list
  answers neither.

Surface-pattern "slop" tools (sloplint, grain, ai-slop-detector) solve
a different problem and compose freely with this project.
