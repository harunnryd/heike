---
title: Provider and Auth
sidebar_position: 37
---

## Model Registry

Models are configured in `models.registry` with fields such as:

- `name`
- `provider`
- `base_url`
- `api_key`
- `auth_file`
- `request_timeout`

## Supported Provider Keys

- `OPENAI_API_KEY`
- `ANTHROPIC_API_KEY`
- `GEMINI_API_KEY`
- `ZAI_API_KEY`

## OpenAI Codex OAuth

For `provider: openai-codex`, use:

```sh
heike provider login openai-codex
```

Interactive login currently supports `openai-codex` only.

Config keys:

- `auth.codex.callback_addr`
- `auth.codex.redirect_uri`
- `auth.codex.oauth_timeout`
- `auth.codex.token_path`

## Common Failures

- Token expired or invalid
- Callback port blocked
- Model name not present in registry
- Request timeout too short for long responses

Recommended validation:

1. Confirm `models.default` or requested model exists in `models.registry`.
2. Confirm token file path matches `auth.codex.token_path`.
3. Re-run `heike provider login openai-codex` after token expiry.
