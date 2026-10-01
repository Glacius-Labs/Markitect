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

## Measurement

Use fixed snapshots, predeclared expected effects, correctness-first scoring, and repeated comparable tasks. Missing measurements are unavailable, not zero. Do not infer model token savings from context bytes or one successful evidence-reuse decision. [Measurement](measurement.md) owns the procedure.
