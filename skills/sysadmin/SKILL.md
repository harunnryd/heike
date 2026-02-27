---
name: "sysadmin"
description: "Use for environment diagnostics, process/runtime operations, and safe system-level maintenance via shell."
tags:
  - sysadmin
  - operations
  - shell
  - diagnostics
  - runtime
tools:
  - "exec_command"
  - "write_stdin"
  - "apply_patch"
  - "time"
metadata:
  heike:
    icon: "🛠️"
    category: "operations"
    kind: "guidance"
---
# Sysadmin

Run safe and auditable operational tasks in the local environment.

## Workflow
1. Inspect current state first (`exec_command` for logs, processes, filesystem, config).
2. Plan minimal-risk commands before execution.
3. Execute commands with `exec_command`.
4. Use `write_stdin` only for interactive sessions created with TTY mode.
5. Use `apply_patch` for deterministic config/file edits when command flags are error-prone.
6. Use `time` for timestamp correlation and incident timelines.

## Operating rules
- Prefer read-only diagnostics before write operations.
- Avoid destructive commands unless explicitly requested.
- Always report exact command outcomes and remaining risk.
