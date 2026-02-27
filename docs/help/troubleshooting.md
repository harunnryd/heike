---
title: Troubleshooting
sidebar_position: 40
---

## Tool Not Found

Check:

1. Exact tool name
2. Policy config (`heike policy show`)
3. Runtime bootstrap path (`internal/tooling.Build`)
4. Skill discovery roots

## Approval Required Loops

- Inspect pending approval prompts in session output
- Run `/approve <id>` or `/deny <id>`
- Verify that the tool is not accidentally always in `require_approval`

## Provider Failures

Common symptoms:

- `context canceled`
- Timeout errors
- Auth/token failures

Checks:

1. Provider credentials
2. Network reachability
3. Model registry config
4. Timeout settings (`models.registry[].request_timeout`)

## Docusaurus Rendering Issues

- Ensure every page has frontmatter
- Keep link paths relative and valid
- Avoid deleted target files in nav/index pages
