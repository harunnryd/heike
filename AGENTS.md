# AGENTS.md

This file is the machine-facing contract for AI coding agents working on Heike.

## 1) Scope and Precedence

1. This `AGENTS.md` applies to the whole repository.
2. If a deeper path contains another `AGENTS.md`, the deeper file overrides this one for that subtree.
3. Direct user instruction overrides this file.
4. Deterministic runtime and policy invariants always take priority over style preferences.

## 2) Project Identity

Heike is a deterministic AI runtime with:

- policy-gated tool execution
- workspace isolation
- dual runtime modes (`run` and `daemon`)
- hierarchical orchestration for complex tasks

Primary language: Go (`go 1.25`).

## 3) Mandatory Agent Workflow

Before making claims or edits:

1. Read relevant files first (never answer from assumptions).
2. Trace the runtime call chain impact (entrypoint -> wiring -> internal package).
3. Implement minimal coherent changes, not speculative rewrites.
4. Run required verification commands.
5. Report concrete evidence: changed files + commands executed + residual risks.

## 4) Local Commands

Default verification set:

```bash
go test ./...
go build ./cmd/heike
go vet ./...
```

Quick smoke:

```bash
heike config init
heike run
# or
heike daemon
curl -fsS http://127.0.0.1:8080/health
```

If running from local build directory, replace `heike` with `./heike`.

## 5) Repository Map

- `cmd/heike/*`: CLI surfaces (`run`, `daemon`, `config`, `skill`, `policy`, etc.)
- `cmd/heike/runtime/*`: runtime composition and initializers
- `internal/orchestrator/*`: kernel, command/task managers, DAG coordinator
- `internal/cognitive/*`: plan-think-act-reflect loop
- `internal/tool/*`: tool registry, validation, runner, built-ins
- `internal/policy/*`: governance and approvals
- `internal/store/*`: persistence and workspace locking
- `internal/daemon/*`: component lifecycle orchestration
- `skills/*`: built-in skills
- `docs/*`: product and engineering documentation

## 6) Runtime Invariants (Must Not Be Broken)

1. All tool calls go through `tool.Runner`.
2. Policy checks happen before every tool execution.
3. Tool names are exact; no alias/canonical/legacy naming layer in orchestration.
4. Deterministic cognitive flow is preserved (`plan -> think -> act -> reflect`).
5. Session/workspace locking guarantees single-writer behavior.
6. Interactive and background lanes remain separated.
7. Complex-task orchestration must preserve dependency order for DAG execution.

## 7) Configuration Rules

1. Prefer YAML/config-driven behavior over hardcoded constants.
2. If a key exists in `config.yaml`, treat it as source of truth.
3. Defaults are allowed only as centralized fallback when config is absent.
4. No silent fallback for critical config: fail fast with explicit errors.
5. New config must be wired end-to-end:
   - `cmd/heike/templates/config.yaml`
   - `internal/config/*` parsing/defaults
   - runtime consumption
   - docs update

Reference template: `cmd/heike/templates/config.yaml`.

## 8) Tooling Rules

1. Built-in tools must be wired through the unified registry/runner path.
2. New built-in tool names must match runtime API names directly.
3. Migrate call sites instead of adding compatibility wrappers.
4. Remove dead legacy pathways when refactoring tool APIs.
5. Tool input/output contracts must stay deterministic and JSON-safe.

Current built-in tools include:

- `apply_patch`
- `click`
- `exec_command`
- `finance`
- `find`
- `image_query`
- `open`
- `screenshot`
- `search_query`
- `sports`
- `time`
- `view_image`
- `weather`
- `write_stdin`

## 9) Skills Rules

1. A skill is defined by `SKILL.md`.
2. Keep `SKILL.md` format consistent across built-in skills.
3. Prefer metadata-driven contracts over implicit behavior.
4. If a skill requires runtime tools, wire them deterministically in runtime tool registry.

## 10) Coding and Refactor Rules

1. Prefer small, explicit functions and predictable state transitions.
2. Avoid hidden fallbacks, implicit magic, and dead code.
3. Keep package boundaries clean (no cross-layer leakage).
4. Do not introduce TODO/FIXME placeholders in production paths.
5. For architectural cleanup, migrate call sites fully instead of leaving dual paths.
6. Keep `run` and `daemon` behavior parity when changing runtime wiring.

## 11) Testing Requirements by Change Type

1. Tooling change:
   - run tool package tests and relevant orchestrator tests.
2. Orchestrator/cognitive change:
   - run affected unit tests + end-to-end cognitive loop tests.
3. Config change:
   - run `heike config init` and `heike config view` smoke flow.
4. CLI command change:
   - validate command help/output path manually or with tests.

Minimum required before finalize:

```bash
go test ./...
go build ./cmd/heike
go vet ./...
```

## 12) Documentation Contract

When behavior or contracts change, update docs in the same change-set:

- architecture/cognitive changes -> `docs/core/*`
- tool/runtime changes -> `docs/tools/*`
- command/config changes -> `docs/reference/*`

## 13) CI Enforcement

`scripts/ci/agents_guard.sh` enforces core repository guardrails:

1. `AGENTS.md` presence and required sections.
2. Built-in tool inventory in docs matches code registration.
3. No TODO/FIXME/XXX/ANCHOR markers in production Go paths.
4. Key referenced docs/files exist.

## 14) Final Response Contract (for AI Agents)

When finishing a task, include:

1. What changed (files and behavior).
2. What verification was run (exact commands).
3. What could not be verified.
4. Any residual risk or follow-up migration needed.

## 15) References

- `README.md`
- `docs/intro.md`
- `docs/core/architecture.md`
- `docs/core/cognitive-contract.md`
- `docs/reference/configuration.md`
- `docs/reference/command-reference.md`
- `docs/reference/testing.md`
