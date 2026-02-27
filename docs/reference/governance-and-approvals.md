---
title: Governance and Approvals
sidebar_position: 36
---

Heike governance is enforced in the tool runner path.

## Policy Sources

- Static config: `governance.auto_allow`, `governance.require_approval`
- Domain list: workspace `governance/domains.json`
- Approval state: workspace `governance/approvals.json`

## Decision Flow

1. Tool call enters `tool.Runner.Execute`.
2. Policy engine checks allow/approval/domain rules.
3. If approval is required, execution is blocked with an approval ID.
4. User resolves via `/approve <id>` or `/deny <id>`.

## Recommended Baseline

- Keep high-risk tools in `require_approval`:
  - `exec_command`
  - `write_stdin`
  - `apply_patch`
- Keep low/medium read tools in `auto_allow` where safe.

## Update Policy

```sh
heike policy set exec_command --require-approval
heike policy set time --allow
heike policy show
```

## Operational Notes

- `open` may require domain allowlist approval depending on policy state.
- Policy updates persist in config/governance state, not just in-memory.
- `heike policy set <tool> --allow` and `--require-approval` append entries; avoid duplicate tool names in config.
