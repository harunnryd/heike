---
title: Skill System
sidebar_position: 22
---

## Skill Layout

```text
<skill>/
  SKILL.md
  tools/
    tools.yaml
    scripts...
```

## Resolution Order

1. `<workspace_path>/skills`
2. `$HOME/.heike/skills`
3. `$HOME/.heike/workspaces/<workspace-id>/skills`
4. `<workspace_path>/.heike/skills`

Later sources override earlier ones for duplicate names.

## Runtime Discovery

1. Parse `tools/tools.yaml` if present.
2. If `tools/tools.yaml` is absent, scan scripts in `tools/`.
3. Register runtime adapters through `internal/tooling.Build`.
