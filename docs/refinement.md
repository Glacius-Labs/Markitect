# Product refinement decisions

This document owns product direction beyond the implemented source model. Supported Project syntax is in [Usage](usage.md); delivery order is in the [roadmap](implementation-plan.md).

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

## Content-package design

Content reuse is a later product feature, not part of v0.2. A first useful package slice should be deliberately small:

1. Give each package a canonical identity, exact version, explicit exports, immutable source coordinate, and digest.
2. Keep local resource identity unchanged. External references include package identity; one lock selects an exact version per package.
3. Resolve a package from a vendored immutable archive offline. Validation, context, and review do not fetch network content implicitly.
4. Keep package files confined to the verified package or its declared dependencies. Never permit paths to escape into arbitrary project files.
5. Importing content does not apply its Rules. The project explicitly selects exported Rules and binds Contracts.
6. Include package resources and file inputs in context identity, impact, and evidence eligibility. Changes invalidate affected evidence conservatively.
7. Initially reject transitive imports and conflicting versions with clear diagnostics. Add a resolver only after one-package behavior and migration are well tested.
8. Treat updates as a candidate change to archive and lock together; compare old and new graphs before accepting the change.

A package is an API boundary, not a confidentiality boundary: compiled context can include dependencies needed to use an exported resource. Distribution authority must be explicit.

## Templates and interfaces

A future initializer should copy project-owned files once, validate typed parameters and planned outputs, refuse unmanaged overwrites, and validate the result before writing. It should not synchronize created projects with an evolving template, execute arbitrary scripts, or introduce an expression language.

Expose graph inspection through MCP or LSP only when calls through the stable CLI/application API show measurable friction. Runtime/operator integration requires an explicit desired-state and reconciliation contract. A conceptual diagram is not a commitment to ship every interface.

## Measurement

Use fixed snapshots, predeclared expected effects, correctness-first scoring, and repeated comparable tasks. Missing measurements are unavailable, not zero. Do not infer model token savings from context bytes or one successful evidence-reuse decision. [Measurement](measurement.md) owns the procedure.
