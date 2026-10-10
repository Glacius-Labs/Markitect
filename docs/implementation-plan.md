# Implementation roadmap

Updated 10 October 2026. This page summarizes the current unreleased source direction. The [Product Readiness backlog](work-items/product-readiness/backlog.yaml) is the single owner of current item statuses, dependencies, assignments, and evidence pointers; [integration progress](work-items/product-readiness/integration-progress-20261009.md) records the source handoff and current technical boundaries.

## Current source

The product path is a fixed-root project Host around the recursively managed YAML model in `.markitect/`. A normal outer Codex or Claude Code client connects through local stdio MCP and submits typed operations. The CLI and MCP compose the same application services. Source includes project setup/check/model/context/coverage, durable Plan/Run/Resume/Repair, Verify/Preflight/Apply/Deliver, and Explore/readiness/Brownfield operations. Inner Managers and reviewers use the configured Codex App Server adapter. Workspace changes reach the adopting repository only through guarded Apply.

PR #89 integrated the model-first source and documentation into Main at merge commit `f12ffb00bd6fd7bc45d0014487434a5a58aafadb`, from final head `669cecd2f3594d452a2ea820682ee8d1922d0b1b`, on 10 October 2026. The exact-head hosted run [38057729908](https://github.com/Glacius-Labs/Markitect/actions/runs/38057729908) passed both required Linux and Windows Quality jobs. The [integration progress log](work-items/product-readiness/integration-progress-20261009.md) owns merge-SHA CI and the P10 post-merge handoff; the [acceptance ledger](work-items/product-readiness/evidence/native-acceptance-ledger.yaml) preserves native attempt outcomes and accounting. Native results, CI, integration, human semantic acceptance and productivity benefit retain separate evidence boundaries. A02/A03, matched Case Studies, and comparative Playground work remain separate from this Main integration. No release publication is authorized.

## Next completion boundary

PR #89 supplies the integrated source layout and current documentation. The [progress log](work-items/product-readiness/integration-progress-20261009.md) records Main CI and post-merge handoff checkpoints; the [backlog](work-items/product-readiness/backlog.yaml) and [P09 contract](work-items/product-readiness/P09-readiness.md) own item status and acceptance. No release publication is authorized by this roadmap.

## Releases and historical evidence

The latest published release is v0.14.1. It preserves the earlier Project/Domain CLI and an experimental canonical Projection alpha; it does not include the current model-first project workflow. Source changes do not change a pinned release. See the [production assessment](production-assessment.md) for exact release assets and evidence.

The prior roadmap, including dated release summaries, proof dispositions, architecture assessments, and their original pinned references, is preserved in the [dated historical roadmap](history/implementation-plan-before-product-integration-20261009.md). Use its linked validation reports for evidence details; those records are not current work status.

## Agreed directions not yet scheduled

The [concept register](concepts/register.md#future-enhancements) lists product enhancements that Markitect's product owner has agreed or endorsed but that no work item schedules yet. When one is scheduled, its backlog item links to the register entry, and the backlog owns its status from then on.

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
