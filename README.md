<div align="center">
  <img src="assets/logo.png" alt="Heike logo" width="112" />
  <h1>Heike</h1>
  <p><strong>Deterministic AI Runtime for Production-Grade Agent Systems</strong></p>
  <p>Reproducible execution · Policy-gated tools · Workspace-safe operations</p>
</div>

<p align="center">
  <a href="#quick-start"><img src="https://img.shields.io/badge/Quick_Start-Get_Started-0ea5e9?style=for-the-badge" alt="Quick Start" /></a>
  <a href="docs/intro.md"><img src="https://img.shields.io/badge/Documentation-Read_Now-1d4ed8?style=for-the-badge" alt="Documentation" /></a>
  <a href="docs/domains/overview.md"><img src="https://img.shields.io/badge/Domain_Deep_Dives-Explore-334155?style=for-the-badge" alt="Domain Deep Dives" /></a>
  <a href="CONTRIBUTING.md"><img src="https://img.shields.io/badge/Contributing-Join-16a34a?style=for-the-badge" alt="Contributing" /></a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.25%2B-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go 1.25+" />
  <img src="https://img.shields.io/badge/runtime-deterministic-111827?style=flat-square" alt="Deterministic runtime" />
  <img src="https://img.shields.io/badge/tools-policy--gated-b91c1c?style=flat-square" alt="Policy-gated tools" />
  <img src="https://img.shields.io/badge/modes-run%20%7C%20daemon-2563eb?style=flat-square" alt="Run and daemon modes" />
  <img src="https://img.shields.io/badge/license-MIT-16a34a?style=flat-square" alt="MIT license" />
</p>

<p align="center">
  <img src="assets/banner.png" alt="Heike banner" width="920" />
</p>

<table>
  <tr>
    <td width="33%" align="center">
      <h3>Deterministic Core</h3>
      <p><code>plan -> think -> act -> reflect</code></p>
      <p>Predictable state transitions and auditable execution traces.</p>
    </td>
    <td width="33%" align="center">
      <h3>Governed Tooling</h3>
      <p><code>policy + approval + sandbox</code></p>
      <p>Tool calls are enforced by policy, not best effort prompts.</p>
    </td>
    <td width="33%" align="center">
      <h3>Production Runtime</h3>
      <p><code>run + daemon</code></p>
      <p>Shared runtime core across local REPL and long-running services.</p>
    </td>
  </tr>
</table>

## Why Heike

| Common Runtime Risk | Heike Default |
| --- | --- |
| Non-reproducible behavior across runs | Deterministic loop with explicit phase boundaries |
| Unsafe tool execution paths | Policy gate and approval workflow in the tool runner path |
| Session/workspace race conditions | Session locking and single-writer workspace model |
| Divergent local vs service behavior | Shared runtime core for interactive and daemon modes |
| Hidden behavior in runtime paths | Config-driven runtime (`config.yaml`) with explicit defaults |

> Heike is not anti LLM-first frameworks. It treats deterministic orchestration and governance as runtime invariants.

## Architecture Snapshot

```mermaid
flowchart LR
  A["Ingress"] --> B["Worker"]
  B --> C["Orchestrator Kernel"]
  C --> D["Task Manager"]
  D --> E["Cognitive Engine"]
  E --> F["Tool Runner"]
  F --> G["Policy Engine"]
  F --> H["Tool Registry"]
  C --> I["Egress"]
```

## Quick Start

1. Install and run:

```sh
curl -fsSL https://raw.githubusercontent.com/harunnryd/heike/main/install.sh | sh
heike config init
export OPENAI_API_KEY="your-key"
heike run
```

2. Build from source:

```sh
go build -o heike ./cmd/heike
./heike version
./heike run
```

3. Run as daemon:

```sh
heike daemon
curl -fsS http://127.0.0.1:8080/health
```

4. Run with Docker:

```sh
docker build -t heike:local .
docker run --rm -p 8080:8080 -e OPENAI_API_KEY="your-key" heike:local
```

Common REPL commands:

```text
/help
/approve <approval_id>
/deny <approval_id>
/clear
/exit
```

## Run Modes

| Mode | Command | Best For |
| --- | --- | --- |
| Interactive REPL | `heike run` | Local development and manual agent workflows |
| Service/Daemon | `heike daemon` | Long-running operations with health checks and scheduling |

## Tooling Highlights

| Category | Tools |
| --- | --- |
| Core execution | `exec_command`, `write_stdin`, `apply_patch` |
| Web and data | `search_query`, `open`, `click`, `find`, `weather`, `finance`, `sports`, `time`, `image_query` |
| Local interaction | `view_image`, `screenshot` |

Full contracts:

- [Built-in Tools](docs/tools/built-in-tools.md)
- [Tool Reference](docs/tools/tool-reference.md)

## Documentation Map

<table>
  <tr>
    <td width="33%" valign="top">
      <h3>Start</h3>
      <ul>
        <li><a href="docs/intro.md">Docs Intro</a></li>
        <li><a href="docs/start/getting-started.md">Get Started</a></li>
        <li><a href="docs/start/quickstart.md">Quickstart</a></li>
      </ul>
    </td>
    <td width="33%" valign="top">
      <h3>Runtime Core</h3>
      <ul>
        <li><a href="docs/core/architecture.md">Architecture</a></li>
        <li><a href="docs/core/runtime-and-cli.md">Runtime and CLI</a></li>
        <li><a href="docs/core/components.md">Components</a></li>
        <li><a href="docs/core/cognitive-contract.md">Cognitive Contract</a></li>
      </ul>
    </td>
    <td width="33%" valign="top">
      <h3>Domain Deep Dives</h3>
      <ul>
        <li><a href="docs/domains/overview.md">Domain Overview</a></li>
        <li><a href="docs/domains/event-pipeline.md">Event Pipeline Domain</a></li>
        <li><a href="docs/domains/model.md">Model Domain</a></li>
        <li><a href="docs/domains/policy-and-tool-runner.md">Policy and Tool Runner Domain</a></li>
        <li><a href="docs/domains/executor.md">Executor Domain</a></li>
        <li><a href="docs/domains/sandbox.md">Sandbox Domain</a></li>
        <li><a href="docs/domains/skill-runtime.md">Skill Runtime Domain</a></li>
      </ul>
    </td>
  </tr>
</table>

### Reference and Ops

- [Skill System](docs/tools/skill-system.md)
- [Configuration](docs/reference/configuration.md)
- [Command Reference](docs/reference/command-reference.md)
- [Governance and Approvals](docs/reference/governance-and-approvals.md)
- [Provider and Auth](docs/reference/provider-auth.md)
- [Workspace Layout](docs/reference/workspace-layout.md)
- [Testing](docs/reference/testing.md)
- [Release Checklist](docs/reference/release-checklist.md)
- [Container and Release](docs/reference/container-and-release.md)
- [Repository Readiness](docs/reference/repository-readiness.md)

## Contributing and Community

- [AGENTS.md](AGENTS.md)
- [CONTRIBUTING.md](CONTRIBUTING.md)
- [SECURITY.md](SECURITY.md)
- [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
- [SUPPORT.md](SUPPORT.md)
- [CHANGELOG.md](CHANGELOG.md)

## Release Assets

- [.goreleaser.yaml](.goreleaser.yaml)
- [Dockerfile](Dockerfile)
- [Dockerfile.goreleaser](Dockerfile.goreleaser)
- [CI Workflow](.github/workflows/ci.yaml)
- [Release Workflow](.github/workflows/release.yaml)
- [LICENSE](LICENSE)
