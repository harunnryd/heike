---
title: Quickstart
sidebar_position: 4
---

## 1. Initialize Config

```sh
heike config init
```

Default config path: `~/.heike/config.yaml`.

## 2. Set Provider Credentials

Example:

```sh
export OPENAI_API_KEY="your-key"
```

Common env vars:

- `OPENAI_API_KEY`
- `ANTHROPIC_API_KEY`
- `GEMINI_API_KEY`
- `ZAI_API_KEY`

## 3. Run Interactive Mode

```sh
heike run
```

Useful commands inside REPL:

- `/help`
- `/approve <approval_id>`
- `/deny <approval_id>`
- `/clear`
- `/exit`

## 4. Run Daemon Mode

```sh
heike daemon
curl -fsS http://127.0.0.1:8080/health
```

If you are running from a local build, replace `heike` with `./heike`.
