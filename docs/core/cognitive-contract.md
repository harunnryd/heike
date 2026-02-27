---
title: Cognitive Contract
sidebar_position: 13
---

## Scope

- Interactive request path
- `plan -> think -> act -> reflect`
- Tool execution and policy mediation

## Runtime Contract

1. `AvailableTools` must be set before thinker execution.
2. Tool names must match exact registered names.
3. No alias/canonical/legacy tool-name layer is allowed.
4. All tool calls go through `tool.Runner`.
5. All runner calls are policy-checked.

## Control Signals

Reflector returns one of:

- `continue`
- `retry`
- `replan`
- `stop`

Engine exits on final answer, stop signal, max-turn budget, or context cancellation.
