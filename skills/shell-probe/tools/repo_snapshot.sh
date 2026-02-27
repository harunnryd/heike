#!/usr/bin/env sh
set -eu

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
workspace_root="$(CDPATH= cd -- "$script_dir/../../.." && pwd)"

entry_count="$(find "$workspace_root" -mindepth 1 -maxdepth 1 2>/dev/null | wc -l | tr -d ' ')"
timestamp="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

printf '{"status":"ok","tool":"repo_snapshot","workspace_entries":%s,"timestamp":"%s"}\n' "${entry_count:-0}" "$timestamp"
