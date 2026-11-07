# Documentation map

Use this map to find the document that owns a decision, a guide or a piece of evidence. It routes to those sources instead of repeating their changing conclusions.

## Start by task

| If you want to… | Start here |
|---|---|
| Implement Markitect | [Concept 4.0](concepts/v4-concept.md) (the target, which Main implements), the [lean plan](concepts/lean-tool-plan.md), the [delivery plan](implementation-plan.md), [architecture](architecture.md) and [code map](development/code-map.md); [concept 3.0](concepts/v3-concept.md) and [concept 2.0](concepts/v2-concept.md) are records of earlier targets, and [V1 concept 1.0](concepts/v1-concept.md) and the [qualification record](validation/v1-qualification-20261024.md) record V1 |
| Understand what Markitect is for, before assessing or redirecting it | [Markitect in brief](vision.md#markitect-in-brief), [Common misreadings](vision.md#common-misreadings) and the [concept register](concepts/register.md) |
| Understand where the idea came from and why its mechanisms might help | [Origin and rationale](vision.md#origin-and-rationale) and the [owner's original account](concepts/sources/owner-origin-20261015.md); the archived product reflections are listed in the [archive index](archive.md#archived-pages) |
| Use Markitect on a project: set it up, write and register the model, check, vet, and build and review it against the record of what was verified | [Project workflow](guides/project-workflow.md), the one user guide |
| Write or read a model: Definitions, Kinds and Libraries, links, the registry, the commands and their diagnostics | [Model language](guides/model-language.md), the canonical description of the language |
| Judge whether a spec is well designed: placement, coupling, hubs, size, and what `vet` measures | [What makes a good spec](guides/good-spec.md), the [model language guide](guides/model-language.md#vet-warnings-and-notes), and the [`vet` plan](concepts/vet-plan.md), the record of its decisions |
| Understand the technical boundaries | [Architecture](architecture.md), the package READMEs it links and [Measurement](guides/measurement.md) |
| Contribute source | [Contribution checks](../CONTRIBUTING.md), the [development documents](#development) below, the [repository layout](development/repository-layout.md) and [documentation maintenance](development/documentation.md) |
| Operate a release | The [distribution guide](../packaging/README.md#release-operations) and the [release evidence](validation/README.md) |
| Find an earlier design, experiment, trial or dated record | The [archive index](archive.md) |

## Canonical status and evidence owners

This is the one owner table; other pages link here instead of keeping their own.

| Subject | Owner |
|---|---|
| Product thesis, human/agent responsibilities, the delegated method, Markitect's origin and the long-term operating model | [Vision](vision.md) |
| Product decisions, promises, assumptions, long-term vision, agreed but unscheduled directions, ideas and open questions, with their origin | [Concept record](concepts/README.md) |
| The model language, the `.markitect/` layout and the model commands | [Model language](guides/model-language.md) |
| Technical behavior, product boundaries, package contracts and invariants | [Architecture](architecture.md), the package READMEs and the guides above |
| Workstreams: the Build and evaluation, Topics and Desktop sessions, the variants, their branches and state pages | [Workstreams](workstreams.md) |
| Work-package scope, dependencies and reservations, current source and release status and planned implementation | [Delivery plan](implementation-plan.md); actual candidates and checks live in chats and PRs |
| Evaluation procedure, staged readiness and how benefits are tested | [Measurement](guides/measurement.md) |
| Source, native, adoption and method evidence | [Qualification record](validation/v1-qualification-20261024.md), the [validation map](validation/README.md) and the individual report, read at their recorded scope and source |
| Published asset identity of the retired releases | [Distribution retirement record](validation/distribution-retirement-20261017.md) and its [retired release mapping](validation/retired-releases-20261017.json) |
| Current design documents | [Design map](design/README.md) |
| Repository layout and where new files go | [Repository layout](development/repository-layout.md) |
| Pre-reset commits, earlier designs, experiments and records, strategy bundles and retired scope | [Archive index](archive.md) |
| Contribution and hosted validation commands, pull requests and parallel work | [CONTRIBUTING.md](../CONTRIBUTING.md) |

Source checks, provider runs, release evidence, comparative findings and human acceptance answer different questions. Follow each owner for its current scope; a dated report remains evidence for the inputs and outcome it records.

Markitect itself is developed as an ordinary software project without a canonical self-model ([DEC-076](concepts/register.md#dec-076-markitect-is-developed-as-a-regular-software-project)); these documents are its canonical description, one owner per topic. The [canonical Library model](https://github.com/Glacius-Labs/Markitect/blob/b8fd2c94722af066ba4ff6337aeba0459cb804ec/testdata/library/README.md), the shared fixture of V1's qualification, went with the evaluation tools on 7 November 2026 ([archive](archive.md#removed-with-the-evaluation-tools)).

## Development

Start engineering work through [AGENTS.md](../AGENTS.md), the [working method](development/working-method.md) and the canonical owner of the change. Earlier handoffs, assignments, audits and the R1 repository structure record are not kept on Main; the [archive index](archive.md) lists them and the verified pre-cutover archive, and they cannot dispatch current work. This map lists the development documents:

| Document | Covers |
|---|---|
| [Working method](development/working-method.md) | The owner-adopted practice of explicit learning, complete workflows, focused design review and short milestone accounts, and the scope of evidence |
| [Documentation maintenance](development/documentation.md) | Placement, navigation, finding the canonical owner and evidence preservation |
| [Repository layout](development/repository-layout.md) | Where everything lives in this repository, where new things go and what enforces it |
| [Code map](development/code-map.md) | Every Go package with its layer, purpose and owning document; generated and checked by a test |
| [Cleanup inventory](development/cleanup-inventory-20261027.md) | R3b inventory of every Go package, command and maintained page as keep, merge, archive or delete, the canonical page per topic and the R3c work list (DEC-076) |
