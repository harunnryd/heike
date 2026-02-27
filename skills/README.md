# Heike Built-In Skills

This folder contains bundled skills shipped with Heike.

## Load order and override rules

Runtime skill sources are loaded in this order:

1. `<workspace>/skills` (bundled, in-repo defaults)
2. `$HOME/.heike/skills` (user-global)
3. `$HOME/.heike/workspaces/<workspace-id>/skills` (workspace store)
4. `<workspace>/.heike/skills` (project-local)

Later sources override earlier ones by skill `name`.

## Skill format

Each skill must live in its own folder and expose `SKILL.md` with YAML frontmatter:

```markdown
---
name: my-skill
description: One sentence about trigger/use-case.
tags:
  - domain
  - operation
tools:
  - exec_command
  - apply_patch
---
# Skill Title

Operational guidance for the agent.
```

Optional metadata can be added for documentation and future runtime enrichment.

## Bundled skills

- `codebase-stats`: deterministic repository statistics grouped by extension for quick workspace assessment.
- `coder`: code implementation, refactor, debugging, tests, review.
- `openai-image-gen`: batch-generate images via OpenAI Images API and produce local gallery outputs.
- `researcher`: web and data research with source-aware synthesis.
- `sysadmin`: shell operations, diagnostics, and runtime maintenance.
- `skill-creator`: create and maintain high-quality skills.

## Built-in tools (reference)

`apply_patch`, `click`, `exec_command`, `find`, `finance`, `image_query`,
`open`, `screenshot`, `search_query`, `sports`, `time`, `view_image`,
`weather`, `write_stdin`.

Bundled skills should only reference valid tool names from this set unless you are
introducing new built-ins in runtime.
