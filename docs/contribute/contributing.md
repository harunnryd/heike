---
title: Contributing
sidebar_position: 51
---

## Contribution Principles

- Deterministic runtime behavior first.
- Policy and governance boundaries are mandatory.
- No bypass paths around runner/policy.
- Fail fast on missing critical config.

## Local Workflow

```sh
./scripts/ci/agents_guard.sh
go test ./...
go build ./cmd/heike
go vet ./...
```

## PR Checklist

1. Tests and build are green.
2. No dead legacy pathways.
3. Docs are updated for behavior/call-chain changes.
4. Run/daemon parity is preserved when runtime wiring changes.
