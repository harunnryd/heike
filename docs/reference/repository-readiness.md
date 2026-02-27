---
title: Repository Readiness
sidebar_position: 34
---

This page tracks baseline repository assets required before pushing to GitHub and cutting releases.

## Legal and Governance Files

- `LICENSE`
- `AGENTS.md`
- `CONTRIBUTING.md`
- `SECURITY.md`
- `CODE_OF_CONDUCT.md`
- `SUPPORT.md`
- `CHANGELOG.md`

## Container and Release Files

- `Dockerfile` (local/container runtime build)
- `Dockerfile.goreleaser` (release-image packaging path)
- `.goreleaser.yaml` (binary/archive release config)

See details:

- [Container and Release](container-and-release.md)

## GitHub Repository Health Files

- `.github/CODEOWNERS`
- `.github/pull_request_template.md`
- `.github/ISSUE_TEMPLATE/bug_report.md`
- `.github/ISSUE_TEMPLATE/feature_request.md`
- `.github/ISSUE_TEMPLATE/config.yml`
- `.github/FUNDING.yml`
- `.github/dependabot.yml`

## Required Verification Before Push

```sh
./scripts/ci/agents_guard.sh
go test ./...
go build ./cmd/heike
go vet ./...
```

## Configuration Notes

- Update `CODEOWNERS` with real GitHub users/teams.
- Ensure issue/security links in templates point to the final repo path.
- Keep release workflows aligned with `.goreleaser.yaml` and Docker assets.
