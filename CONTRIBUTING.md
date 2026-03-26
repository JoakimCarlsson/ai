# Contributing

## Development Setup

```bash
git clone https://github.com/joakimcarlsson/ai.git
cd ai
make install  # installs dev tools (goimports, golangci-lint, golines, air)
make fmt      # format code
make lint     # run vet + linter
make test     # run unit tests
```

## Project Structure

This repository uses a Go multi-module layout:

- **Root module:** `github.com/joakimcarlsson/ai`
- **Integration sub-modules** (each has its own `go.mod`):
  - `integrations/postgres`
  - `integrations/sqlite`
  - `integrations/pgvector`

Integration `go.mod` files contain a `replace` directive pointing to `../..`
for local development. This directive is ignored when the module is consumed
as a dependency (per Go module spec), so it does not affect consumers.

## Versioning

Each module is versioned independently using path-prefixed git tags. The tag
prefix **must** match the subdirectory path exactly — this is how the Go module
system resolves versions.

| Module | Tag format | Example |
|--------|-----------|---------|
| Root | `vX.Y.Z` | `v0.15.0` |
| postgres | `integrations/postgres/vX.Y.Z` | `integrations/postgres/v0.1.0` |
| sqlite | `integrations/sqlite/vX.Y.Z` | `integrations/sqlite/v1.1.0` |
| pgvector | `integrations/pgvector/vX.Y.Z` | `integrations/pgvector/v0.1.0` |

All modules follow [semantic versioning](https://semver.org). The root module
and integration modules are versioned independently.

## Release Process

Releases follow the AWS SDK v2 pattern: CI on main is the safety net, git tags
drive `go get` resolution, and dated GitHub Releases provide changelogs.

### 1. Ensure main is green

CI must pass on the latest commit before tagging.

### 2. Tag modules that changed

```bash
# Tag a single module (dry-run — creates local tag only)
scripts/release.sh tag -m postgres -v v0.1.0

# Tag and push
make release-tag MODULE=postgres VERSION=v0.1.0
```

For integration modules, the `require` version for the root module in
`integrations/<name>/go.mod` must match the latest published root tag.
The script warns if this is stale.

### 3. Warm the Go module proxy

```bash
scripts/release.sh warm -t integrations/postgres/v0.1.0
```

This ensures the tagged version is immediately available via `go get`.

### 4. Create a dated GitHub Release

```bash
# Dry-run (shows what would be published)
scripts/release.sh release

# Publish
make release-publish
```

This creates a `release-YYYY-MM-DD` tag and a GitHub Release listing all
module versions tagged since the previous release.
