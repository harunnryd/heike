---
name: skill-creator
description: Use for designing, creating, auditing, and refactoring Heike skills (metadata, guidance, and tool mapping).
tags:
  - skill
  - skill-design
  - agent-capabilities
  - prompt-engineering
  - tooling
tools:
  - "exec_command"
  - "apply_patch"
  - "open"
  - "search_query"
metadata:
  heike:
    icon: "🧩"
    category: "platform"
    kind: "guidance"
---

# Skill Creator

Create skills that are small, clear, and production-usable.

## Workflow
1. Define trigger/use-case in one sentence.
2. Select a short, stable skill name (lowercase, digits, hyphen, underscore).
3. Create `skills/<name>/SKILL.md` with complete frontmatter: `name`, `description`, `tags`, `tools`, `metadata` (`metadata.heike.kind` is required: `guidance` or `runtime`).
4. If `metadata.heike.kind` is `runtime`, add `skills/<name>/tools/tools.yaml` and executable scripts in `tools/`.
5. Write body guidance as executable workflow, not theory.
6. Keep references to tool names exact and current.
7. Validate by loading skills and running related tests.

## Content rules
- Put only essential instructions in `SKILL.md`.
- Use short steps with clear decisions and expected outputs.
- Avoid duplicate tool entries.
- Prefer deterministic procedures over open-ended advice.

## Tool mapping rules
- Prefer `exec_command` for exploration and verification.
- Prefer `apply_patch` for precise file edits.
- Use `open` and `search_query` only when external references are required.
- Do not list tools that are not registered in built-in runtime.

## Definition of done
- Skill loads without parser errors.
- Metadata is specific enough for `GetRelevant` matching.
- Guidance is concise, operational, and testable.
