---
name: "shell-probe"
description: "Use for quick workspace snapshot checks via a deterministic shell custom tool."
tags:
  - shell
  - diagnostics
  - workspace
  - operations
tools:
  - "exec_command"
metadata:
  heike:
    icon: "🧪"
    category: "operations"
    kind: "runtime"
---

# Shell Probe

Run a lightweight shell-based probe to return deterministic workspace metadata.

## Custom tool

- `repo_snapshot`: returns simple workspace summary fields in JSON.

## Usage

Call `repo_snapshot` when you need a quick diagnostic pulse before doing larger actions.
