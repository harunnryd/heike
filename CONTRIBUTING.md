# Contributing to Heike

Thank you for contributing.

## Development Principles

- Deterministic runtime behavior first.
- Keep policy checks in the tool execution path.
- Prefer config-driven wiring over hardcoded behavior.
- Migrate call sites fully when refactoring, avoid dead legacy paths.

## Local Setup

```sh
go version
go test ./...
go build ./cmd/heike
go vet ./...
```

## Required Checks Before PR

```sh
./scripts/ci/agents_guard.sh
go test ./...
go build ./cmd/heike
go vet ./...
```

## Docs

If behavior, config, or call chain changes, update docs in `docs/` in the same change set.

- Architecture/runtime changes: `docs/core/*`
- Tool and skill changes: `docs/tools/*`
- CLI/config/governance changes: `docs/reference/*`

## Pull Request Guidelines

1. Keep PR scope focused and coherent.
2. Describe behavior change and affected call chain.
3. Include verification commands and results.
4. Link related issues when applicable.

For detailed contributor docs, see:

- `docs/contribute/developer-guide.md`
- `docs/contribute/contributing.md`
