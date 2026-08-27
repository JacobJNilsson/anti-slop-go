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
  internal/          // machinery that more than one rule needs
  docs/spec/
```

- `antislop.go` exports `func Analyzers() []*analysis.Analyzer`.
- `cmd/antislop` wraps the list in
  `golang.org/x/tools/go/analysis/multichecker`.
- `plugin` holds the `register.Plugin` call. It stands apart, so a
  consumer of `cmd/antislop` or of `Analyzers()` never compiles the
  golangci-lint dependency.
- `internal/signature` holds the machinery that more than one rule
  needs: the generated-file test that every rule applies, the
  justification comment contract, the signature tests that G03, G04,
  G06, and G09 share, and the type-parameter walk that G05, G06, and
  G09 share. The sections "The generated-file exemption" and "The
  justification comment contract" below state those two contracts. The
  interface scan of the signature tests has two entry points.
  `NewContracts` reads the imported packages, and
  `NewContractsWithHome` reads the package under analysis as well. G09
  takes the second one, because a method cannot narrow a result that a
  local interface declares. 002 states both stances with the rules that
  hold them. One implementation of a contract cannot drift from itself.
- `internal/pathmatch` holds the package path patterns that the
  settings use. The Configuration section states their syntax.
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

A rule that reads a setting takes it in two ways, because the three
consumption paths differ.

**A constructor**, for a program that holds Go values. A configurable
rule exports `New`, which takes the setting and returns a new
`*analysis.Analyzer`. The package-level `Analyzer` value is `New` with
no setting. Every instance carries its own configuration, so two
callers never share one.

**A flag**, for `cmd/antislop` and for `go vet -vettool`. Both paths
read no configuration file. `New` registers the flag of the rule on the
instance it builds, and the flag writes into the configuration of that
instance. `-noreflect.allow`, `-noadhoctypeswitch.boundary`,
`-fullstructcomp.min`, `-fullstructcomp.maxignore`, and
`-errsemantics.equality` are the flags today, and the multichecker
registers each one because it registers the analyzer. The first two
take a comma-separated list, and a repeated flag adds patterns. The
next two take one number each, and the last takes a boolean.

The settings block is the configuration surface of the plugin.
golangci-lint gives the block to `New` of the plugin through
`register.DecodeSettings`, and it never sets `analysis.Analyzer.Flags`,
so a flag has no route in from a `.golangci.yml` file. `BuildAnalyzers`
therefore calls the constructor of each configurable rule with the
decoded setting, and it returns the instances it built. It never writes
to the package-level analyzer values, which every consumer of the
module shares.

The decoder rejects an unknown key. A key therefore appears in the
settings only when a rule reads it, and a key that ships stays.

### Package path patterns

Every setting that names packages takes path patterns, and
`internal/pathmatch` holds the one implementation of them. A pattern
reads against the import path of the package:

- A pattern matches the whole path. A pattern is therefore no prefix,
  no infix, and no suffix of a longer path: `codec` matches the package
  whose path is `codec`, and never `example.com/app/codec`.
- `*` matches any run of characters that holds no slash. The run may be
  empty. `*/internal/codec` matches `app/internal/codec`, and it does
  not match `example.com/app/internal/codec`.
- `...` matches any run of characters, slashes included. The run may be
  empty. `.../internal/codec` matches both paths above.
- A pattern that ends in `/...` matches the package above that suffix
  as well, which is the rule of the go command. `example.com/app/...`
  matches `example.com/app` and every package under it.
- Every other character matches itself. The dot of a domain name is a
  dot and no wildcard.

A rule drops a trailing `_test` from the path before it reads the
patterns, so one entry covers a package and its external test package.
An entry therefore names the production package path. An entry that
names an external test package, such as `example.com/app/codec_test`,
matches nothing and reports no error, because no path reaches the
comparison with that suffix. An entry that names a production package
also allows a real package that lives in a directory named `foo_test`.
002 states both directions with the rule that needs them.

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
            - ".../internal/ingest"
            - "example.com/app/api/..."
          reflect-allow:            # G07
            - "example.com/app/internal/codec"
          fullstructcomp-min: 3     # G12: fields before a report
          fullstructcomp-maxignore: 5   # G12: ignore names a fix may need
          errsemantics-equality: true   # G13
          disable:
            - noerrorassert         # when staticcheck covers it
          enable:
            - nointerfacereturn     # opt-in rules
            - justifypanic
            - fullstructcomp
            - errsemantics
```

Every key of the example exists today: `boundary-packages`,
`reflect-allow`, `fullstructcomp-min`, `fullstructcomp-maxignore`,
`errsemantics-equality`, `enable`, and `disable`.
`nointerfacereturn` (G09), `justifypanic`
(G11), `fullstructcomp` (G12), and `errsemantics` (G13) are the opt-in
rules that `enable` names.

Semantics:

- `boundary-packages`: package path patterns where decode-time
  dynamic checks are legal. G06 accepts every type switch of such a
  package, and `-noadhoctypeswitch.boundary` takes the same patterns
  on the standalone path. G02 needs no entry, because it never flags a
  local variable.
- `reflect-allow`: package path patterns that may import `reflect`.
  G07 reads it, and `-noreflect.allow` takes the same patterns on the
  standalone path.
- `fullstructcomp-min`: the number of distinct fields of one value that
  a report of G12 needs. `-fullstructcomp.min` takes the same number on
  the standalone path. A configuration with no such key gets the
  default of the rule, which is 2. The key therefore takes a pointer in
  the settings structure. The decoder writes zero for an absent key of
  an integer field, and zero is a setting of its own.
- `fullstructcomp-maxignore`: the number of `cmpopts.IgnoreFields`
  names that the comparison of a report of G12 may need. G12 counts
  those names. It reports no group above this number, because the
  ignore list of such a fix states more than the assertions it
  replaces. `-fullstructcomp.maxignore` takes the same number on the
  standalone path. The default is 5, and the key takes a pointer for
  the reason above: zero is a setting of its own.
- `errsemantics-equality`: a boolean. G13 reads it, and it adds the
  equality forms of that rule, which are a comparison of an error
  message against a string. The default is false, because a package
  that tests its own message text writes those forms and the Go wiki
  page TestComments accepts them. `-errsemantics.equality` is the same
  setting on the standalone path. The setting changes nothing until
  `enable` names G13, because the rule is opt-in.
- `enable` / `disable`: rule toggles. Defaults follow the severity
  column in 002. `disable` drops a rule from the default set. `enable`
  turns on an opt-in rule, so it rejects a rule that is on by default.
  Both settings reject a name that is not a rule, and the run stops. A
  configuration that disables every rule is legal; the linter then
  reports nothing.

### Opt-in rules and the three paths

The module holds one registry. `antislop.Analyzers()` returns every
rule, an opt-in rule included, and the three consumption paths read
that one list. The opt-in severity of 002 takes effect in the plugin,
because the plugin is the path that reads a configuration file.
`optInRules` in `plugin/plugin.go` names such a rule, and
`BuildAnalyzers` drops it until `enable` names it.

`nointerfacereturn` (G09), `justifypanic` (G11), `fullstructcomp`
(G12), and `errsemantics` (G13) are the opt-in rules today.

The other two paths read no configuration file, so they run every rule.
Their switch is the flag that `multichecker` and `go vet` give to each
analyzer. A test with `cmd/antislop` and with Go 1.26.2 verified both
directions. `-nointerfacereturn=false` drops one rule and keeps the
rest. `-nointerfacereturn` alone selects that rule and drops the rest,
which is the form the measurement scans use. `go vet
-vettool=./antislop -nointerfacereturn=false ./...` gave the same
result, because `go vet` passes an unknown flag to the tool.

The alternative was a second registry function, such as
`DefaultAnalyzers` beside `OptInAnalyzers`. The project rejected it. A
consumer that ranges over `Analyzers()` must see every rule the module
holds. A rule that the list drops disappears from `go vet`, with no
diagnostic and no setting to bring it back. Two lists also split the
definition of the rule set across two packages, and the plugin already
owns the settings.
`antislop_test.go` pins the registry membership of the opt-in rule for
that reason.

A test with golangci-lint v2.10.1 verified this path end to end. The
project held two packages, and each one imported `reflect`.
`reflect-allow` named one of them, and the run reported the other one
only. A subtree pattern that ended in `/...` gave the same result, and
so did `example.com/*/internal/codec/...`, which names the allowed
package through a wildcard prefix.

A second test verified `boundary-packages` the same way. The project
held two packages, and each one held a type switch on an `any` value.
The setting named `example.com/proj/internal/ingest`, and the run
reported the other package only. `example.com/*/internal/...` gave the
same result, and a run with no `boundary-packages` key reported both.
The run stopped with an unknown-field error when the settings held a
key that no rule reads.

This project adds no inline suppression comments. The `SAFETY:`,
`PANICS:`, and `CONTRACT:` comments are justifications with content,
not switches. A finding that a project cannot fix and cannot justify
indicates a wrong rule; file an issue instead.

One caveat, recorded after a test with golangci-lint v2.10.1: on the
golangci-lint path, `//nolint:antislop` suppresses a finding. That
mechanism belongs to golangci-lint, and a plugin cannot turn it off.
The `go vet -vettool` path has no such directive. A project that wants
a reason on every directive enables the `nolintlint` linter.

## The generated-file exemption

Every rule skips a generated file. A rule states the shape of
hand-written code. A report against a file that a program writes has no
reader who can act on it.

`internal/signature.GeneratedFiles` holds the one implementation.
`Justifications` and `Contracts` expose that test as their `Generated`
method. A rule therefore takes one answer, whichever entry point it
uses.

**The accepted header forms.** The first test is `go/ast.IsGenerated`.
It accepts the canonical header `// Code generated ... DO NOT EDIT.` of
https://go.dev/s/generatedcode. The second test below reads a wider set
of marker strings, and it accepts the canonical header as well. The
first test therefore exempts no file on its own. It stays because
`go/ast` owns the canonical form. A later change to the marker strings
then cannot drop that form by accident.

Generators exist that write no canonical header. The second test
therefore reads the header in lower case and looks for one of four
strings:

- `code generated`
- `do not edit`
- `autogenerated file`, which easyjson writes
- `* generated by: swagger codegen `, which Swagger Codegen writes in a
  block comment

golangci-lint holds the same four strings in the lax mode of its
generated-file matcher, and the module adopts them with no change. The
generated marker text is therefore the same in both tools. The window
is not the same, and the paragraphs below state the difference. The
golangci-lint default is `strict` and not `lax`. The two tools
therefore agree only where a project selects the lax mode.

A string matches at any point of the header text. A header sentence
that holds `do not edit` therefore exempts the file. That result is the
behaviour of golangci-lint, and the module keeps it.

**The window.** A comment counts only above the package clause. That
window is the window of `go/ast.IsGenerated`, and the wider marker set
reads the same window. `ast.CommentGroup.Text` supplies the text, so
the build directives drop out of it. golangci-lint reads one comment
more: its window holds the first comment below the package clause as
well. The module is narrower here on purpose.

**Why the module reads no line below the package clause.** staticcheck
and revive scan every line of a file for the canonical header. That
practice widens the window, and it raises the chance to hide a true
report. A `do not edit` comment beside one statement of a function says
nothing about the file. The module would then drop every report of that
file on the strength of one line. A generator states its authorship in
the header, so the header is the only place the module reads.

**The surface the wider marker set leaves.** A hand-written file whose
header holds one of the four strings goes silent, and every rule then
skips it. The module accepts that surface. The four strings are the
evidence a generator leaves, and no test can separate that evidence
from a sentence that quotes it.

The measurement below sizes the surface. It also rejects one repair
that looks obvious. A test that reads the first comment group alone
would end the exemption of the hand-written files below. It would end
the exemption of files that `cgo -godefs` writes as well, because that
generator puts its generated marker below a build constraint. The same
generator puts the generated marker in the first group in other files
of the same repository. The number of the group therefore states
nothing about the writer of a file.

**Measurement.** The corpus is every `.go` file of the module cache of
this machine and of `GOROOT/src`, with no directory named `testdata`.
That corpus holds 96,624 files, and 96,623 of them parse. The one file
that fails to parse counts in no bucket below. The method reads the
header of each file and applies both tests.

- 11,664 files carry the canonical header. The wider marker set
  adds 517.
- 472 of the 517 hold the generated marker in the first comment group,
  and 45 hold it below. Of those 45, a program writes 39. `cgo -godefs`
  writes 19 of the 39, and `protoc-gen-go` of 2016 writes 20.
- 511 of the 517 are files that a program writes. The generators are
  `cgo -godefs`, `protoc-gen-go` of 2016, `gotmpl`, `go run gen.go`,
  Swagger Codegen, easyjson, mockery, `stringer`, and others.
- 6 of the 517 are hand-written. They are 4 distinct files, because the
  module cache holds `runtime/cgo/cgo.go` three times. The rate is 6 of
  the 96,624 paths, and those paths hold 4 distinct files.
- 3 of those 4 hold the generated marker inside a package doc that
  describes generated code: `runtime/cgo/cgo.go` of the standard
  library, and `proto/lib.go` and `gogoproto/doc.go` of gogo/protobuf.
  The fourth is `base.go` of golangci/golines, where the author wrote
  `PLEASE DO NOT EDIT.` above code that the file borrows.
- The largest single loss is `proto/lib.go` of gogo/protobuf. It
  reports 24 findings before the change and none after it.

The change also has a cost on the repositories that motivated it. Two
work repositories report 382 findings before the change and 376 after
it. All 6 findings that stop sit in 3 files of an in-house identifier
generator. The Go standard library reports 4,880 findings before and
4,880 after, and the two lists are the same byte for byte.

## The justification comment contract

Three markers share one contract: `SAFETY:` (G01), `PANICS:` (G11),
and `CONTRACT:` (G03, G04, and G09).

- The comment sits directly above the flagged statement, or above the
  statement that contains it.
- `CONTRACT:` justifies a signature. It sits directly above the line
  where the signature starts. Where the signature starts on a later
  line, it sits above the statement or the variable declaration that
  holds it.
- The comment owns its line. A comment beside code justifies the code
  beside it, never the line below.
- The marker starts a line of the comment text. It matches the regular
  expression `(?m)^[\s*]*<MARKER>\s*:`, where `<MARKER>` is one of
  `SAFETY`, `PANICS`, and `CONTRACT`.
- The marker may start any line of a comment group, not the first line
  only.
- A marker inside a sentence counts for nothing, and so does a marker
  with a prefix. `NOT-SAFETY:` is no justification.
- The text after the marker must state the invariant, the reason to
  stop the process, or the API that sets the signature. The analyzer
  cannot judge the text; review must.

The input of the expression is the text of `ast.CommentGroup.Text`.
That method removes the comment markers and the first space of a line
comment. It keeps the rest of the leading text of a line. A block
comment may start with a space, and a gutter of stars stays. The class
before the marker accepts both.

One implementation enforces this contract for every rule.
`internal/signature.NewJustifications` takes the marker word and builds
the expression. The `Justifications` value it returns runs the position
tests. A rule supplies its marker word and its candidate lines, and
nothing else.

One implementation states the placement rule as well.
`internal/signature.EnclosingStmtLines` walks the syntax stack and
returns the line of the innermost statement, and the line of the
outermost statement below the enclosing block. It stops at a block, and
it holds the rule of a case clause and of a communication clause. G01
and G11 both read it, so the two rules cannot drift apart.

A rule adds the candidate lines of its own shape, and each one is
listed here:

- G01 adds the line of the `.(` token of the assertion. An operand that
  spans lines pushes that token below the line where the assertion
  starts.
- G11 adds nothing. A call starts where it starts.

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
5. **M5**: opt-in rules `fullstructcomp` (G12), with the
   `fullstructcomp-min` and `fullstructcomp-maxignore` settings, and
   `errsemantics` (G13), with the `errsemantics-equality` setting.

## Open questions

- G05 scope: settled for now. The analyzer that shipped is
  single-function, flow-insensitive, and binding-aware. It reports a
  widening in the operand of an assertion. It also reports a widening
  into a local variable, when a later assertion in the same function
  takes the type back. It builds no SSA. It drops a binding that a
  function literal reads, and a binding whose address escapes. An SSA
  pass would add three things. The order of the statements separates a
  widening that reaches the assertion from one that does not. The
  value graph follows a value through a struct field, a slice, a map,
  or a channel. The call graph follows a value across functions. The
  first measurement is in 002. The whole standard library, tests
  included, gives 3 findings, and four x repositories give 2. Measure
  again on a project that decodes a lot of input, before that work
  starts.
- G06 sealed-interface guidance: settled. The message names the three
  fixes, because the reader of a diagnostic needs the shape of the
  repair. 002 records the decision with the rule.
- Generics: G03 accepts constrained type parameters. Decide whether an
  unconstrained `[T any]` parameter used only for pass-through counts
  as evidence or as laundering. G05 answered the question for itself
  only: it reads no generic function, because Go demands the widening
  there. The widening of a type parameter is settled for G05 and G06
  together. `internal/signature.MentionsTypeParam` holds the walk, both
  rules read it, and `switch any(v).(type)` on a parameterized value is
  clean in both. The scan of the standard library gives 3 of them.
