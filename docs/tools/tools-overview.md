---
title: Tools Overview
sidebar_position: 20
---

Heike has two tool classes at runtime:

1. Built-in tools (`internal/tool/builtin/*`)
2. Runtime skill tools loaded from `skills/*/tools`

## Execution Boundary

All tools execute through:

- `tool.Registry` for lookup
- `tool.Runner` for execution
- `policy.Engine` for allow/approval checks

No direct tool bypass is allowed in cognitive/orchestrator components.

## Selection Model

`task.Manager` uses broker selection before each run and exposes a bounded tool set in `AvailableTools`.
