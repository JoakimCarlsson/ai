#!/usr/bin/env bash
# Release tooling for the multi-module layout.
#
# After the modular refactor (Phase 1–8), the repo publishes ~50 independent Go
# modules. Each is tagged separately with the prefix convention Go's monorepo
# tooling expects: <relative-path>/vX.Y.Z. For example:
#
#   v1.0.0                       — n/a; root is no longer a module
#   message/v1.0.0               — Tier 0 leaf
#   llm/v1.0.0                   — Tier 1 modality interface
#   llm/openai/v1.0.0            — Tier 2 vendor sub-module
#   agent/memory/pgvector/v1.0.0 — Tier 5 integration
#
# Modules are discovered dynamically from go.mod files at runtime — no
# hardcoded list to keep stale.
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
REPO_URL="$(git remote get-url origin | sed 's/\.git$//' | sed 's|git@github.com:|https://github.com/|')"

usage() {
	cat <<-'EOF'
		Usage: scripts/release.sh <command> [options]

		Commands:
		  modules   List every Go module in the workspace (one per line)
		  tag       Tag a module release
		  release   Create a dated GitHub Release aggregating recent module tags
		  warm      Warm the Go module proxy for a tag

		Options for 'tag':
		  -m, --module <path>     Module dir relative to repo root (e.g. llm/openai)
		  -v, --version <semver>  Version to tag (e.g. v0.1.0)
		  --push                  Push the tag to origin (default: local only)

		Options for 'release':
		  -d, --date <YYYY-MM-DD> Date for the release (default: today)
		  --publish               Create the GitHub Release (default: dry-run)

		Options for 'warm':
		  -t, --tag <tag>         Git tag to warm (e.g. llm/openai/v0.1.0)
	EOF
	exit 1
}

# discover_modules — print every module dir relative to repo root.
discover_modules() {
	cd "$REPO_ROOT"
	# All go.mod files under the workspace; strip the trailing /go.mod.
	# Skip vendored/cache/etc. by anchoring under tracked directories.
	find . -name 'go.mod' -not -path './.git/*' -print0 |
		xargs -0 -n1 dirname |
		sed 's|^\./||' |
		grep -v '^\.$' || true
}

# module_path — print "github.com/joakimcarlsson/ai/<dir>" for a module dir.
module_path() {
	local dir="$1"
	echo "github.com/joakimcarlsson/ai/${dir}"
}

# validate_module — fail unless dir is a real module under the workspace.
validate_module() {
	local mod="$1"
	local found=false
	while IFS= read -r m; do
		[[ "$m" == "$mod" ]] && { found=true; break; }
	done < <(discover_modules)
	if [[ "$found" != true ]]; then
		echo "Error: '$mod' is not a workspace module. Run: $0 modules"
		exit 1
	fi
}

validate_semver() {
	local ver="$1"
	if [[ ! "$ver" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
		echo "Error: version '$ver' does not match semver (vX.Y.Z)"
		exit 1
	fi
}

cmd_modules() {
	discover_modules | sort
}

cmd_tag() {
	local module="" version="" push=false

	while [[ $# -gt 0 ]]; do
		case "$1" in
		-m | --module) module="$2"; shift 2 ;;
		-v | --version) version="$2"; shift 2 ;;
		--push) push=true; shift ;;
		*) echo "Unknown option: $1"; usage ;;
		esac
	done

	[[ -z "$module" ]] && { echo "Error: --module is required"; usage; }
	[[ -z "$version" ]] && { echo "Error: --version is required"; usage; }

	validate_module "$module"
	validate_semver "$version"

	if [[ -n "$(git status --porcelain)" ]]; then
		echo "Error: working tree is not clean"
		exit 1
	fi

	local branch
	branch="$(git rev-parse --abbrev-ref HEAD)"
	if [[ "$branch" != "main" ]]; then
		echo "Error: must be on main branch (currently on '$branch')"
		exit 1
	fi

	# Confirm that every internal require in this module's go.mod points at
	# either an already-tagged version or another workspace module that's
	# being tagged in this same release. We don't enforce — only warn — since
	# the release coordinator may be tagging a multi-module bundle.
	local gomod="${REPO_ROOT}/${module}/go.mod"
	local internal_reqs
	internal_reqs=$(awk '/^require [(]/,/^[)]/' "$gomod" |
		grep -oP 'github\.com/joakimcarlsson/ai/\S+' | sort -u || true)
	for dep in $internal_reqs; do
		local dep_dir="${dep#github.com/joakimcarlsson/ai/}"
		if ! git tag -l "${dep_dir}/v*" | grep -q .; then
			echo "Warning: $module requires $dep but no $dep_dir/v* tag exists yet."
		fi
	done

	local full_tag="${module}/${version}"
	if git rev-parse "$full_tag" >/dev/null 2>&1; then
		echo "Error: tag '$full_tag' already exists"
		exit 1
	fi

	git tag "$full_tag"
	echo "Created tag: $full_tag"

	if [[ "$push" == true ]]; then
		git push origin "$full_tag"
		echo "Pushed tag: $full_tag"
		echo ""
		echo "Warm the proxy with: scripts/release.sh warm -t $full_tag"
	else
		echo ""
		echo "To push: git push origin $full_tag"
	fi
}

cmd_release() {
	local date="" publish=false

	while [[ $# -gt 0 ]]; do
		case "$1" in
		-d | --date) date="$2"; shift 2 ;;
		--publish) publish=true; shift ;;
		*) echo "Unknown option: $1"; usage ;;
		esac
	done

	[[ -z "$date" ]] && date="$(date +%Y-%m-%d)"

	local release_tag="release-${date}"
	local title="Release (${date})"

	local prev_release
	prev_release=$(git tag -l 'release-*' --sort=-creatordate | head -n1)

	is_new_tag() {
		local tag="$1"
		if [[ -n "$prev_release" ]]; then
			git merge-base --is-ancestor "$prev_release" "$tag" 2>/dev/null && \
				! git merge-base --is-ancestor "$tag" "$prev_release" 2>/dev/null
		else
			return 0
		fi
	}

	local body="## Module Highlights"$'\n'
	local found_tags=false

	# Iterate every workspace module and find new <module>/v* tags since
	# the previous release.
	while IFS= read -r mod; do
		[[ -z "$mod" ]] && continue
		local mod_tags
		mod_tags=$(git tag -l "${mod}/v[0-9]*" --sort=-version:refname --merged HEAD)
		for tag in $mod_tags; do
			if is_new_tag "$tag"; then
				local ver="${tag##*/}"
				body+="* \`github.com/joakimcarlsson/ai/${mod}\`: [${ver}](${REPO_URL}/tree/${tag})"$'\n'
				found_tags=true
			fi
		done
	done < <(discover_modules | sort)

	if [[ "$found_tags" == false ]]; then
		echo "Warning: no module tags found since last release"
	fi

	if [[ "$publish" == true ]]; then
		git tag "$release_tag"
		git push origin "$release_tag"
		gh release create "$release_tag" --title "$title" --notes "$body"
		echo "Created release: $title"
	else
		echo "=== DRY RUN ==="
		echo ""
		echo "Tag:   $release_tag"
		echo "Title: $title"
		echo ""
		echo "$body"
		echo ""
		echo "Run with --publish to create the release."
	fi
}

cmd_warm() {
	local tag=""

	while [[ $# -gt 0 ]]; do
		case "$1" in
		-t | --tag) tag="$2"; shift 2 ;;
		*) echo "Unknown option: $1"; usage ;;
		esac
	done

	[[ -z "$tag" ]] && { echo "Error: --tag is required"; usage; }

	# Convert "<dir>/<version>" tag back to module path + version.
	# Tag format is always <relative-dir>/vX.Y.Z.
	if [[ ! "$tag" =~ ^(.+)/(v[0-9]+\.[0-9]+\.[0-9]+)$ ]]; then
		echo "Error: tag '$tag' does not match <module>/vX.Y.Z"
		exit 1
	fi
	local mod_dir="${BASH_REMATCH[1]}"
	local version="${BASH_REMATCH[2]}"
	local mod_path
	mod_path="$(module_path "$mod_dir")"

	echo "Warming proxy for ${mod_path}@${version}..."
	GOPROXY=proxy.golang.org go list -m "${mod_path}@${version}"
	echo "Done."
}

[[ $# -eq 0 ]] && usage

case "$1" in
modules) shift; cmd_modules "$@" ;;
tag) shift; cmd_tag "$@" ;;
release) shift; cmd_release "$@" ;;
warm) shift; cmd_warm "$@" ;;
*) usage ;;
esac
