#!/bin/sh
# coverage-gate-test.sh - self-test for the load-bearing gate script.
#
# The coverage gate is the repo's primary quality control; a fail-open bug
# in it is invisible everywhere else, so the gate itself is tested on every
# `make check` run. Each case builds a synthetic profile + exclude list and
# asserts the gate's exit code. Cases pin COVERAGE_MIN so the assertions
# hold whatever bar the Makefile sets.
set -eu

gate="$(dirname "$0")/coverage-gate.sh"
module=$(awk '/^module /{print $2; exit}' go.mod)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

fails=0

expect() {
	desc="$1" want="$2" min="$3" profile="$4" excludes="$5"
	if COVERAGE_MIN="$min" "$gate" "$profile" "$excludes" >/dev/null 2>&1; then got=0; else got=1; fi
	if [ "$got" -ne "$want" ]; then
		echo "coverage-gate-test: FAIL: $desc (exit $got, want $want)" >&2
		fails=$((fails + 1))
	fi
}

: >"$tmp/none.txt"

# Fully covered profile passes at any bar.
cat >"$tmp/covered.out" <<EOF
mode: atomic
$module/analyzers/a/a.go:1.1,2.2 3 1
$module/analyzers/b/b.go:1.1,2.2 2 5
EOF
expect "fully covered passes at 100" 0 100 "$tmp/covered.out" "$tmp/none.txt"

# 60% coverage (3 of 5 statements) sits between the bars.
cat >"$tmp/partial.out" <<EOF
mode: atomic
$module/analyzers/a/a.go:1.1,2.2 3 1
$module/analyzers/b/b.go:1.1,2.2 2 0
EOF
expect "60% fails a 90 bar" 1 90 "$tmp/partial.out" "$tmp/none.txt"
expect "60% passes a 50 bar" 0 50 "$tmp/partial.out" "$tmp/none.txt"
expect "an uncovered statement fails a 100 bar" 1 100 "$tmp/partial.out" "$tmp/none.txt"

# An exact exclude removes only the listed file.
echo "analyzers/b/b.go # reason: test fixture" >"$tmp/exact.txt"
expect "excluded file is ignored" 0 90 "$tmp/partial.out" "$tmp/exact.txt"

# '*' must not cross directory boundaries (shell-glob semantics).
cat >"$tmp/nested.out" <<EOF
mode: atomic
$module/analyzers/a/a.go:1.1,2.2 3 1
$module/analyzers/deep/nested/logic.go:1.1,2.2 30 0
EOF
echo "analyzers/deep/*.go # reason: test fixture" >"$tmp/star.txt"
expect "glob does not cross directories" 1 90 "$tmp/nested.out" "$tmp/star.txt"

# Globs are anchored at the module root, not matched as path suffixes.
cat >"$tmp/suffix.out" <<EOF
mode: atomic
$module/foo/cmd/x/main.go:1.1,2.2 40 0
$module/analyzers/a/a.go:1.1,2.2 3 1
EOF
echo "cmd/x/main.go # reason: test fixture" >"$tmp/anchor.txt"
expect "exclude does not match deeper same-named path" 1 90 "$tmp/suffix.out" "$tmp/anchor.txt"

# Nothing left to measure fails closed.
echo "analyzers/* # reason: test fixture" >"$tmp/all.txt"
cat >"$tmp/only.out" <<EOF
mode: atomic
$module/analyzers/a.go:1.1,2.2 3 1
EOF
expect "empty remainder fails closed" 1 90 "$tmp/only.out" "$tmp/all.txt"

# An allowlist entry on a final line without a trailing newline still applies.
printf 'analyzers/b/b.go # reason: test fixture' >"$tmp/no-newline.txt"
expect "exclude without trailing newline applies" 0 90 "$tmp/partial.out" "$tmp/no-newline.txt"

# '?' matches exactly one character and never crosses '/'.
echo "analyzers/b/?.go # reason: test fixture" >"$tmp/qmark.txt"
expect "question-mark glob excludes a one-char name" 0 90 "$tmp/partial.out" "$tmp/qmark.txt"
echo "analyzers?b/b.go # reason: test fixture" >"$tmp/qslash.txt"
expect "question-mark glob does not cross directories" 1 90 "$tmp/partial.out" "$tmp/qslash.txt"

# An exclude entry without the mandatory '# reason:' annotation fails the
# gate, even on a fully covered profile.
echo "analyzers/b/b.go" >"$tmp/bare.txt"
expect "exclude entry without a reason fails" 1 90 "$tmp/covered.out" "$tmp/bare.txt"

# A non-numeric bar fails closed, even on a fully covered profile. "nan"
# is the dangerous case: awk parses it and every comparison against it is
# false, so without validation the bar would vanish.
expect "garbage COVERAGE_MIN fails closed" 1 abc "$tmp/covered.out" "$tmp/none.txt"
expect "NaN COVERAGE_MIN fails closed" 1 nan "$tmp/partial.out" "$tmp/none.txt"

# Header-only profile fails closed.
printf 'mode: atomic\n' >"$tmp/header.out"
expect "header-only profile fails closed" 1 90 "$tmp/header.out" "$tmp/none.txt"

# Scoped mode (COVERAGE_GATE_ALLOW_EMPTY, set only by coverage-gate-scoped.sh):
# an all-excluded remainder passes with a notice, but a below-bar profile
# still fails - the escape hatch must never weaken the bar itself.
expect_env() {
	desc="$1" want="$2" min="$3" profile="$4" excludes="$5"
	if COVERAGE_GATE_ALLOW_EMPTY=1 COVERAGE_MIN="$min" "$gate" "$profile" "$excludes" >/dev/null 2>&1; then got=0; else got=1; fi
	if [ "$got" -ne "$want" ]; then
		echo "coverage-gate-test: FAIL: $desc (exit $got, want $want)" >&2
		fails=$((fails + 1))
	fi
}
expect_env "scoped mode passes an all-excluded remainder" 0 90 "$tmp/only.out" "$tmp/all.txt"
expect_env "scoped mode still fails a below-bar profile" 1 90 "$tmp/partial.out" "$tmp/none.txt"

# The scoped wrapper: filters the profile to the given packages' files, then
# applies the same bar. Out-of-scope lines are dropped; in-scope below-bar
# coverage fails; a scope with no coverage data passes with a notice.
scoped="$(dirname "$0")/coverage-gate-scoped.sh"
expect_scoped() {
	desc="$1" want="$2" min="$3" profile="$4" excludes="$5"
	shift 5
	if COVERAGE_GATE_EXCLUDES="$excludes" COVERAGE_MIN="$min" sh "$scoped" "$profile" "$@" >/dev/null 2>&1; then got=0; else got=1; fi
	if [ "$got" -ne "$want" ]; then
		echo "coverage-gate-test: FAIL: $desc (exit $got, want $want)" >&2
		fails=$((fails + 1))
	fi
}
expect_scoped "scoped: out-of-scope uncovered file is dropped" 0 90 "$tmp/partial.out" "$tmp/none.txt" \
	"$module/analyzers/a"
expect_scoped "scoped: in-scope below-bar coverage fails" 1 90 "$tmp/partial.out" "$tmp/none.txt" \
	"$module/analyzers/a" "$module/analyzers/b"
expect_scoped "scoped: no coverage data for scope passes with notice" 0 90 "$tmp/partial.out" "$tmp/none.txt" \
	"$module/analyzers/nosuchpkg"
expect_scoped "scoped: fully excluded scope passes with notice" 0 90 "$tmp/partial.out" "$tmp/exact.txt" \
	"$module/analyzers/b"

if [ "$fails" -gt 0 ]; then
	echo "coverage-gate-test: $fails case(s) failed" >&2
	exit 1
fi
echo "coverage-gate-test: all cases pass"
