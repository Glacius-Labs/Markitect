# Architecture

Status: source release `v0.1.0` is published and immutable from `45e0017a3d25d2abd464de934b5ef3a802732d0a`; consumer acceptance remains separate. See the [production delivery assessment](production-assessment.md). Updated 2026-09-30. [Usage](usage.md) owns exact current syntax. [Refinement decisions](refinement.md) owns proposed changes; [the plan](implementation-plan.md) owns their order.

## Purpose

Treat AI-facing mechanisms as software architecture: clear responsibility, explicit dependencies, controlled scope and predictable change. Go implements deterministic mechanics; YAML describes structure; Markdown remains the reading surface and prose body. Authoring is a core Markitect capability: modelling guidance, workflows, skills and supporting queries share its release lifecycle. Agents handle intent and semantic judgment; deterministic operations do not require a model. Portable core authoring resources are embedded in the tool; consumer workflows add local delivery policy.

```mermaid
flowchart LR
    Intent[Human intent] --> Agent[Authoring agent]
    Agent --> Sources[Canonical YAML and declared files]
    Sources --> Snapshot[Fixed Git snapshot]
    Snapshot --> Graph[Parse and resolve graph]
    Graph --> Checks[Structural checks]
    Graph --> Context[Compiled context]
    Graph --> Views[Managed views]
    Context --> Review[Bounded semantic review]
    Graph --> Impact[Old and new change impact]
    Impact --> Reuse[Eligible report reuse]
    Review --> Reuse
```

## Small domain

| Kind | Meaning |
|---|---|
| Text | Reusable prose/context without another machine role |
| Rule | Scoped requirement; only named implemented checks are executable |
| Workflow | Procedure and dependencies |
| Skill | Agent entrypoint to a procedure |
| Agent | Responsibility and supported provider settings |
| Contract | Required kind and exact symbolic input/output signature |
| Project | Area policy, imports, bindings and output configuration |

Local resource identity is `namespace/kind/name`. Namespaces and names are DNS labels. File paths determine area ownership; the longest containing area owns the resource. Area `imports` permits direct references and does not grant transitive access.

`rules` declares requirements; `uses` declares concrete dependencies; `needs` requires a Contract; `implements` promises its signature. A Project binding selects the implementation. All implementations are checked; compiled context follows the selected one. `files` includes exact ordinary UTF-8 file inputs, subject to area access.

Rules explicitly listed on any containing area apply to its descendant resources. This is path scope, including ancestor paths. A namespace name never creates inheritance; there is no resource inheritance or load-order override. Preserve this behavior until an explicit migration changes it.

## Compiler and adapters

The parser rejects unknown fields, duplicates, extra YAML documents, aliases, merge keys and unsupported tags. The graph checks types, identities, references, scopes, bindings, signatures and cycles. Generated YAML schemas assist editors; they do not replace these semantic checks.

The core has no dependency on Kubernetes, a model API, an IDE or a provider SDK. CLI orchestration, filesystem/Git access, rendering, release packaging and consumer checks are adapters. Existing Konfyra/Cockpit profile code remains in-tree compatibility code; extract an interface only when another adapter demonstrates a stable boundary.

Each output has one owner. Generic rendering supports declared native targets. During the Konfyra pilot, Markitect generates adjacent Markdown and the existing Python renderer owns provider files. It requires a passing Markitect check before rendering. A later replacement must reproduce its mappings and retirement behavior before switching ownership.

## Authoring and structural queries

Core resolution records why each relationship exists alongside the adjacency graph. Area ownership is captured when the existing longest-path rule is validated. Application queries consume this resolved model; CLI adapters do not implement another reference resolver.

`find` performs literal, case-insensitive discovery with exact optional filters and returns concise resource locations. `explain` shows direct incoming/outgoing relationships with declaration provenance, structural area ownership, ordinary file inputs and Contract implementation declarations/selection. A Project binding and an `implements` claim are distinct facts. `context` follows transitive graph edges and reports one deterministic inclusion reason per input. Queries require a valid graph and never claim semantic relevance or human authority.

`internal/authoring` embeds canonical YAML for a small Project, Rule, Text, Workflow and Skill. `authoring` parses and compiles this collection through the ordinary application pipeline. Its embedded source marker and digests distinguish immutable release guidance from a consumer Git revision. No model runtime, optional package or copied provider installation is required. The distribution archive includes the embedded assets, so a consumer-built binary has the same guidance as a standalone build from those sources.

## Fixed inputs and change

A revision resolves once to a committed Git tree, captured with paths, modes and bytes. Git reads and branch checks bind to the explicit repository and discard inherited Git repository/object/config overrides. Later working-tree edits cannot change that snapshot. Working-tree results are provisional. Controlled writes check source state and refuse unmanaged collisions; per-file writes are atomic, but there is no repository-wide transaction guarantee.

Context includes the selected entry, transitive dependencies, applicable rules, selected implementations and explicit file contents. It reports inclusion reasons and hashes. Markitect currently requires an explicit resource entry; selecting that entry from a natural-language task is the agent's responsibility.

Impact compares old and new dependency closures. Deleted edges cannot hide their former consumers. Configuration/inventory changes and unmodelled paths conservatively broaden the result. This preserves caution but can limit savings until real inputs are modelled more precisely.

`verify` materializes the fixed snapshot and runs supported profile gates there. Konfyra and Cockpit include native Go bootstrap tests when both integration files occur in that snapshot; an incomplete pair fails. Migrated Cockpit snapshots use their Go documentation checker and its tests; historical snapshots retain their compatibility gates. Execution time and output are bounded, partial gate results remain visible, and unavailable verification cannot be reported as passed. [Operations](operations.md) defines the limits and failure handling. `generic` has no extra repository gate. `check` includes generated drift, so the normal authoring order is edit, format, render, check, commit, fixed context/impact/review.

## Evidence limits

`snapshotDigest` identifies the loaded snapshot; context `digest` identifies declared relevant inputs and tool identity; `toolDigest` identifies executable bytes. None establishes that prose is true or complete.

Advisory review reuse requires matching tool/config/context and eligible impact. Markitect records an actual report and checks whether its inputs permit reuse; it neither invokes a model nor interprets a successful result from prose. Hashes do not authenticate a reviewer or hosted CI. Human acceptance is never transferred.

An undeclared semantic dependency remains a modelling gap. Ordinary links are navigation, not inferred graph edges. Structural correctness cannot prove arbitrary instructions compatible, sufficient or followed by a runtime agent.

## Standalone ownership

This repository owns tool source, schemas, generic examples and product design. Consumers own their policies and pinned integration. `internal/release` builds and validates the deterministic distribution using a fixed source snapshot. The complete bundle contains the source archive, existing flat tool lock, Go bootstrap, paired tests and a YAML release manifest binding their hashes to a source commit. `internal/app` owns installation plans, existing-pin ownership checks and controlled filesystem writes; CLI code selects inputs and reports outcomes. Release packaging has no provider or customer policy dependency.

`bundle` requires a fixed revision and matching source version. `install` verifies the complete bundle before planning or writing and refuses ambiguous existing ownership. GitHub release attestation establishes downloaded asset provenance; local hashes establish byte consistency. Neither replaces consumer validation. The low-level `package` command still creates only a source archive and flat tool lock. Reusable content packages and template initialization are proposed work, not current commands.

The API-shaped YAML envelope leaves room for a later Kubernetes adapter, but these resources and schemas are not CRDs. Runtime lifecycle, API conversion, status and reconciliation need an explicit future contract. The Go module follows the private `Glacius-Labs/Markitect` GitHub repository. The API group remains a placeholder until a controlled domain and explicit format migration are chosen.
