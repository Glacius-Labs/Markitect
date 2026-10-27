# Documentation map

Use this map to find the document that owns a decision, a guide or a piece of evidence. It routes to those sources instead of repeating their changing conclusions.

## Start by task

| If you want to… | Start here |
|---|---|
| Implement Markitect V1 | [V1 concept 1.0](concepts/v1-concept.md), [delivery plan](implementation-plan.md), [architecture](architecture.md), [code map](development/code-map.md) and [qualification record](validation/v1-qualification-20261024.md) |
| Understand what Markitect is for, before assessing or redirecting it | [Markitect in brief](vision.md#markitect-in-brief), [Common misreadings](vision.md#common-misreadings) and the [concept register](concepts/register.md) |
| Understand where the idea came from and why its mechanisms might help | [Origin and rationale](vision.md#origin-and-rationale) and the [owner's original account](concepts/sources/owner-origin-20261015.md); the archived product reflections are listed in the [archive index](archive.md#archived-pages) |
| Use the current Model → Artifacts workflow | [Project workflow](guides/project-workflow.md), [Project operations](guides/project-operations.md) and [Provider adapters](guides/provider-adapters.md) |
| Understand the technical boundaries | [Architecture](architecture.md), the package READMEs it links and [Measurement](guides/measurement.md) |
| Read the language specification | [Language contract](design/language-contract.md) and [diagnostic contract](design/diagnostic-contract.md), listed in the [design map](design/README.md) |
| Contribute source | [Contribution checks](../CONTRIBUTING.md), the [development documents](#development) below, the [repository layout](development/repository-layout.md) and [documentation maintenance](development/documentation.md) |
| Operate a release | [Operations](guides/operations.md), the [distribution guide](../packaging/README.md) and the [release evidence](validation/README.md) |
| Find an earlier design, experiment, trial or dated record | The [archive index](archive.md) |

## Canonical status and evidence owners

This is the one owner table; other pages link here instead of keeping their own.

| Subject | Owner |
|---|---|
| Product thesis, human/agent responsibilities, the delegated method, Markitect's origin and the long-term operating model | [Vision](vision.md) |
| Product decisions, promises, assumptions, long-term vision, agreed but unscheduled directions, ideas and open questions, with their origin | [Concept record](concepts/README.md) |
| Technical behavior, product boundaries, package contracts and invariants | [Architecture](architecture.md), the package READMEs and the guides above |
| Work-package scope, dependencies and reservations, current source and release status and planned implementation | [Delivery plan](implementation-plan.md); actual candidates and checks live in chats and PRs |
| Evaluation procedure, staged readiness and how benefits are tested | [Measurement](guides/measurement.md) |
| Source, native, adoption and method evidence | [Qualification record](validation/v1-qualification-20261024.md), the [validation map](validation/README.md) and the individual report, read at their recorded scope and source |
| Published asset identity of the retired releases | [Distribution retirement record](validation/distribution-retirement-20261017.md) and its [retired release mapping](validation/retired-releases-20261017.json) |
| Current design documents | [Design map](design/README.md) |
| Repository layout and where new files go | [Repository layout](development/repository-layout.md) |
| Pre-reset commits, earlier designs, experiments and records, strategy bundles and retired scope | [Archive index](archive.md) |
| Contribution and hosted validation commands, pull requests and parallel work | [CONTRIBUTING.md](../CONTRIBUTING.md) |

Source checks, provider runs, release evidence, comparative findings and human acceptance answer different questions. Follow each owner for its current scope; a dated report remains evidence for the inputs and outcome it records.

Markitect itself is developed as an ordinary software project without a canonical self-model ([DEC-076](concepts/register.md#dec-076-markitect-is-developed-as-a-regular-software-project)); these documents are its canonical description, one owner per topic. The [canonical Library model](../testdata/library/README.md) is the shared fixture of the qualification.

## Development

Start engineering work through [AGENTS.md](../AGENTS.md), the [working method](development/working-method.md) and the canonical owner of the change. Earlier handoffs, assignments, audits and the R1 repository structure record are not kept on Main; the [archive index](archive.md) lists them and the verified pre-cutover archive, and they cannot dispatch current work. This map lists the development documents:

| Document | Covers |
|---|---|
| [Working method](development/working-method.md) | The owner-adopted practice of explicit learning, complete workflows, focused design review and short milestone accounts, and the scope of evidence |
| [Documentation maintenance](development/documentation.md) | Placement, navigation, finding the canonical owner and evidence preservation |
| [Repository layout](development/repository-layout.md) | Where everything lives in this repository, where new things go and what enforces it |
| [Code map](development/code-map.md) | Every Go package with its layer, purpose and owning document; generated and checked by a test |
| [Cleanup inventory](development/cleanup-inventory-20261027.md) | R3b inventory of every Go package, command and maintained page as keep, merge, archive or delete, the canonical page per topic and the R3c work list (DEC-076) |
