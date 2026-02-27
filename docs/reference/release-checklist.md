---
title: Release Checklist
sidebar_position: 32
---

A candidate is release-ready when:

1. `./scripts/ci/agents_guard.sh` passes
2. `go test ./...` passes
3. `go build ./cmd/heike` passes
4. `go vet ./...` passes
5. interactive smoke passes
6. daemon `/health` + graceful shutdown passes

Operational checks:

- Approval flow (`/approve`, `/deny`)
- Tool runner + policy enforcement
- Workspace lock behavior

Repository health checks:

- Legal/community files exist (`LICENSE`, `SECURITY.md`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`)
- Release assets exist (`.goreleaser.yaml`, `Dockerfile`, `Dockerfile.goreleaser`)
- GitHub templates exist (`.github/ISSUE_TEMPLATE/*`, `.github/pull_request_template.md`, `.github/CODEOWNERS`)

Container and packaging checks:

- Local Docker build succeeds (`docker build -t heike:local .`)
- Daemon health check works from container
- Release workflow and GoReleaser config remain aligned

Related page:

- [Container and Release](container-and-release.md)
