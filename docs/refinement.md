# Product refinement decisions

This document owns product principles and deferred direction. The [roadmap](implementation-plan.md) is the sole owner of current source and planned-work status; [Usage](usage.md) owns supported Project syntax and CLI behavior.

## Product principles

| Area | Decision |
|---|---|
| Deterministic core | One Go implementation serves people, agents, and automation. It validates structure without calling a model. |
| Core authoring | Resource-modelling guidance, workflows, skills, and supporting queries ship with Markitect. They are not optional packages. |
| Typed relationships | Model dependencies that affect execution, context, impact, or ownership. Keep ordinary documentation and navigation lightweight. |
| Explicit completeness | Compile context from an explicitly selected resource and its declared graph. Semantic task-to-resource selection remains outside the deterministic engine. |
| No implicit policy | Imports do not activate requirements or hooks. Project checks and output targets are explicit. |
| Stable identity | Local identity remains `namespace/kind/name`; namespace labels are not paths and do not imply inheritance. |
| Measured interfaces | Extend the query API only when actual authoring work shows a need. MCP, LSP, studio, and runtime reconciliation are options, not release promises. |
| One data format | Markitect-owned configuration and evidence use YAML. No telemetry or statistics subsystem is needed to ship core authoring. |

## Content packages

The v0.3.0 source model implements the first bounded content-package slice. Its user-facing contract is in [Content packages](content-packages.md). It uses exact direct pins and offline archives, keeps package resources read-only, rejects nested imports, and treats packages as context/API boundaries rather than confidentiality boundaries.

## Templates and interfaces

Minimal one-time project initialization is in the current source model. It creates only the Project file and one area README after preview and validation. It uses no custom templates, executes no scripts, and does not synchronize created projects with an evolving template. The precise contract and its structural-verification limit are in [Usage](usage.md); source status remains in the [roadmap](implementation-plan.md).

Expose graph inspection through MCP or LSP only when calls through the stable CLI/application API show measurable friction. Runtime/operator integration requires an explicit desired-state and reconciliation contract. A conceptual diagram is not a commitment to ship every interface.

## CLI distribution assessment (2026-10-01)

The public Apache-2.0 repository already publishes immutable Windows and Linux amd64 executables. A pinned release URL plus its published SHA-256 gives a direct installation path without Go, GitHub CLI, or an account; the README owns the concrete commands. `go install` is a short alternative for developers who already have the required Go toolchain. Project pinning through `markitect install` is a separate workflow.

Consider [WinGet's portable package format](https://learn.microsoft.com/en-us/windows/package-manager/winget/) as the first package-manager publication. Its community manifest must carry a versioned download URL and SHA-256 and must be validated against the actual executable. Publishing there would add a familiar `winget install` and managed upgrades, but requires a manifest update and installation test for each Markitect release. No WinGet package is published by this source change.

For Linux, retain the direct binary now. A [Homebrew tap](https://docs.brew.sh/Taps) can serve Linux, but would require users to install Homebrew and maintain another repository and formula for each release. Native `.deb`/`.rpm` assets can be considered if users need distribution-native installation; an APT repository adds signing and repository operations. The current assets do not include macOS or arm64, so a cross-platform Homebrew path would overstate support. Do not document a package-manager command until its package has been published and installation tested.

## MCP assessment (2026-10-01)

MCP could help an authoring agent discover Markitect's graph and request typed, bounded results without constructing shell commands or parsing CLI text. The existing `find`, `explain`, `context`, and `impact` operations are plausible read-only tools; embedded authoring guidance is a plausible resource. The current CLI already exposes these operations and returns structured YAML with snapshot and tool identity. A local `find` call against the minimal example returned a resource identity, canonical path, `provisional: true`, a snapshot digest, and a tool digest. MCP would add an interface, not a new source of truth or new analysis capability.

The [MCP server model](https://modelcontextprotocol.io/specification/2026-07-28/server/index) distinguishes model-invoked tools from contextual resources. The [standard transports](https://modelcontextprotocol.io/specification/2026-07-28/basic/transports) include local stdio and remote HTTP. If a pilot is justified, start with local stdio over the existing Go application API. The user selects one canonical repository root when starting the server; model-supplied paths cannot select another checkout or escape that root. Calls requiring fixed evidence must name an exact revision, and `impact` must name both base and candidate commits. Working-tree queries remain explicitly provisional. Preserve diagnostics, tool and snapshot identity, and package bytes pinned by the selected snapshot. The first pilot should have no write operations, Project-declared command execution, model calls, remote service, implicit repository selection, or claim of human acceptance. [MCP tool descriptions](https://modelcontextprotocol.io/specification/2026-07-28/server/tools) alone do not enforce these boundaries.

Do not commit an MCP server to the product roadmap on interface appeal alone. First compare the same preselected real authoring tasks and commits through the CLI and a small MCP prototype in two clients that prospective users actually use. Record successful task completion, wrong-repository or wrong-revision attempts and rejections, tool-call count, latency, and the size and usefulness of returned context. Include invalid revisions, package references, and paths in the boundary checks. Proceed only if the adapter improves the workflow without changing Markitect's deterministic results or ownership boundaries. This assessment is a pilot criterion, not evidence that MCP has already delivered a benefit.

A [local read-only stdio prototype](../experiments/mcp-pilot/README.md) now wraps `find`, `explain`, and `context` for a pinned repository and commit. One synthetic fixed-snapshot `find` smoke test produced the same YAML through MCP and the CLI. The prototype returns text blocks and uses MCP protocol version `2025-11-25`; it has not been tried in two real clients or shown to improve an authoring task. It remains outside the supported CLI binaries and product roadmap.

## Measurement

Use fixed snapshots, predeclared expected effects, correctness-first scoring, and repeated comparable tasks. Missing measurements are unavailable, not zero. Do not infer model token savings from context bytes or one successful evidence-reuse decision. [Measurement](measurement.md) owns the procedure.
