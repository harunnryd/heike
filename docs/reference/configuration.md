---
title: Configuration
sidebar_position: 30
---

This page documents runtime configuration in `~/.heike/config.yaml`.

## Config Lifecycle

- Initialize config: `heike config init`
- Inspect resolved config: `heike config view`
- Override via env: `HEIKE_*`

The generated template lives at `cmd/heike/templates/config.yaml`.

## Top-Level Keys

- `models`
- `server`
- `governance`
- `auth`
- `prompts`
- `store`
- `tools`
- `orchestrator`
- `ingress`
- `worker`
- `scheduler`
- `daemon`
- `adapters`

## Models

Key fields:

- `default`
- `fallback`
- `embedding`
- `max_fallback_attempts`
- `registry[]`

`registry[]` fields:

- `name`
- `provider`
- `base_url`
- `api_key`
- `auth_file`
- `request_timeout`
- `embedding_input_max_chars`

Default template models include OpenAI, Anthropic, Gemini, ZAI, Ollama, and OpenAI Codex entries.

## Governance

- `require_approval[]`: tools that require approval
- `auto_allow[]`: tools that execute directly
- `idempotency_ttl`
- `daily_tool_limit`

## Auth (OpenAI Codex)

- `auth.codex.callback_addr`
- `auth.codex.redirect_uri`
- `auth.codex.oauth_timeout`
- `auth.codex.token_path`

## Tool Runtime Config

### `tools.web`

- `base_url`
- `timeout`
- `max_content_length`

### `tools.weather`

- `base_url`
- `timeout`

### `tools.finance`

- `base_url`
- `timeout`

### `tools.sports`

- `base_url`
- `timeout`

### `tools.image_query`

- `base_url`
- `timeout`

### `tools.screenshot`

- `timeout`
- `renderer`

### `tools.apply_patch`

- `command`

## Orchestrator

- `verbose`
- `max_sub_tasks`
- `max_tools_per_turn`
- `max_turns`
- `token_budget`
- `decompose_word_threshold`
- `session_history_limit`
- `subtask_retry_max`
- `subtask_retry_backoff`

## Server and Runtime Loops

### `server`

- `port`
- `log_level`
- `read_timeout`
- `write_timeout`
- `idle_timeout`
- `shutdown_timeout`

### `ingress`

- `interactive_queue_size`
- `background_queue_size`
- `interactive_submit_timeout`
- `drain_timeout`
- `drain_poll_interval`

### `worker`

- `shutdown_timeout`

### `scheduler`

- `tick_interval`
- `shutdown_timeout`
- `lease_duration`
- `max_catchup_runs`
- `in_flight_poll_interval`
- `heartbeat_workspace_id`

### `daemon`

- `shutdown_timeout`
- `health_check_interval`
- `startup_shutdown_timeout`
- `preflight_timeout`
- `stale_lock_ttl`
- `workspace_path`

## Adapters

### `adapters.slack`

- `enabled`
- `port`
- `signing_secret`
- `bot_token`

### `adapters.telegram`

- `enabled`
- `update_timeout`
- `bot_token`

## Environment Override Pattern

Examples:

- `HEIKE_SERVER_PORT=9090`
- `HEIKE_SERVER_LOG_LEVEL=debug`
- `HEIKE_MODELS_DEFAULT=gpt-5.2-codex`
- `HEIKE_MODELS_EMBEDDING=nomic-embed-text`
- `HEIKE_GOVERNANCE_IDEMPOTENCY_TTL=48h`
- `HEIKE_ORCHESTRATOR_MAX_TOOLS_PER_TURN=8`
- `HEIKE_DAEMON_WORKSPACE_PATH=/var/lib/heike/workspaces`
- `HEIKE_AUTH_CODEX_TOKEN_PATH=/secure/heike/auth/codex.json`

Provider credential env vars:

- `OPENAI_API_KEY`
- `ANTHROPIC_API_KEY`
- `GEMINI_API_KEY`
- `ZAI_API_KEY`
