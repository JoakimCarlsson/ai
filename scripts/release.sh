#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
VALID_MODULES=("root" "postgres" "sqlite" "pgvector")

usage() {
	cat <<-'EOF'
		Usage: scripts/release.sh <command> [options]

		Commands:
		  tag       Tag a module release
		  release   Create a dated GitHub Release
		  warm      Warm the Go module proxy for a tag

		Options for 'tag':
		  -m, --module <name>     Module: root, postgres, sqlite, pgvector
		  -v, --version <semver>  Version to tag (e.g., v0.15.0)
		  --push                  Push the tag to origin (default: local only)

		Options for 'release':
		  -d, --date <YYYY-MM-DD> Date for the release (default: today)
		  --publish               Create the GitHub Release (default: dry-run)

		Options for 'warm':
		  -t, --tag <tag>         Git tag to warm (e.g., v0.15.0)
	EOF
	exit 1
}

tag_prefix() {
	case "$1" in
	root) echo "" ;;
	*) echo "integrations/$1/" ;;
	esac
}

module_path() {
	case "$1" in
	root) echo "github.com/joakimcarlsson/ai" ;;
	*) echo "github.com/joakimcarlsson/ai/integrations/$1" ;;
	esac
}

tag_to_module_path() {
	local tag="$1"
	if [[ "$tag" =~ ^integrations/([^/]+)/v ]]; then
		echo "github.com/joakimcarlsson/ai/integrations/${BASH_REMATCH[1]}"
	else
		echo "github.com/joakimcarlsson/ai"
	fi
}

tag_to_version() {
	local tag="$1"
	if [[ "$tag" =~ (v[0-9]+\.[0-9]+\.[0-9]+)$ ]]; then
		echo "${BASH_REMATCH[1]}"
	else
		echo "$tag"
	fi
}

latest_root_tag() {
	git tag -l 'v[0-9]*' --sort=-version:refname | head -n1
}

validate_module() {
	local mod="$1"
	for valid in "${VALID_MODULES[@]}"; do
		[[ "$mod" == "$valid" ]] && return 0
	done
	echo "Error: invalid module '$mod'. Valid: ${VALID_MODULES[*]}"
	exit 1
}

validate_semver() {
	local ver="$1"
	if [[ ! "$ver" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
		echo "Error: version '$ver' does not match semver (vX.Y.Z)"
		exit 1
	fi
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

	local prefix full_tag
	prefix="$(tag_prefix "$module")"
	full_tag="${prefix}${version}"

	if git rev-parse "$full_tag" >/dev/null 2>&1; then
		echo "Error: tag '$full_tag' already exists"
		exit 1
	fi

	if [[ "$module" != "root" ]]; then
		local gomod="${REPO_ROOT}/integrations/${module}/go.mod"
		local require_ver
		require_ver=$(grep 'github.com/joakimcarlsson/ai ' "$gomod" | grep -oP 'v[0-9]+\.[0-9]+\.[0-9]+')
		local latest
		latest="$(latest_root_tag)"
		if [[ -n "$latest" && "$require_ver" != "$latest" ]]; then
			echo "Warning: $gomod requires root at $require_ver but latest root tag is $latest"
			echo "         Consider updating the require version before tagging."
		fi
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

	local range
	if [[ -n "$prev_release" ]]; then
		range="${prev_release}..HEAD"
	else
		range="HEAD"
	fi

	local body="## Module versions in this release"$'\n\n'
	body+="| Module | Version |"$'\n'
	body+="|--------|---------|"$'\n'

	local found_tags=false

	local root_tags
	root_tags=$(git tag -l 'v[0-9]*' --sort=-creatordate --merged HEAD)
	for tag in $root_tags; do
		if [[ -n "$prev_release" ]]; then
			if git merge-base --is-ancestor "$prev_release" "$tag" 2>/dev/null && \
			   ! git merge-base --is-ancestor "$tag" "$prev_release" 2>/dev/null; then
				body+="| \`github.com/joakimcarlsson/ai\` | ${tag} |"$'\n'
				found_tags=true
			fi
		else
			body+="| \`github.com/joakimcarlsson/ai\` | ${tag} |"$'\n'
			found_tags=true
		fi
	done

	for mod in postgres sqlite pgvector; do
		local mod_tags
		mod_tags=$(git tag -l "integrations/${mod}/v*" --sort=-creatordate --merged HEAD)
		for tag in $mod_tags; do
			if [[ -n "$prev_release" ]]; then
				if git merge-base --is-ancestor "$prev_release" "$tag" 2>/dev/null && \
				   ! git merge-base --is-ancestor "$tag" "$prev_release" 2>/dev/null; then
					body+="| \`github.com/joakimcarlsson/ai/integrations/${mod}\` | $(tag_to_version "$tag") |"$'\n'
					found_tags=true
				fi
			else
				body+="| \`github.com/joakimcarlsson/ai/integrations/${mod}\` | $(tag_to_version "$tag") |"$'\n'
				found_tags=true
			fi
		done
	done

	if [[ "$found_tags" == false ]]; then
		echo "Warning: no module tags found since last release"
		body+="| *(none)* | |"$'\n'
	fi

	body+=$'\n'"## Changes"$'\n\n'
	if [[ -n "$prev_release" ]]; then
		body+=$(git log --oneline "$range")
	else
		body+=$(git log --oneline -20)
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

	local mod_path version
	mod_path="$(tag_to_module_path "$tag")"
	version="$(tag_to_version "$tag")"

	echo "Warming proxy for ${mod_path}@${version}..."
	GOPROXY=proxy.golang.org go list -m "${mod_path}@${version}"
	echo "Done."
}

[[ $# -eq 0 ]] && usage

case "$1" in
tag) shift; cmd_tag "$@" ;;
release) shift; cmd_release "$@" ;;
warm) shift; cmd_warm "$@" ;;
*) usage ;;
esac
