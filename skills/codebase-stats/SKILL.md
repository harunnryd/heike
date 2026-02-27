---
name: "codebase-stats"
description: "Analyze workspace structure and return deterministic codebase statistics."
tags:
  - codebase
  - analysis
  - metrics
  - filesystem
tools:
  - "exec_command"
metadata:
  heike:
    icon: "📊"
    category: "analysis"
    kind: "runtime"
---

# Codebase Stats

Use this skill when you need a quick structural snapshot of a repository before planning refactors or reviews.

## Custom tool

- `codebase_stats`: scans the workspace and returns file/line stats grouped by extension.

## Usage

Call `codebase_stats` with optional filters (depth, hidden file handling, extension count limit).
