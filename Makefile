# The statement coverage bar the gate enforces, in percent. One knob,
# read by scripts/coverage-gate.sh. Raise to 100 when the analyzer set
# stabilizes; see AGENTS.md.
COVERAGE_MIN ?= 90
export COVERAGE_MIN

.PHONY: build test gate-test lint lint-fast vet tidy tidy-check check check-scoped selfcheck setup clean

build:
	go build ./...

test:
	go test -race -coverprofile=coverage.out ./...
	./scripts/coverage-gate.sh coverage.out

# The gate script is load-bearing and fail-open bugs in it are invisible
# elsewhere, so it is self-tested on every check.
gate-test:
	sh scripts/coverage-gate-test.sh

lint:
	golangci-lint run ./...

# The fast lint pass the pre-commit tier uses for commits with no staged
# .go files (docs): only the linters golangci-lint marks fast.
lint-fast:
	golangci-lint run --fast-only ./...

vet:
	go vet ./...

tidy:
	go mod tidy

# check must verify, not mutate: the pre-commit hook runs it after files
# are staged, so rewriting go.mod/go.sum here would ship the stale staged
# versions. Run `make tidy` manually to apply fixes.
tidy-check:
	go mod tidy -diff

# The repo lints itself with its own analyzers. The multichecker exits
# nonzero on any finding, so slop in this codebase fails the gate.
selfcheck:
	go run ./cmd/antislop ./...

check: tidy-check vet lint gate-test test build selfcheck

# The pre-commit tier of the two-tier gate: vet/lint/-race-test only the
# packages the staged change can affect, with the SAME coverage bar on
# the scoped profile. Gate/toolchain changes escalate to the full
# `check` inside the script. Pre-push and CI always run the full
# `check`; it remains the definition of green.
check-scoped:
	sh scripts/precommit-check.sh

setup:
	git config core.hooksPath .githooks

clean:
	rm -f coverage.out coverage.scoped.out
