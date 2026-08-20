#!/bin/sh
# precommit-check.sh - the pre-commit tier of the two-tier gate.
#
# The full `make check` remains the definition of green and still runs,
# unchanged, at pre-push and in CI (the enforcement points). This script
# gives the commit loop a fast LOCAL APPROXIMATION: it checks only the
# packages the staged change can affect - the changed packages plus every
# package whose compiled code or test binary depends on one of them - under
# the SAME bars: go vet, the full lint config, -race tests, and the
# coverage gate applied to the scoped profile
# (scripts/coverage-gate-scoped.sh). The bar is never weakened; only the
# blast radius examined per commit shrinks.
#
# What counts as a changed package:
#   - the package of every staged .go file;
#   - for every staged non-.go file, the nearest enclosing Go package that
#     can SEE it: the file is among the package's go:embed files, lives
#     under a testdata/ dir, or was deleted (conservative - embed lists
#     are computed from the worktree, so a deleted embed file would
#     otherwise vanish). Non-.go files no package sees (README.md,
#     docs/, ...) are docs.
#
# Escalation rules (anything that can change what every package means, or
# what the gate itself checks, gets the full gate):
#   - staged go.mod, go.sum, Makefile, scripts/, .githooks/, or
#     .golangci.yml           -> full `make check`
#   - a staged .go file whose package cannot be resolved (e.g. the package
#     was deleted)             -> full `make check`
#   - `go list ./...` failing  -> full `make check` (a broken package
#     anywhere must not silently under-scope)
#   - docs-only staged sets    -> tidy-check + fast lint
#
# Empty scope never silently passes: every path prints what it scoped and
# which tier ran.
#
# Known blind spot, accepted: the import graph cannot see cross-package DATA
# reads (a test opening ANOTHER package's testdata at run time). The full
# gate at pre-push and CI is the backstop for those.
set -eu

cd "$(git rev-parse --show-toplevel)"

# Read the staged list BEFORE scrubbing git's location variables: for partial
# commits (git commit <paths>, git commit -p) git points GIT_INDEX_FILE at a
# temporary index naming exactly the files being committed, and that is the
# right list to scope from. --no-renames keeps a rename as a delete+add pair
# so BOTH sides' packages enter the scope. NOTE: the staged list only CHOOSES
# the scope; the checks below run against the WORKTREE. A partial commit
# whose staged half differs from the worktree is enforced at head by
# pre-push and CI.
staged=$(git diff --cached --name-only --no-renames)

# Defense in depth: the checks below may run tests that shell out to git.
# A test git subprocess that inherited these variables would operate on
# THIS repository instead of its own temp repo. Unset them so nothing the
# gate runs can inherit an ambient git location.
unset GIT_DIR GIT_INDEX_FILE GIT_WORK_TREE GIT_OBJECT_DIRECTORY \
	GIT_ALTERNATE_OBJECT_DIRECTORIES GIT_COMMON_DIR GIT_NAMESPACE \
	GIT_INDEX_VERSION GIT_PREFIX

started=$(date +%s)
finish() {
	echo "pre-commit gate: $1 in $(($(date +%s) - started))s"
}

full_gate() {
	echo "pre-commit gate: $1; running the FULL gate (make check)"
	make check
	finish "full gate green"
	exit 0
}

# Tier escalation: these paths change the meaning of every package or of the
# gate itself, so the scoped approximation is not sound for them.
escalate=$(printf '%s\n' "$staged" |
	grep -E '^(go\.mod$|go\.sum$|Makefile$|\.golangci\.yml$|scripts/|\.githooks/)' || true)
if [ -n "$escalate" ]; then
	printf '%s\n' "$escalate" | sed 's/^/  staged: /'
	full_gate "staged gate/toolchain files"
fi

changed=""
add_changed() {
	case " $changed " in
	*" $1 "*) ;;
	*) changed="$changed $1" ;;
	esac
}

# Staged .go files: their package dirs, resolved to import paths. A dir that
# no longer resolves (a deleted or renamed-away package) escalates: its
# dependents need the full gate.
gofiles=$(printf '%s\n' "$staged" | grep '\.go$' || true)
if [ -n "$gofiles" ]; then
	for d in $(printf '%s\n' "$gofiles" | xargs -n1 dirname | sort -u); do
		if p=$(go list "./$d" 2>/dev/null); then
			add_changed "$p"
		else
			full_gate "cannot resolve a package for staged dir '$d'"
		fi
	done
fi

# Staged non-.go files: walk up to the nearest enclosing Go package and
# include it when that package can see the file (go:embed, testdata/, or a
# deletion). Everything else is docs and adds nothing to the scope.
otherfiles=$(printf '%s\n' "$staged" | grep -v '\.go$' || true)
if [ -n "$otherfiles" ]; then
	while IFS= read -r f; do
		[ -n "$f" ] || continue
		d=$(dirname "$f")
		pkgdir=""
		while :; do
			if go list "./$d" >/dev/null 2>&1; then
				pkgdir="$d"
				break
			fi
			[ "$d" = "." ] && break
			d=$(dirname "$d")
		done
		[ -n "$pkgdir" ] || continue
		rel="$f"
		[ "$pkgdir" != "." ] && rel=${f#"$pkgdir"/}
		include=""
		case "/$f" in */testdata/*) include=1 ;; esac
		# A deleted file cannot appear in the worktree-computed embed list,
		# so treat any staged deletion under a package as affecting it.
		[ -e "$f" ] || include=1
		if [ -z "$include" ] && go list -f \
			'{{range .EmbedFiles}}{{.}}{{"\n"}}{{end}}{{range .TestEmbedFiles}}{{.}}{{"\n"}}{{end}}{{range .XTestEmbedFiles}}{{.}}{{"\n"}}{{end}}' \
			"./$pkgdir" | grep -qxF "$rel"; then
			include=1
		fi
		if [ -n "$include" ]; then
			p=$(go list "./$pkgdir")
			echo "pre-commit gate: staged $f -> package $p"
			add_changed "$p"
		fi
	done <<EOF
$otherfiles
EOF
fi

if [ -z "$changed" ]; then
	echo "pre-commit gate: no staged file affects a Go package; staged set:"
	if [ -n "$staged" ]; then
		printf '%s\n' "$staged" | sed 's/^/  /'
	else
		echo "  (nothing staged)"
	fi
	echo "pre-commit gate: running tidy-check + fast lint (full gate at pre-push/CI)"
	make tidy-check lint-fast
	finish "docs-only scope green"
	exit 0
fi

# The scope: the changed packages plus every package whose code or tests can
# see them. {{.Deps}} is already transitive for compiled code; test imports
# are one edge deep, so a dependent's test closure is its TestImports and
# XTestImports plus each in-module test import's own transitive Deps (a test
# import outside the module can never reach back into it). A failing
# `go list ./...` must escalate, not silently under-scope.
if ! graph=$(go list -f '{{.ImportPath}}|{{join .Deps ","}}|{{join .TestImports ","}},{{join .XTestImports ","}}' ./...); then
	full_gate "go list over ./... failed (broken package outside the staged set?)"
fi
scope=$(printf '%s\n' "$graph" |
	awk -v changed="$changed" '
	function touches(p,   m, d, j) {
		if (p in hit) return 1
		m = split(deps[p], d, ",")
		for (j = 1; j <= m; j++) if (d[j] in hit) return 1
		return 0
	}
	BEGIN {
		n = split(changed, c, " ")
		for (i = 1; i <= n; i++) if (c[i] != "") hit[c[i]] = 1
	}
	{
		split($0, f, "|")
		pkg[NR] = f[1]
		deps[f[1]] = f[2]
		timps[NR] = f[3]
	}
	END {
		for (r = 1; r <= NR; r++) {
			p = pkg[r]
			if (touches(p)) { print p; continue }
			m = split(timps[r], t, ",")
			for (j = 1; j <= m; j++) {
				if (t[j] == "") continue
				if ((t[j] in deps) && touches(t[j])) { print p; break }
			}
		}
	}')

if [ -z "$scope" ]; then
	full_gate "staged files resolved to an empty scope"
fi

echo "pre-commit gate: scoped to $(printf '%s\n' "$scope" | wc -l | tr -d ' ') package(s):"
printf '%s\n' "$scope" | sed 's/^/  /'

# golangci-lint takes relative dirs, not import paths.
module=$(awk '/^module /{print $2; exit}' go.mod)
lintdirs=""
for p in $scope; do
	rel=${p#"$module"}
	rel=${rel#/}
	if [ -z "$rel" ]; then lintdirs="$lintdirs ."; else lintdirs="$lintdirs ./$rel"; fi
done

make tidy-check
go vet $scope
golangci-lint run $lintdirs
go test -race -coverprofile=coverage.scoped.out $scope
sh scripts/coverage-gate-scoped.sh coverage.scoped.out $scope
# Self-lint the scoped packages with the repo's own analyzers, so slop
# fails at commit time, not only at pre-push and CI.
go run ./cmd/antislop $scope
finish "scoped gate green (full gate still runs at pre-push/CI)"
