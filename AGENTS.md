# Working in this repository

Guidance for any agent or contributor making changes here. Read it before your first commit.

## First thing, every clone

Run `make setup` once per clone. It points git at the tracked hooks in `.githooks/`:

- `pre-commit` runs the scoped tier of the gate (`make check-scoped`) before each commit.
- `pre-push` runs the full `make check` before each push.

`core.hooksPath` is per-clone local config and is NOT version-controlled, so a fresh
clone or a new worktree has the hooks disabled until you run `make setup`. A clone
that skips this can commit and push code that fails CI. Do not skip it.

## The gate

The full `make check` is the definition of green: it must exit 0 before any push or
merge, and pre-push and CI run it unchanged (they are the enforcement points). It
runs, in order:

1. `go mod tidy -diff` (no dependency drift)
2. `go vet ./...`
3. `golangci-lint run ./...`
4. the coverage-gate self-test
5. `go test -race ./...` plus the coverage gate
6. `go build ./...`
7. `make selfcheck`: the repo's own analyzers over the repo (dogfood)

The gate is two-tier: pre-commit runs a fast local approximation (`make
check-scoped`) - vet, lint, and `-race` tests scoped to the packages the staged
change can affect, with the SAME coverage bar applied to the scoped profile.
Commits touching go.mod, go.sum, Makefile, scripts/, .githooks/, or .golangci.yml
escalate to the full `make check`; docs-only commits run tidy-check plus a fast
lint. The scoped tier never substitutes for the full gate, it only shortens the
commit loop.

The coverage gate requires the `COVERAGE_MIN` bar (90% statement coverage, set in
the Makefile) after the reviewed exclusions in `scripts/coverage-exclude.txt`.
Keep it honest: cover code with real, behavior-asserting tests. Do not pad
coverage with assertion-free tests, do not add `coverage-exclude.txt` entries to
dodge it, do not edit the gate script, and do not silence it with `t.Skip` or
pragmas. If a branch is genuinely unreachable, prove it and remove it rather than
faking a test for it. Analyzers are highly testable through `analysistest`; most
packages should sit at 100% regardless of the bar.

## Implementing analyzers

- The rule catalogue and architecture live in `docs/spec/`. An analyzer that
  drifts from its spec entry must change the spec in the same pull request, with
  the reasoning.
- One analyzer per package under `analyzers/<name>/`, registered in
  `antislop.go`, tested with `analysistest` against `testdata/src/` fixtures.
- Write the tests first. Author the fixtures and their `want` expectations
  before the analyzer, run them to see them fail, then implement until they
  pass. A test written after the code tends to describe what the code
  happens to do, not what the rule requires.
- Every rule ships with accepted AND rejected fixtures before it merges.
- Fixture code under `testdata/` is deliberately bad; keep it out of lint and
  coverage scopes (already configured).

## Design restraint

It is easy to pile rules onto a linter without taking a step back. Apply YAGNI
and KISS: before building, ask whether the rule should exist at all, whether a
smaller check exists, and whether an established linter already covers it.
A rule that fires on idiomatic Go is a bug in the rule. Flag speculative parts
in the pull request instead of shipping them silently.

## Merging

- Every substantive change gets an adversarial review after commits, before the PR
  opens - process, briefing requirements, and the review's dual mandate (code
  correctness AND solution fit) are in [REVIEW.md](REVIEW.md).
- Never merge a pull request whose CI is not green. Verify with `gh pr checks <pr>`
  before merging.
- Rebase-merge. Keep commits story-sized and conventional (see `git log` for tone).
  Squash only true fixups.
- Never push to `main` directly; every change arrives through a pull request.
- Never post comments, replies, or reviews on pull requests. Report findings in
  your run report instead.

## House style

- Never use em dashes anywhere, in code, comments, commit messages, or docs. Use hyphens.
- Write committed text (comments, commit messages, docs) in ASD-STE100 Simplified
  Technical English: active voice, short sentences, one word for one meaning.
- READMEs and docs stand alone for a reader with zero prior context.
