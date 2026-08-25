# Contributing

Thank you for the interest. Three documents carry the working rules of
this repository:

- [AGENTS.md](AGENTS.md) states the gate, the test-first rule, and the
  house style. Read it before your first commit, and run `make setup`
  once per clone.
- [REVIEW.md](REVIEW.md) states the adversarial review that every
  substantive change gets before its pull request opens.
- [docs/spec](docs/spec) holds the rule catalogue. An analyzer and its
  spec entry change together, in one pull request.

A new rule starts as a measurement, not as code. The spec records the
evidence behind every rule, and it records the rules the measurements
rejected. A proposal that adds a rule brings the same kind of evidence:
where the pattern occurs, what the fix costs, and what an existing
linter already covers.

Every change arrives through a pull request with green CI. The `main`
branch takes no direct push.
