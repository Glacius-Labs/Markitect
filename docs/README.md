# Documentation map

Use this map to find the product guide or the document that owns a decision or status. It routes to those sources instead of repeating their changing conclusions.

## Start by task

| If you want to… | Start here |
|---|---|
| Implement the approved narrow Markitect V1 core | [Source implementation approval](concepts/sources/v1-source-implementation-approval-20261023.md), [verified archive](development/v1-source-archive-20261023.md), [source audit](development/v1-architecture-audit-20261023.md), [concrete Go architecture](design/v1-execution-architecture.md), [current plan](implementation-plan.md#v1-implementation-organization), and [bounded assignments](development/v1-workstream-assignments.md) |
| Read the retired Cockpit design and its dated evidence | [Historical Cockpit concept](design/cockpit.md), [Claude handoff](development/claude-cockpit-handoff-20261020.md), [organization chart handoff](development/codex-cockpit-org-chart-handoff-20261021.md), and [original commission](concepts/sources/cockpit-commission-20261020.md); Cockpit and interactive authoring are excluded from V1 |
| Use the current Model -> Artifacts workflow | [Project workflow](project-workflow.md) and [V1 scope](implementation-plan.md); the [three starting situations](vision.md#three-starting-situations) retain their conceptual history, while Extract is retired from V1 |
| Implement or review the accepted Apply refinement | [Evidence-backed acceptance](design/canonical-model/manager-apply-workflow.md#evidence-backed-acceptance-after-the-codex-retest), [Codex conformance handoff](development/codex-apply-conformance-handoff-20261021.md), [completed Codex retest](validation/apply-codex-conformance-20261021.md), [DEC-064](concepts/register.md#dec-064-checked-recursive-apply-passes-and-manager-teams), then the [implemented base runtime contract](design/canonical-model/manager-apply-runtime.md) |
| Inspect current canonical v1 preparation | [Project workflow](project-workflow.md), [Project operations](project-operations.md), and the [self-model](../.markitect/README.md) |
| Understand what Markitect is for, before assessing or redirecting it | [Markitect in brief](vision.md#markitect-in-brief), [Common misreadings](vision.md#common-misreadings), and the [concept register](concepts/register.md) |
| Understand where the idea came from and why its mechanisms might help | [Origin and rationale](concepts/origin-and-rationale.md) and the [owner's original account](concepts/sources/owner-origin-20261015.md) |
| Explain Markitect, prepare a pitch or plan marketing material | [Marketing working plan](strategy/marketing-plan.md), with draft pitches, audience hypotheses and evidence-based artifact sequencing |
| Trace the project's development, major turns and working method | [Project history, timeline and process assessment — 17 October 2026](history/project-history-20261017.md), with a fixed historical cutoff |
| Apply the adopted design and delivery working method | [Working method](development/working-method.md), [DEC-038](concepts/register.md#dec-038-learn-through-complete-workflows), and the [delivery plan](implementation-plan.md) |
| Continue thoughts about future use and organization | [Product reflections](concepts/reflections/README.md), starting with [trust through organized agent work](concepts/reflections/organization-and-trust-20261016.md) |
| Understand the product goal, method, or technical boundaries | [Vision](vision.md), its [delegated method](vision.md#the-delegated-method), [Architecture](architecture.md), and [Measurement](measurement.md) |
| Contribute source or authoring guidance | [Contribution checks](../CONTRIBUTING.md), [Development coordination](development/README.md), [Project workflow](project-workflow.md), and [Documentation maintenance](development/documentation.md) |
| Use the current model-first CLI | [Project operations](project-operations.md), [Project workflow](project-workflow.md), and [Provider adapters](provider-adapters.md) |
| Check source plans, earlier work records, or fixed evidence | [Delivery plan](implementation-plan.md), the [historical surveys](history/surveys/README.md), and the [validation map](validation/README.md) |
| Find a design proposal, research input, or historical record | [Design map](design/README.md), [Research map](research/README.md), [Strategy map](strategy/README.md), and [History map](history/README.md) |
| Read the completed canonical-model v1 design baseline | [Normative language specification v1](design/canonical-model/minimal-language/language-contract.md), [overview and evidence boundaries](design/canonical-model/README.md), and [earlier consolidation history](design/canonical-model/consolidation.md) |

## Canonical status and evidence owners

| Subject | Owner |
|---|---|
| Product thesis and human/agent responsibilities | [Vision](vision.md) |
| Product decisions, promises, assumptions, long-term vision, agreed but unscheduled directions, ideas, and open questions, with their origin | [Concept record](concepts/README.md) |
| Technical behavior and product boundaries | [Architecture](architecture.md) and the task-specific current guides above |
| Work-package scope, dependencies and reservations | [Delivery plan](implementation-plan.md); actual candidates/checks live in chats and PRs. [Earlier records](history/surveys/README.md) are historical. |
| Read earlier coordination checkpoints | [Historical handoffs](history/README.md#handoffs-and-surveys); current assignments remain in the delivery plan and delivery chats |
| Earlier interrupted sessions and restart records | [Historical session handoff](history/handoffs/session-handoff.md); its retired queue grants no current assignment |
| Native A01 result, attempt accounting and acceptance limits | [A01 validation record](validation/a01-native-smoke-20261010.md) |
| Known technical limits and lessons from Product Readiness | [Product Readiness lessons survey](history/surveys/product-readiness-lessons-20261010.md) |
| Source/release evidence for a dated validation | The linked report in the [validation map](validation/README.md), read at its recorded scope and source |
| Published asset identity and release claims | [Production assessment](history/distribution/production-assessment.md) |
| Contribution and hosted validation commands | [CONTRIBUTING.md](../CONTRIBUTING.md) |

Source checks, provider runs, release evidence, comparative findings, and human acceptance answer different questions. Follow each owner for its current scope; a dated report remains evidence for the inputs and outcome it records.

## Product and CLI references

| Area | Documents |
|---|---|
| Current model-first operation | [Project workflow](project-workflow.md), [Project operations](project-operations.md), [Provider adapters](provider-adapters.md), [Repository layout](repository-layout.md), and the [canonical Library model](design/canonical-model/library-example/README.md) |
| Product model and method | [Vision](vision.md), its [delegated method](vision.md#the-delegated-method), [Architecture](architecture.md), [Source snapshots](source-snapshots.md), and [Measurement](measurement.md) |
| Authoring and project-artifact references | [Project workflow](project-workflow.md), [Project operations](project-operations.md), and [Documentation maintenance](development/documentation.md) |
| Historical projection APIs | The removed canonical Projection alpha is preserved in the [v0.14.1 historical guide](history/canonical-projections-v0.14.1.md) and the immutable [release source](https://github.com/Glacius-Labs/Markitect/tree/v0.14.1). |
| Versioned model delivery record | [Canonical engineering model delivery](https://github.com/Glacius-Labs/Markitect/blob/1f4a1ede214d05fd2fee615b3905207df7702426/docs/canonical-engineering-plan.md) records its v0.10.0 delivery scope; use the roadmap and production assessment for current source and release scope. |
| Distribution and release work | [Operations](operations.md) and the repository [distribution guide](../integration/README.md); old release identities and assessments are under [History](history/README.md#distribution-evidence) |

The generated [project view](markitect/project.md) is a readable output of the canonical project model. Change its owner and regenerate the view when it needs an update.

## Dated assessments and supporting maps

[Measurement](measurement.md) owns evaluation procedure. The [core authoring exercise](history/assessments/authoring-assessment-20260930.md) is dated 30 September 2026; the [documentation authoring pilot](history/assessments/authoring-pilot-20261001.md) and [MCP evaluation](history/assessments/mcp-evaluation-20261001.md) are dated 1 October 2026. The [Windows CLI onboarding exercise](history/assessments/windows-onboarding.md) reproduces a v0.5.0 machine-run path. Their outcomes belong to their original inputs and do not update the roadmap.

Use the [design map](design/README.md) for design records, the [research map](research/README.md) for research direction and research results, the [strategy map](strategy/README.md) for original strategy sources, the [validation map](validation/README.md) for fixed reports, and the [history map](history/README.md) for the preserved pre-integration roadmap snapshot. The [historical surveys](history/surveys/README.md) routes earlier records and surveys.
