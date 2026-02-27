#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT_DIR}"

fail() {
  echo "[agents-guard] ERROR: $*" >&2
  exit 1
}

info() {
  echo "[agents-guard] $*"
}

has_rg() {
  command -v rg >/dev/null 2>&1
}

require_file() {
  local path="$1"
  [[ -f "${path}" ]] || fail "required file missing: ${path}"
}

require_file "AGENTS.md"
[[ -s "AGENTS.md" ]] || fail "AGENTS.md exists but is empty"

required_headings=(
  "## 1) Scope and Precedence"
  "## 6) Runtime Invariants (Must Not Be Broken)"
  "## 7) Configuration Rules"
  "## 8) Tooling Rules"
  "## 11) Testing Requirements by Change Type"
  "## 13) CI Enforcement"
)

for heading in "${required_headings[@]}"; do
  grep -Fq "${heading}" AGENTS.md || fail "AGENTS.md missing required heading: ${heading}"
done

required_refs=(
  "README.md"
  "docs/intro.md"
  "docs/core/architecture.md"
  "docs/core/cognitive-contract.md"
  "docs/reference/configuration.md"
  "docs/reference/command-reference.md"
  "docs/reference/testing.md"
  "cmd/heike/templates/config.yaml"
)

for ref in "${required_refs[@]}"; do
  require_file "${ref}"
done

if has_rg; then
  code_builtins="$(
    rg -o --no-filename 'RegisterBuiltin\("[a-z0-9_]+"' internal/tool/builtin \
      | sed -E 's/RegisterBuiltin\("([a-z0-9_]+)"/\1/' \
      | sort -u
  )"
else
  code_builtins="$(
    grep -RhoE 'RegisterBuiltin\("[a-z0-9_]+"' internal/tool/builtin \
      | sed -E 's/RegisterBuiltin\("([a-z0-9_]+)"/\1/' \
      | sort -u
  )"
fi

agents_builtins="$(
  awk '
    /^Current built-in tools include:/ { in_list=1; next }
    in_list && /^## / { in_list=0 }
    in_list && /^- `/ { print }
  ' AGENTS.md \
    | sed -E 's/^- `([^`]+)`.*/\1/' \
    | sort -u
)"

[[ -n "${code_builtins}" ]] || fail "failed to parse built-in tools from internal/tool/builtin"
[[ -n "${agents_builtins}" ]] || fail "failed to parse built-in tools from AGENTS.md"

if ! diff -u <(printf "%s\n" "${code_builtins}") <(printf "%s\n" "${agents_builtins}") >/tmp/heike-agents-builtins.diff; then
  echo "[agents-guard] Built-in tool inventory mismatch between code and AGENTS.md" >&2
  cat /tmp/heike-agents-builtins.diff >&2
  fail "sync AGENTS.md built-in tool list with code registration"
fi

if has_rg; then
  forbidden_markers="$(
    rg -n --glob '!**/*_test.go' 'TODO|FIXME|XXX|ANCHOR' cmd internal || true
  )"
else
  forbidden_markers="$(
    grep -RInE --exclude='*_test.go' 'TODO|FIXME|XXX|ANCHOR' cmd internal || true
  )"
fi
if [[ -n "${forbidden_markers}" ]]; then
  echo "[agents-guard] Found forbidden markers in production paths:" >&2
  echo "${forbidden_markers}" >&2
  fail "remove TODO/FIXME/XXX/ANCHOR markers from cmd/ or internal/"
fi

info "AGENTS contract checks passed"
