---
title: Container and Release
sidebar_position: 33
---

This page documents how Heike is packaged and released.

## Container Build Paths

Heike provides two Dockerfiles:

- `Dockerfile`: local and CI-friendly multi-stage build from source.
- `Dockerfile.goreleaser`: minimal runtime image expecting prebuilt `heike` binary.

## Local Container Smoke

Build:

```sh
docker build -t heike:local .
```

Run daemon:

```sh
docker run --rm -p 8080:8080 -e OPENAI_API_KEY="your-key" heike:local
```

Health check:

```sh
curl -fsS http://127.0.0.1:8080/health
```

## Release Workflow

Release automation lives in:

- `.github/workflows/release.yaml`

Current flow:

1. verify job runs `./scripts/ci/agents_guard.sh` and `go test ./...`
2. build job cross-compiles (`linux|darwin` x `amd64|arm64`)
3. artifacts are archived as `heike_<os>_<arch>.tar.gz`
4. checksums are generated
5. GitHub release is published

## GoReleaser Config

GoReleaser config lives at `.goreleaser.yaml` and defines:

- build matrix
- binary name (`heike`)
- archive naming template
- checksum output

Optional local snapshot (requires GoReleaser installed):

```sh
goreleaser release --snapshot --clean
```

## Aligning Docker and Release Assets

Keep these files in sync when changing build/distribution behavior:

- `Dockerfile`
- `Dockerfile.goreleaser`
- `.goreleaser.yaml`
- `.github/workflows/ci.yaml`
- `.github/workflows/release.yaml`
