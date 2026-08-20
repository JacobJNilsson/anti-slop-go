#!/bin/sh
# coverage-gate-scoped.sh - the scoped variant of the coverage gate.
#
# Filters a coverage profile down to the files of the given packages (the
# pre-commit scope), then applies the SAME bar via coverage-gate.sh. Two
# departures from the full gate, both safe because the full gate still
# runs unchanged at pre-push and in CI:
#   - profile lines for files outside the scoped packages are dropped
#     before measuring;
#   - a scope with nothing left to measure - packages with no test files,
#     or only reviewed-excluded files - PASSES with a printed notice
#     instead of failing closed.
#
# COVERAGE_GATE_EXCLUDES overrides the exclude list (used by the self-test).
#
# Usage: coverage-gate-scoped.sh coverage.out pkg [pkg...]
set -eu

profile="$1"
shift
if [ $# -eq 0 ]; then
	echo "coverage-gate-scoped: no packages given" >&2
	exit 1
fi
if [ ! -f "$profile" ]; then
	echo "coverage-gate-scoped: profile '$profile' not found" >&2
	exit 1
fi

tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT

head -n 1 "$profile" >"$tmp"
tail -n +2 "$profile" | awk -v pkgs="$*" '
	BEGIN { n = split(pkgs, p, " "); for (i = 1; i <= n; i++) scoped[p[i]] = 1 }
	{
		path = $1
		sub(/:.*$/, "", path)      # <import path>/<file>.go
		sub(/\/[^\/]*$/, "", path) # -> <import path> (the package dir)
		if (path in scoped) print
	}' >>"$tmp"

if [ "$(wc -l <"$tmp")" -le 1 ]; then
	echo "coverage-gate-scoped: no coverage data for the scoped packages (no test files in scope); the full gate at pre-push/CI is the backstop"
	exit 0
fi

COVERAGE_GATE_ALLOW_EMPTY=1 export COVERAGE_GATE_ALLOW_EMPTY
sh "$(dirname "$0")/coverage-gate.sh" "$tmp" ${COVERAGE_GATE_EXCLUDES:+"$COVERAGE_GATE_EXCLUDES"}
