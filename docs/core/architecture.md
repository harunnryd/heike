---
title: Architecture
sidebar_position: 10
---

The Heike runtime is deterministic by design.

## High-Level Flow

```mermaid
flowchart LR
    A[Input Sources] --> B[Ingress]
    B --> C[Interactive Queue]
    B --> D[Background Queue]
    C --> E[Worker]
    D --> F[Worker]
    E --> G[Orchestrator Kernel]
    F --> G
    G --> H[Task Manager]
    H --> I[Cognitive Engine]
    I --> J[Tool Runner]
    J --> K[Policy Engine]
    J --> L[Tool Registry]
    G --> M[Egress]
    B --> N[Store Worker]
```

## Request Call Chain

1. `ingress.Submit`
2. `worker.processEvent`
3. `orchestrator.Kernel.Execute`
4. `task.Manager.HandleRequest`
5. `cognitive.Engine.Run`
6. `planner -> thinker -> actor -> reflector`

## Complex Task Path

1. Heuristic gate (`ShouldDecompose`)
2. LLM decomposition (`Decompose`)
3. DAG execution (`Coordinator.ExecuteDAG`)
4. Shared cognitive engine per sub-task, each with isolated context

## Invariants

- Tool execution only via `tool.Runner`
- Policy checks before every tool execution
- Session lock per `session_id`
- Workspace single-writer lock across processes
