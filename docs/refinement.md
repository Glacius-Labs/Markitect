# Refinement decisions

Status: product direction and bounded next design; 2026-09-30. This document evaluates the supplied concept notes, including the consolidated `markitect-vision-and-system-overview.md` and `markitect-benchmark-and-metrics-guide.md`. Proposed package/template APIs below are not supported RC2 syntax. [Architecture](architecture.md) describes implementation.

## Adopt and refine

| Idea | Decision | Refinement |
|---|---|---|
| AI architecture as an engineering toolchain | Adopt | One deterministic Go core serves agents, people and automation. |
| Agent authoring as primary daily UX | Adopt | Authoring is a core product capability shipped with Markitect; manual editing and deterministic CI still need no LLM. |
| Authoring using Markitect's own resources | Implemented in development | Portable core Rule, Skill, Workflow and Text ship with the tool; package support is not a prerequisite. |
| Versioned reusable engineering knowledge | Adopt next design | Identity, exports, immutable inputs, lock and evidence integration ship together. |
| Templates as project bootstrap | Adopt later | One-time ownership transfer; no ongoing synchronization or inheritance. |
| Ordinary documentation and formal dependencies | Adopt | Type relationships that affect execution, context, impact or ownership. Keep ordinary prose/README navigation lightweight. |
| Local completeness | Refine | Compile a complete relevant context from canonical sources; do not maintain copied facts. |
| No inheritance | Refine terminology | No resource overrides or namespace inheritance. Current explicit rules on parent paths still apply to descendants. |
| Natural-language authoring-context lookup | Keep at agent layer | The deterministic API takes selected identities, paths or structural filters. Semantic selection remains explicit agent reasoning. |
| MCP, LSP, graph inspection | Incremental | Expose proven CLI queries first. MCP may precede LSP if agent usage shows greater benefit. |
| Studio and Kubernetes Operator | Defer | Need measured inspection or runtime-state use cases. No separate validator, agent runtime or generic plugin framework. |
| Go throughout Markitect | Adopt now | Core, bootstrap and tests use Go. Existing consumer-owned tools remain adapters during migration. |

The user explicitly refined the new authoring document: authoring belongs to the Markitect core, rather than an optional module/package. Its workflow, modelling resources and structural queries are part of the product and release. The deterministic engine remains independent of model execution. The new product view makes the interface model useful, but its component list is an option map rather than a promise to build nine products. CLI structured output already exists as YAML; MCP is not required for an agent to use it.

## Consolidated vision and metrics

The new vision confirms the separation between deterministic structure, semantic judgment and human acceptance. Adopt the distinction between lightweight ordinary documentation and typed, machine-relevant inputs: README routes remain useful without making every Markdown link a dependency.

Refine four details against the actual implementation:

- Section 22 still makes authoring optional. The user's explicit core decision supersedes that wording. Core authoring is released with the tool and may be extended by local consumer policy.
- Namespace examples containing `/` describe organizational structure, not accepted identifiers. Namespaces remain simple DNS labels; paths and explicit area imports model organization. Adding a second identity syntax needs a demonstrated use case and migration.
- `owner` means the canonical resource path and governing area in these queries. It is not a person, role assignment, authorization or proof that prose ownership was modelled completely.
- General interface and open-source diagrams describe future options. The standalone source repository already exists; no public hosting, license, registry or Kubernetes lifecycle contract is inferred from the vision.

Adopt the benchmark guide's hidden oracle, fixed inputs, matched model settings, repetition, correctness-first scoring and explicit missing values. Start with deterministic Go scenarios and one isolated authoring exercise. A full model comparison is a later controlled experiment; no token savings are inferred from context bytes or one successful review reuse.

The suggested NDJSON/SQLite persistence is replaced by the user's single-format decision: any future Markitect-owned measurement records use YAML. Local opt-in metadata can be considered after the pilot demonstrates useful fields; no telemetry or statistics subsystem is needed to ship core authoring. [Measurement](measurement.md) owns the concrete protocol.

## Correct the proposed authoring sequence

The notes sometimes put `check` and review before rendering. Current `check` includes generated drift. Use: select owner/entry, edit sources, format, render owned views, render consumer-owned provider outputs, run checks, commit the complete candidate, calculate fixed impact/context, review affected meaning. A new semantic edit starts a new candidate. `impact` needs both base and candidate commits.

The same executable should produce context and record/reuse evidence. A review must declare every input required for its question. Deterministic checks cannot discover all omitted semantic dependencies.

## Content package design

This is the next bounded design, not a second resource type hierarchy.

1. A package is a versioned collection of the existing resource kinds. Give it a canonical identity, an exact version, explicit exported identities, immutable source coordinates and a digest. Package manifest schema comes only with a working resolver.
2. Local identity remains `namespace/kind/name`. An external reference additionally names its package. The lock selects one immutable version per canonical package identity. Reject two incompatible requested versions instead of introducing aliases, ranges or a solver initially.
3. Package imports belong to Project/package dependency configuration. Existing area imports continue to grant local cross-area access; they must not be overloaded with external dependency resolution.
4. Consumers may reference exported resources. An exported resource may need private resources inside its package; resolve those inside the package and include them in context, while rejecting direct consumer access. Exports provide API encapsulation, not confidentiality: context can necessarily contain private implementation text.
5. A consumer explicitly applies exported Rules in its own area policy and explicitly binds exported Contracts. Importing a package alone applies no rules, activates no providers and executes no hooks. Package-local references resolve against that package's namespace map, never coincidentally matching consumer names.
6. Package `files` are confined to its own verified snapshot or declared package dependencies. They cannot reach arbitrary consumer paths. The consumer supplies project-specific behavior through a Contract and binding, not an escaping path.
7. Start with one vendored immutable content archive plus manifest and digest, resolved offline. The repository snapshot includes the archive and lock; no implicit network fetch during `check`, `context` or review. Missing or mismatched bytes fail. Network acquisition is a future explicit operation.
8. Include package provenance and every required external resource/file in context identity, old/new impact and review eligibility. A lock/export/dependency change conservatively invalidates evidence until narrower analysis is justified. Cover removals and private dependencies as well as exports.
9. Initially forbid transitive package imports with a clear diagnostic. Add them only after the single-package slice works; then record the full dependency graph, reject cycles and enforce one exact version per identity. No hidden floating `latest`.
10. Treat an update as a candidate change: acquire/resolve separately, stage candidate archive+lock, compare old/new graphs, regenerate, check and review, then commit. Source configuration, lock and bytes are one reviewable change. Do not mutate the accepted lock before evaluating the candidate.

No implicit replacement or load-order override. Begin with local additions and Contract binding. A generic package must not acquire customer data merely because it is technically importable; distribution authority requires an explicit content decision. Introduce machine classifications only with a real policy and examples to test.

### Lock names and compatibility

RC2's `markitect.lock.yaml` pins the **tool**, and `markitect package` builds its **source distribution**. Neither currently means a content package. Reserve `markitect.lock.yaml` for a future versioned YAML envelope containing separate `tool` and `packages` sections. Before adding content packages, implement an explicit conversion of the current flat lock and Go bootstrap compatibility. Do not silently reinterpret existing locks or make two independent files own one dependency fact. Keep the existing command meaning until a deliberate CLI migration names distribution and dependency operations clearly.

## Core authoring

Ship a small portable core authoring collection with Markitect: resource modelling, ownership/scope, one canonical source, an author-resource-change Workflow, an entry Skill and a modelling Text. It should explain when to edit an existing owner, choose a kind, declare dependencies and interpret diagnostics. It must not embed Konfyra role assignments, Azure Boards delivery policy or customer examples.

The agent selects relevant resources from intent, using inventory and the `find`/`explain` queries. The core explains the selected graph. A proposed free-text `context --for-authoring` must not pretend deterministic completeness; defer it until its semantic selection layer and uncertainty are explicit.

Core authoring resources live in this repository and share the tool release lifecycle. Bundle them with the distribution and verify them with a synthetic consumer. Their use must not depend on installing an optional knowledge package. A future package may extend authoring with organization policy, but cannot become the owner of the core workflow.

## Templates

After package composition is proven, implement one local template via `init`. Validate a few typed parameters, plan outputs, refuse unmanaged overwrites and validate the generated configuration before writing. No arbitrary scripts or template expression language. Record template provenance as informational history only; generated sources belong to the consumer.

A template may declare package intent, but acquisition must be explicit and reproducible. A copied minimal example demonstrates the ownership model now; it is not the future initializer.

## Acceptance examples

- Exported Workflow includes a private Text; context includes both, direct import of the Text fails.
- Same local resource name in two packages is unambiguous through package identity.
- Changing only a pinned package Rule invalidates every affected local review.
- Importing a package changes no applicable rules until explicitly composed.
- A template update changes no already-created project.
- Local `check`, context compilation and CI run without a model or Python dependency owned by Markitect.
