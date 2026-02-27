---
title: Tool Reference
sidebar_position: 23
---

Built-in tools use strict exact names and JSON object input.

## Core Workspace Tools

### `exec_command`

Key input fields:

- `cmd` or `command`
- `args`
- `tty`
- `workdir`
- `shell`
- `login`
- `yield_time_ms`
- `max_output_tokens`

Example:

```json
{"cmd":"ls -la","workdir":"/path/to/repo","yield_time_ms":1000}
```

### `write_stdin`

Key input fields:

- `session_id` (required)
- `chars`
- `yield_time_ms`
- `max_output_tokens`

Example:

```json
{"session_id":123,"chars":"echo ready\\n","yield_time_ms":500}
```

### `apply_patch`

Key input fields:

- `patch` (required)
- `workdir`
- `dry_run` (currently unsupported)

Example:

```json
{"patch":"*** Begin Patch\\n*** Update File: README.md\\n@@\\n-old\\n+new\\n*** End Patch\\n"}
```

### `view_image`

Key input fields:

- `path` (absolute local path)

## Web Navigation Tools

### `search_query`

Key input fields:

- `query` or `q`
- `domains`
- `recency`
- `max_results`
- `search_query` (batch)
- `response_length`

Example:

```json
{"search_query":[{"q":"heike ai runtime"},{"q":"golang policy engine"}],"response_length":"short"}
```

### `open`

Key input fields:

- `url` or `ref_id`
- `lineno`
- `open` (batch)

Example:

```json
{"ref_id":"turn0search0","lineno":120}
```

### `click`

Key input fields:

- `ref_id`
- `id`
- `click` (batch)

### `find`

Key input fields:

- `ref_id`
- `pattern`
- `case_sensitive`
- `find` (batch)

### `screenshot`

Key input fields:

- `ref_id`
- `pageno` (0-based)
- `screenshot` (batch)

Current behavior: PDF-focused rendering.

## Live Data Tools

### `time`

Key input fields:

- `utc_offset`
- `time` (batch)

### `finance`

Key input fields:

- `ticker`
- `type` (`equity|fund|crypto|index`)
- `market`
- `finance` (batch)

### `weather`

Key input fields:

- `location`
- `start` (`YYYY-MM-DD`)
- `duration`
- `weather` (batch)

Example:

```json
{"weather":[{"location":"San Francisco, CA","duration":3}]}
```

### `sports`

Key input fields:

- `fn` (`schedule|standings`)
- `league` (`nba|wnba|nfl|nhl|mlb|epl|ncaamb|ncaawb|ipl`)
- `team`
- `opponent`
- `date_from`, `date_to`
- `num_games`
- `locale`
- `sports` (batch)

Example:

```json
{"sports":[{"fn":"schedule","league":"nba","team":"GSW","date_from":"2026-02-01"}]}
```

### `image_query`

Key input fields:

- `query` or `q`
- `domains`
- `recency`
- `limit`
- `image_query` (batch)

Example:

```json
{"image_query":[{"q":"waterfalls"},{"q":"tokyo skyline night"}]}
```

## Name Contract

Do not call dot-style aliases (for example, `search.query`).
Use exact registry names only (for example, `search_query`).
