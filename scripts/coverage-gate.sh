#!/bin/sh
# coverage-gate.sh - the statement coverage gate.
#
# Reads a Go coverage profile, drops files matching the path globs in
# scripts/coverage-exclude.txt (the reviewed allowlist), recomputes total
# statement coverage over the remaining files, and exits 1 if it is below
# the bar. COVERAGE_MIN sets the bar in percent; the default is 90.
#
# The gate fails closed: a profile with nothing left to measure is an
# error, never a pass - an over-broad exclude or a botched test run must
# not look like a pass.
#
# Scoped mode (COVERAGE_GATE_ALLOW_EMPTY=1, set only by
# coverage-gate-scoped.sh for the pre-commit tier): a profile whose files
# are ALL reviewed exclusions passes with a notice instead of failing
# closed, because a pre-commit scope can land entirely inside the exclude
# allowlist. The default - the full gate at pre-push and CI - keeps
# failing closed.
#
# Usage: scripts/coverage-gate.sh [coverage.out] [exclude-file]
set -eu

profile="${1:-coverage.out}"
excludes="${2:-$(dirname "$0")/coverage-exclude.txt}"
min="${COVERAGE_MIN:-90}"

# The bar must be a whole number of percent. Anything else (including
# awk-parsable poison like "nan", which defeats every comparison) fails
# closed here with a clear message.
case "$min" in
'' | *[!0-9]*)
	echo "coverage-gate: COVERAGE_MIN must be a whole number of percent, got '$min'" >&2
	exit 1
	;;
esac

if [ ! -f "$profile" ]; then
	echo "coverage-gate: profile '$profile' not found (run 'make test' first)" >&2
	exit 1
fi

# Exclude globs use shell-glob semantics: '*' and '?' never cross '/'.
# Each glob is anchored to the module root (profile paths are prefixed
# with the module path from go.mod), so an entry matches exactly the
# files a reviewer reading it as a relative path would expect - never a
# same-named path deeper in the tree.
module=$(awk '/^module /{print $2; exit}' go.mod)
regex=''
if [ -f "$excludes" ]; then
	while IFS= read -r line || [ -n "$line" ]; do
		# Strip inline '# reason: ...' comments and surrounding whitespace,
		# so each entry can carry its mandatory annotation on the same line.
		entry=$(printf '%s' "$line" | sed -e 's/#.*$//' -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')
		[ -n "$entry" ] || continue
		# The annotation is mandatory, not a convention: an entry without
		# a reason fails the gate instead of silently widening the exclude.
		case "$line" in
		*"# reason:"*) ;;
		*)
			echo "coverage-gate: exclude entry '$entry' lacks the mandatory '# reason:' annotation" >&2
			exit 1
			;;
		esac
		pat=$(printf '%s' "$entry" | sed -e 's/[].[^$+?(){}|\\]/\\&/g' -e 's/\*/[^\/:]*/g' -e 's/\\?/[^\/:]/g')
		if [ -z "$regex" ]; then
			regex="^${module}/${pat}:"
		else
			regex="${regex}|^${module}/${pat}:"
		fi
	done <"$excludes"
fi

# Drop the "mode:" header and the excluded files, then merge duplicate
# blocks (a block can appear once per test binary that links the package)
# and sum statements.
tail -n +2 "$profile" |
	{ if [ -n "$regex" ]; then grep -Ev "$regex" || true; else cat; fi; } |
	awk -v allow_empty="${COVERAGE_GATE_ALLOW_EMPTY:-}" -v min="$min" '
	{
		key = $1
		stmts[key] = $2
		if ($3 > hits[key]) hits[key] = $3
	}
	END {
		total = 0
		covered = 0
		for (key in stmts) {
			total += stmts[key]
			if (hits[key] > 0) covered += stmts[key]
		}
		if (total == 0) {
			if (allow_empty == "1") {
				print "coverage-gate: nothing left to measure after reviewed exclusions; passing (scoped mode only - the full gate fails this closed)"
				exit 0
			}
			print "coverage-gate: FAIL - no statements left to measure (empty profile or over-broad excludes)"
			exit 1
		}
		pct = covered * 100 / total
		printf "coverage-gate: %.1f%% statement coverage (%d/%d statements, bar %s%%, after reviewed exclusions)\n", pct, covered, total, min
		if (pct < min) {
			for (key in stmts) {
				if (hits[key] == 0) printf "coverage-gate: uncovered: %s\n", key
			}
			printf "coverage-gate: FAIL - below %s%%\n", min
			exit 1
		}
	}'
