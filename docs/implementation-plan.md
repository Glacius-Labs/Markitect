# Implementation roadmap

Updated 9 October 2026. This page summarizes the current unreleased source direction. The [Product Readiness backlog](work-items/product-readiness/backlog.yaml) is the single owner of current item statuses, dependencies, assignments, and evidence pointers; [integration progress](work-items/product-readiness/integration-progress-20261009.md) records the source handoff and current technical boundaries.

## Current source

The product path is a fixed-root project Host around the recursively managed YAML model in `.markitect/`. A normal outer Codex or Claude Code client connects through local stdio MCP and submits typed operations. The CLI and MCP compose the same application services. Source includes project setup/check/model/context/coverage, durable Plan/Run/Resume/Repair, Verify/Preflight/Apply/Deliver, and Explore/readiness/Brownfield operations. Inner Managers and reviewers use the configured Codex App Server adapter. Workspace changes reach the adopting repository only through guarded Apply.

This remains an integration candidate. Implementation and fixtures do not establish authenticated native acceptance, human acceptance, semantic correctness, or productivity benefit. Final required Linux/Windows gates and the combined native A01 journey are **NOT RUN**. Check the backlog and [acceptance ledger](work-items/product-readiness/evidence/native-acceptance-ledger.yaml) for the latest records. The required Main smoke is one ordinary Work Item flow; A02/A03 are deferred until after Main. Case Studies remain stopped, and comparative Playground work is separate from the Main gate.

## Next completion boundary

P08 completes the source layout and current documentation consolidation. P09 runs the required final-candidate build, install/onboarding, documentation, regression, independent-review, and supported-platform gates, then the separately reserved combined A01 journey when item readiness is met. P10 is ordinary reviewed integration into Main after those conditions pass. The backlog and P09 contract own the exact acceptance criteria and authorization boundaries. No release publication is authorized by this roadmap.

## Releases and historical evidence

The latest published release is v0.14.1. It preserves the earlier Project/Domain CLI and an experimental canonical Projection alpha; it does not include the current model-first project workflow. Source changes do not change a pinned release. See the [production assessment](production-assessment.md) for exact release assets and evidence.

The prior roadmap, including dated release summaries, proof dispositions, architecture assessments, and their original pinned references, is preserved in the [dated historical roadmap](history/implementation-plan-before-product-integration-20261009.md). Use its linked validation reports for evidence details; those records are not current work status.

## Historical roadmap links

These retained anchors route existing references to the corresponding dated historical sections.

<a id="astra-assessment-disposition-and-next-method-checkpoint"></a>
The [Astra disposition](history/implementation-plan-before-product-integration-20261009.md#astra-assessment-disposition-and-next-method-checkpoint) is historical; current readiness lives in the backlog.

<a id="standard-operating-model-checkpoint"></a>
The [operating-model checkpoint](history/implementation-plan-before-product-integration-20261009.md#standard-operating-model-checkpoint) is historical; current readiness lives in the backlog.

<a id="read-only-policy-failure-analysis-unreleased-source"></a>
The [policy-failure analysis](history/implementation-plan-before-product-integration-20261009.md#read-only-policy-failure-analysis-unreleased-source) is historical.

<a id="v0100--canonical-engineering-model-published"></a>
The [v0.10.0 release record](history/implementation-plan-before-product-integration-20261009.md#v0100--canonical-engineering-model-published) is historical.

<a id="knowledge-graph-assessment-and-authoring-decision"></a>
The [knowledge-graph assessment](history/implementation-plan-before-product-integration-20261009.md#knowledge-graph-assessment-and-authoring-decision) is historical; no KG work is a Main gate.
