---
title: Testing
sidebar_position: 31
---

## Required Checks

```sh
./scripts/ci/agents_guard.sh
go test ./...
go build ./cmd/heike
go vet ./...
```

For command examples below, use `heike` if installed in `PATH`, or `./heike` when running from a local build directory.

## Risk-Critical Areas

- Orchestrator + cognitive loop
- Tooling + policy boundaries
- Store locking/persistence
- Daemon lifecycle

## Smoke Checks

### Interactive

```sh
export HOME="$(mktemp -d)"
./heike config init
./heike run
```

### Daemon

```sh
export HOME="$(mktemp -d)"
./heike config init
./heike daemon --workspace smoke
curl -fsS http://127.0.0.1:8080/health
```
