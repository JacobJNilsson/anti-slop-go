# anti-slop-go

Opinionated `go/analysis` rules that reject low-evidence Go patterns.

This project is a Go companion to [dmmulroy/anti-slop](https://github.com/dmmulroy/anti-slop).
The upstream project targets TypeScript and JavaScript through Oxlint.
This project applies the same philosophy to Go.

## Status

Specification phase. No analyzers exist yet.
Read the specification in [`docs/spec`](docs/spec):

1. [Overview](docs/spec/001-overview.md) — philosophy, goals, and scope.
2. [Rules](docs/spec/002-rules.md) — the rule catalogue with examples.
3. [Implementation](docs/spec/003-implementation.md) — architecture, distribution, and configuration.

## The idea in one paragraph

Code generators produce code that compiles but carries no evidence.
A type assertion with no stated invariant, an `any` parameter, or a
`map[string]any` field moves a proof obligation from the author to the
reader. These rules reject such patterns. The author must decode input
at its I/O boundary, keep concrete types inside the program, and write
a `// SAFETY:` justification where an assertion is the correct tool.

## License

MIT. See [LICENSE](LICENSE).
