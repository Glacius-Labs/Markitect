# Documentation map

Use this map to find the product guide or the document that owns a decision or status. It routes to those sources instead of repeating their changing conclusions.

## Start by task

| If you want to… | Start here |
|---|---|
| Work through an ordinary model-first Work Item | [Project workflow](project-workflow.md), [Project operations](project-operations.md), and the [executable Shop example](../examples/project-world/README.md) |
| Understand what Markitect is for, before assessing or redirecting it | [Markitect in brief](vision.md#markitect-in-brief), [Common misreadings](vision.md#common-misreadings), and the [concept register](concepts/register.md) |
| Understand the product goal, method, or technical boundaries | [Vision](vision.md), [Operating methodology](operating-methodology.md), [Architecture](architecture.md), and [Measurement](measurement.md) |
| Contribute source or authoring guidance | [Contribution checks](../CONTRIBUTING.md), [Development coordination](development/README.md), [Markitect-first change guide](markitect-first.md), and [Documentation maintenance](development/documentation.md) |
| Use a retained Project/Domain command or release-era contract | [Usage](usage.md) and [Operations](operations.md); these retain earlier CLI contracts alongside source and release guidance |
| Check source plans, work-item status, or fixed evidence | [Implementation roadmap](implementation-plan.md), the [work-items map](work-items/README.md), and the [validation map](validation/README.md) |
| Find a design proposal, research input, or historical record | [Design map](design/README.md), [Research map](research/README.md), [Strategy map](strategy/README.md), and [History map](history/README.md) |

## Canonical status and evidence owners

| Subject | Owner |
|---|---|
| Product thesis and human/agent responsibilities | [Vision](vision.md) |
| Product decisions, promises, assumptions, long-term vision, agreed but unscheduled directions, ideas, and open questions, with their origin | [Concept record](concepts/README.md) |
| Technical behavior and product boundaries | [Architecture](architecture.md) and the task-specific current guides above |
| Current source scope and future implementation | [Implementation roadmap](implementation-plan.md) |
| Work-package states, dependencies, owners, and zones | [Backlog](work-items/backlog.yaml), routed through the [work-items map](work-items/README.md) |
| Native A01 result, attempt accounting and acceptance limits | [A01 validation record](validation/a01-native-smoke-20261010.md) |
| Known technical limits and lessons from Product Readiness | [Product Readiness lessons survey](work-items/surveys/product-readiness-lessons-20261010.md) |
| Source/release evidence for a dated validation | The linked report in the [validation map](validation/README.md), read at its recorded scope and source |
| Published asset identity and release claims | [Production assessment](production-assessment.md) |
| Contribution and hosted validation commands | [CONTRIBUTING.md](../CONTRIBUTING.md) |

Source checks, provider runs, release evidence, comparative findings, and human acceptance answer different questions. Follow each owner for its current scope; a dated report remains evidence for the inputs and outcome it records.

## Product and CLI references

| Area | Documents |
|---|---|
| Current model-first operation | [Project workflow](project-workflow.md), [Project operations](project-operations.md), [Provider adapters](provider-adapters.md), [Repository layout](repository-layout.md), and the [Shop example](../examples/project-world/README.md) |
| Product model and method | [Vision](vision.md), [Operating methodology](operating-methodology.md), [Engineering constitution](engineering-constitution.md), [Architecture](architecture.md), [Source snapshots](source-snapshots.md), [Refinement decisions](refinement.md), and [Measurement](measurement.md) |
| Authoring and project-artifact references | [Markitect-first change guide](markitect-first.md), [Project artifact inputs](documentation.md), [Documentation routers](documentation-routers.md), [Factual consistency](consistency.md), and [Content packages](content-packages.md) |
| Experimental or retained projection APIs | [Canonical projections](canonical-projections.md) and [Project-local projections](projections.md) identify their experimental preview boundary. Check the roadmap and production assessment for current source and release scope. |
| Versioned model delivery record | [Canonical engineering model delivery](canonical-engineering-plan.md) records its v0.10.0 delivery scope; use the roadmap and production assessment for current source and release scope. |
| Distribution and retained CLI contracts | [Operations](operations.md), [Usage](usage.md), [Production assessment](production-assessment.md), [Release assessment](release-assessment.md), [WinGet package record](winget.md), and the repository [distribution guide](../integration/README.md) |

The generated [Markitect views](markitect/README.md) are readable outputs of canonical resource sources. Change their owners and regenerate the views when they need an update.

## Dated assessments and supporting maps

[Measurement](measurement.md) owns evaluation procedure. The [core authoring exercise](authoring-assessment.md) is dated 30 September 2026; the [documentation authoring pilot](authoring-pilot.md) and [MCP evaluation](mcp-evaluation.md) are dated 1 October 2026. The [Windows CLI onboarding exercise](onboarding.md) reproduces a v0.5.0 machine-run path. Their outcomes belong to their original inputs and do not update the roadmap.

Use the [design map](design/README.md) for design records, the [research map](research/README.md) for research direction and research results, the [strategy map](strategy/README.md) for original strategy sources, the [validation map](validation/README.md) for fixed reports, and the [history map](history/README.md) for the preserved pre-integration roadmap snapshot. The [work-items map](work-items/README.md) routes the backlog and its surveys.
