---
title: Workspace Layout
sidebar_position: 38
---

Heike runtime data is stored under:

- `~/.heike/workspaces/<workspace_id>/`

## Key Files

- `workspace.lock`
- `sessions/index.json`
- `sessions/<session_id>.jsonl`
- `governance/approvals.json`
- `governance/domains.json`
- `governance/processed_keys.json`
- `scheduler/tasks.json`

## Why It Matters

- The lock file enforces single-writer safety.
- Transcript files preserve role-ordered history.
- Governance files make approval and idempotency handling deterministic.
