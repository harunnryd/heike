---
title: Runtime and CLI
sidebar_position: 11
---

## Command Surface

- `heike run`
- `heike daemon`
- `heike config`
- `heike policy`
- `heike provider`
- `heike skill`
- `heike session`
- `heike cron`
- `heike version`

## Shared Tool Bootstrap

Both runtime modes use:

- `internal/tooling.Build(workspaceID, policyEngine, workspacePath)`

This keeps built-in/custom tool parity across `run` and `daemon`.

## Run vs Daemon Wiring

```mermaid
flowchart LR
  A["heike run"] --> B["RuntimeComponents builder"]
  B --> C["Ingress"]
  B --> D["Workers"]
  B --> E["Kernel"]
  B --> F["REPL"]
  F --> C
  A2["heike daemon"] --> G["daemon.Daemon"]
  G --> H["Component graph start order"]
  H --> C
  H --> D
  H --> E
```

## `heike run`

1. Build runtime components (`cmd/heike/runtime/*`)
2. Start orchestrator + scheduler + workers
3. REPL submits events into ingress

## `heike daemon`

1. Build `daemon.Daemon`
2. Dependency-aware component lifecycle
3. Health endpoint exposure
4. Graceful reverse-order shutdown
