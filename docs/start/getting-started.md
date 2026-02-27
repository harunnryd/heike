---
title: Get Started
sidebar_position: 2
---

## Prerequisites

- Go 1.25+
- One model provider credential (for live model calls)

## Installation Paths

1. Install from script: [Install](install.md)
2. Build from source: [Quickstart](quickstart.md)

## Runtime Modes

- `heike run`: interactive REPL
- `heike daemon`: service mode + `/health`

## First Milestone

1. Initialize config: `heike config init`
2. Set provider key (example): `export OPENAI_API_KEY="..."`
3. Run interactive mode: `heike run`
4. Validate response loop and tool usage

If you built from source and did not install the binary to `PATH`, use `./heike` instead of `heike`.

## Before First Push

Review repository baseline files and checks:

- [Container and Release](../reference/container-and-release.md)
- [Repository Readiness](../reference/repository-readiness.md)

For architectural deep dives after setup:

- [Domain Deep Dives](../domains/overview.md)
