# Contributing

Thank you for considering contributing to the Go AI Client Library!

## Development Setup

```bash
git clone https://github.com/joakimcarlsson/ai.git
cd ai
make install  # installs dev tools (goimports, golangci-lint, golines, air)
make fmt      # format code
make lint     # run vet + linter
make test     # run unit tests
```

## Contributing Workflow

1. Fork the repository and clone your fork
2. Create a feature branch from `main` (`git checkout -b feature/my-feature`)
3. Make your changes
4. Run `make fmt && make lint` to ensure formatting and linting pass
5. Run `make test` to ensure all tests pass
6. Commit your changes using [conventional commits](#commit-conventions)
7. Push to your fork and open a pull request against `main`

## Pull Request Guidelines

- Keep PRs focused on a single change
- Provide a clear description of what the PR does and why
- Ensure CI passes (build, unit tests, integration tests, formatting, linting)
- Link related issues in the PR description

## Code Style

- Do not add comments to code
- Formatting is enforced by `goimports` and `golines -m 80` (80 char line limit)
- Linting is enforced by `golangci-lint` (see [.golangci.yml](.golangci.yml) for the full config)
- Run `make fmt && make lint` before committing

## Testing

- **Unit tests:** `make test` (runs `go test -short ./...`)
- **Integration tests:** `go test -v -timeout 180s ./tests/...`
- Write tests for new functionality
- Ensure existing tests continue to pass

## Commit Conventions

This project uses [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` new feature
- `fix:` bug fix
- `chore:` maintenance, dependency updates, etc.
- `docs:` documentation changes
- `refactor:` code restructuring without behavior change
- `test:` adding or updating tests

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

## Working with Sub-modules

When making changes that affect integration sub-modules, run `go mod tidy` in each affected module directory:

```bash
cd integrations/postgres && go mod tidy && cd ../..
cd integrations/sqlite && go mod tidy && cd ../..
cd integrations/pgvector && go mod tidy && cd ../..
```

The `replace` directives in each sub-module's `go.mod` ensure they resolve the root module locally during development, so you don't need to publish a new version to test changes.
