# Documentation map

Use this map to find the document that owns a decision, a guide or a piece of evidence. It routes to those sources instead of repeating their changing conclusions.

## Start by task

| If you want to… | Start here |
|---|---|
| Implement Markitect V1 | [V1 concept 1.0](concepts/v1-concept.md), [delivery plan](implementation-plan.md), [concrete Go architecture](design/v1-execution-architecture.md), [code map](development/code-map.md) and [qualification record](validation/v1-qualification-20261024.md) |
| Understand what Markitect is for, before assessing or redirecting it | [Markitect in brief](vision.md#markitect-in-brief), [Common misreadings](vision.md#common-misreadings) and the [concept register](concepts/register.md) |
| Understand where the idea came from and why its mechanisms might help | [Origin and rationale](concepts/origin-and-rationale.md), the [owner's original account](concepts/sources/owner-origin-20261015.md) and the [product reflections](concepts/reflections/README.md) |
| Use the current Model → Artifacts workflow | [Project workflow](guides/project-workflow.md), [Project operations](guides/project-operations.md) and [Provider adapters](guides/provider-adapters.md) |
| Understand the technical boundaries | [Architecture](architecture.md), [Source snapshots](guides/source-snapshots.md) and [Measurement](guides/measurement.md) |
| Read the language specification | [Language contract](design/language-contract.md) and [diagnostic contract](design/diagnostic-contract.md), listed in the [design map](design/README.md) |
| Contribute source | [Contribution checks](../CONTRIBUTING.md), [development map](development/README.md), [repository layout](development/repository-layout.md) and [documentation maintenance](development/documentation.md) |
| Operate a release | [Operations](guides/operations.md), the [distribution guide](../packaging/README.md) and the [release evidence](validation/README.md) |
| Find an earlier design, experiment, trial or dated record | The [archive index](archive.md) |

## Canonical status and evidence owners

| Subject | Owner |
|---|---|
| Product thesis and human/agent responsibilities | [Vision](vision.md) |
| Product decisions, promises, assumptions, long-term vision, agreed but unscheduled directions, ideas and open questions, with their origin | [Concept record](concepts/README.md) |
| Technical behavior and product boundaries | [Architecture](architecture.md) and the guides above |
| Work-package scope, dependencies and reservations | [Delivery plan](implementation-plan.md); actual candidates and checks live in chats and PRs |
| Source and native evidence of V1 | [Qualification record](validation/v1-qualification-20261024.md) and the [validation map](validation/README.md), read at their recorded scope and source |
| Published asset identity and release claims | [Production assessment](validation/production-assessment.md) |
| Pre-reset commits, earlier designs and retired scope | [Archive index](archive.md) |
| Contribution and hosted validation commands | [CONTRIBUTING.md](../CONTRIBUTING.md) |

Source checks, provider runs, release evidence, comparative findings and human acceptance answer different questions. Follow each owner for its current scope; a dated report remains evidence for the inputs and outcome it records.

The generated [project view](markitect/project.md) is a readable output of the canonical project model. Change its owner and regenerate the view when it needs an update. The [canonical Library model](../testdata/library/README.md) is the shared fixture of the qualification.
