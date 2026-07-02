#!/usr/bin/env bash
# Sync intra-repo require versions to the latest published module tag.
#
# Every module requires its sibling modules by version, e.g.
#
#   require github.com/joakimcarlsson/ai/model v0.1.0
#
# Locally a replace directive masks that version (the workspace builds against
# the source on disk), so the require line can lag the newest model/vX.Y.Z tag
# for a long time without anyone noticing. But a published consumer of this
# module gets exactly the version in the require line — the stale one. This
# script rewrites each intra-repo require to the latest tag for that module so
# the published dependency graph matches what the workspace actually builds.
#
#   scripts/sync-deps.sh          # fix:   rewrite requires, then go mod tidy
#   scripts/sync-deps.sh --check  # check: report drift, exit 1 if any (CI)
#
# Module set and tag convention (<relative-dir>/vX.Y.Z) match scripts/release.sh.
# Examples and tests/ modules are excluded: they are never published, so their
# require versions do not affect any consumer.
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

check_only=false
case "${1:-}" in
	--check) check_only=true ;;
	"") ;;
	*) echo "Usage: $0 [--check]"; exit 1 ;;
esac

# latest_version <module-dir> — highest vX.Y.Z tag for that exact module, or
# empty if the module has never been tagged. The grep anchors the full path so a
# glob like "embeddings/*" cannot leak nested modules (embeddings/openai/...).
latest_version() {
	local dir="$1"
	git tag -l "${dir}/v*" |
		grep -E "^${dir}/v[0-9]+\.[0-9]+\.[0-9]+$" |
		sed "s|^${dir}/||" |
		sort -V | tail -n1
}

# internal_requires <go.mod> — print "path<TAB>version" for every
# github.com/joakimcarlsson/ai/* require with a plain vX.Y.Z version. Pseudo- and
# pre-release versions (containing '-') are replace placeholders; skip them.
internal_requires() {
	awk '
		/^require [(]/     { blk=1; next }
		blk && /^[)]/      { blk=0; next }
		blk                { line=$0 }
		/^require [^(]/     { line=$0; sub(/^require /,"",line) }
		line ~ /github\.com\/joakimcarlsson\/ai\// {
			n=split(line,f," ")
			if (f[1] ~ /^github\.com\/joakimcarlsson\/ai\// &&
			    f[2] ~ /^v[0-9]+\.[0-9]+\.[0-9]+$/)
				print f[1] "\t" f[2]
			line=""
		}
	' "$1"
}

drift=0
fixed=0

while IFS= read -r mod; do
	[[ -z "$mod" ]] && continue
	gomod="${REPO_ROOT}/${mod}/go.mod"
	changed=false

	while IFS=$'\t' read -r path have; do
		[[ -z "$path" ]] && continue
		dep_dir="${path#github.com/joakimcarlsson/ai/}"
		want="$(latest_version "$dep_dir")"
		[[ -z "$want" ]] && continue      # dep not tagged yet; nothing to sync to
		[[ "$have" == "$want" ]] && continue
		drift=$((drift + 1))
		echo "DRIFT  $mod: $path  $have -> $want"
		if [[ "$check_only" == false ]]; then
			(cd "$mod" && go mod edit -require="${path}@${want}")
			changed=true
		fi
	done < <(internal_requires "$gomod")

	if [[ "$changed" == true ]]; then
		(cd "$mod" && GOWORK=off go mod tidy)
		fixed=$((fixed + 1))
	fi
done < <(scripts/release.sh modules)

if [[ "$check_only" == true ]]; then
	if [[ "$drift" -gt 0 ]]; then
		echo ""
		echo "Found $drift stale intra-repo require(s). Run: scripts/sync-deps.sh"
		exit 1
	fi
	echo "No intra-repo version drift."
else
	echo ""
	echo "Synced $drift require line(s) across $fixed module(s)."
fi
