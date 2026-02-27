---
title: Command Reference
sidebar_position: 35
---

This page documents CLI and slash-command surfaces implemented in the current runtime.

Examples below use `heike` as installed binary. If you run from source, use `./heike`.

## Global

- `heike --config <path>`
- `heike --server.log_level <debug|info|warn|error>`
- `heike --server.port <int>`

## Runtime Commands

### `heike run`

Start interactive REPL mode.

Flags:

- `--workspace`, `-w`: target Workspace ID

### `heike daemon`

Start service mode with component lifecycle management.

Flags:

- `--workspace`, `-w`: target Workspace ID
- `--force-clean-locks`: cleanup stale lock files on startup

### `heike version`

Print build metadata.

## Config Commands

### `heike config init`

Create default config at `~/.heike/config.yaml`.

### `heike config view`

Print resolved config with secret redaction.

## Provider Commands

### `heike provider login openai-codex`

Run OAuth login flow and save token to `auth.codex.token_path`.

## Policy Commands

### `heike policy show`

Show governance config and current domain allowlist.

### `heike policy set <tool> --allow`

Append tool to `governance.auto_allow`.

### `heike policy set <tool> --require-approval`

Append tool to `governance.require_approval`.

### `heike policy deny <tool>`

Shortcut to require approval for a tool.

### `heike policy require-approval <tool>`

Require explicit approval for a tool.

### `heike policy audit`

Query policy audit entries.

### `heike policy stats`

Show governance summary stats.

## Session Commands

### `heike session ls`

List session transcripts for workspace.

### `heike session reset <session_id>`

Delete transcript for one session.

## Cron Commands

### `heike cron ls`

List scheduled tasks from scheduler state.

## Skill Commands

### `heike skill ls`

List workspace skills.

Flags:

- `--output`, `-o`: `table|json|yaml`

### `heike skill show <name>`

Show parsed skill metadata and runtime tools.

### `heike skill search <query>`

Search available skills.

### `heike skill test <name>`

Validate skill syntax and loadability.

### `heike skill install <path>`

Install external skill into `./.heike/skills`.

### `heike skill uninstall <name>`

Remove installed external skill.

## Slash Commands (`heike run`)

- `/help`
- `/model <name>`
- `/clear`
- `/approve <approval_id>`
- `/deny <approval_id>`
- `/exit`

`/model <name>` persists per-session metadata.
`/clear` resets transcript history for the current session.
`/exit` is handled at the REPL layer and terminates the interactive session.
