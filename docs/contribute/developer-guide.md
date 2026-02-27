---
title: Developer Guide
sidebar_position: 50
---

## Code Map

- `cmd/heike/*`: command surfaces
- `cmd/heike/runtime/*`: runtime composition
- `internal/orchestrator/*`: kernel + command/task managers
- `internal/cognitive/*`: cognitive loop implementation
- `internal/tool/*`: tool platform
- `internal/policy/*`: governance
- `internal/store/*`: persistence + locking

## Main Call Chain

`run` mode:

1. `cmd/heike/run.go`
2. Runtime builder
3. Worker
4. Kernel
5. Task manager
6. Cognitive engine
7. Runner/policy

`daemon` mode uses the same core packages via component lifecycle orchestration.

## Engineering Rules

- Keep deterministic state transitions.
- Keep policy in the execution path.
- Avoid hidden runtime fallbacks.
- Update docs whenever behavior contracts change.
