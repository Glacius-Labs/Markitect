# Canonical model and Apply design

This folder holds the completed canonical language v1 and the design of the Manager organization that realizes a model through Apply.

## Language v1

Status: completed and confirmed by the owner on 16 October 2026. The [normative language specification v1](minimal-language/language-contract.md) is the sole owner of the canonical form. There are no open language decisions within its scope. The Go production compiler and a broader conformance corpus are implementation work.

| Document | Role |
| --- | --- |
| [Language specification v1](minimal-language/language-contract.md) | Normative baseline: syntax, semantics, Core, models, locks and canonical bytes. |
| [Diagnostic contract](minimal-language/diagnostic-contract.md) | Separate tool profile for exact coordinates; independent of model validity and canonical bytes. |
| [Language history](minimal-language/language-history.md) and [earlier Core explorations](minimal-language/README.md) | Historical choices, superseded variants and their evidence. |
| [Procedure proposals](minimal-language/procedure-proposals.md) | Unselected, nonnormative context, impact and semantic-index procedures. |
| [Executable Exchange example](minimal-language/exchange-example/README.md) | Input files, bounded probe and reports, including the rejected Command/command vocabulary collision. |

The baseline fixes the uniform definition form, nine Core definitions, values and lists, Unicode and naming rules, identities and imports, folder-based Packages, models and dependencies, locks and canonical bytes. It prescribes no universal project vocabulary.

## Apply design

Status: the design basis for the rebuild commissioned in [DEC-024](../../concepts/register.md#dec-024-first-reliable-apply-rebuild-and-coordinated-start). The owner decisions include [DEC-025 to DEC-037](../../concepts/register.md#dec-025-package-managers-organize-realization) and [DEC-039](../../concepts/register.md#dec-039-deterministic-manager-process). DEC-039 supersedes AI-planned delivery order and parent-authored operation prompts with the fixed Manager process; DEC-037 still owns the container runtime. Details below that level, such as exact schemas or default numbers like three review rounds, are design decisions under DEC-024 that review may improve. These documents describe target behavior, not implemented behavior; implementation pull requests establish the evidence, and producer packages refine exact Go names and schemas in review. A change of intent goes to the document that owns the subject, not into a new parallel note.

Read in this order:

| Document | Owns |
| --- | --- |
| [Apply concept](manager-apply-concept.md) | Organization, deterministic operation propagation, message-driven Manager cycle, remaining process questions, fixed-source comparison, review, ownership and reporting. |
| [Library delegation walkthrough](manager-apply-library-flow.md) | The order of exchanges in one Apply of M1 and the first refusal cases. |
| [Go architecture](manager-apply-go-architecture.md) | Target packages, responsibilities and dependency direction. |
| [Port contracts](manager-apply-port-contracts.md) | Interfaces between the core and its adapters and the values they exchange. |
| [Configuration](manager-apply-configuration.md) | Execution inputs, override resolution, host profiles, identities, mandates and questions. |
| [Execution](manager-apply-execution.md) | Budget and admission, blocking findings, dependencies, review rounds, run states and outcomes. |
| [Journal](manager-apply-journal.md) | Durable records and storage, the controller lock, answers and continuation runs. |
| [Rebuild plan](manager-apply-rebuild-plan.md) | The container runtime, the Codex and Claude Code session adapters, and what happens to the current source. |
| [Playground](manager-apply-playground.md) | The external two-Apply experiment and its independent assessment. |
| [Acceptance gates](manager-apply-implementation-plan.md) | What each delivery step must prove. |
| [Library M1](library-example/README.md) | The example model, including its [callable interface](library-example/m1/model/library/callable-interface.yaml), and the [independent expectations](library-example/evaluation-contract.md). |
| [North star](manager-realization-north-star.md) | Ideas and variants that were not selected, later operations and open questions. |

An implementation that deviates from a contract changes that contract in the same pull request; the documents and the code never drift apart.

| Term | Meaning |
| --- | --- |
| Host | Markitect's deterministic orchestration: preparation, scheduling, validation, records and guarded writes. |
| Manager | The logical owner of one Package's realization; the root Manager owns the whole local model. |
| Assignment | One owner request with its caps; it links its runs. |
| Run | One Apply over fixed inputs, from Prepared to one Finished outcome. |
| Continuation | A further run of the same assignment after a run finished without completing it. |
| Resume | Reopening the same unfinished run after an interruption; it reconciles first and keeps the run's basis. |
| Task | One Manager responsibility within a run. |
| Attempt | One submission within a task. |
| Round | A counted submission per Manager and review purpose; the limit on rework. |
| Candidate | An immutable artifact tree with its registry revision. |
| Delivery | A Host-accepted candidate of a task with its reviews, checks and assessments. |
| Finding | A recorded concern, blocking or advisory, that only an independent review resolves. |
| Registry | The record of which Manager owns each artifact and which Definitions it realizes. |
| Mandate | The section of a context package that states responsibility, freedom, authority and limits. |
| Final evidence check | The Host's deterministic check that all required evidence is current before publication. |

The [delivery plan](../../implementation-plan.md) owns packages, order and dispatch. The [owner discussion](sources/manager-realization-20261016.md) and the supplied [concept assessment](sources/concept-assessment-20261016.txt) and [refinements](sources/concept-refinements-20261016.txt) are the sources; the refinements are part of the concept. The [working-day reflections](../../concepts/reflections/README.md) describe the intended use and later acceptance scenarios.

## Working vocabulary

The term documents below are working vocabulary definitions from the earlier product discussion. The [working goal and provisional terms](consolidation.md#arbeitsziel-und-wackelkandidaten) record their open questions. They are possible vocabulary contents, not built-in Core Kinds or open decisions of the language.

The [standard-library discussion](standard-library.md) records the owner's 17 October 2026 selection of Concept, Rule, UseCase, Actor, Interface, Component, Scenario and Check as the first scope, all with body links and empty specs. Interface includes internal technical contracts. The [Package/version proposal](standard-library.md#package-and-version-proposal) remains under discussion. The term pages distinguish selected design directions from implementation.

- [Project](project.md) — the project's name, purpose and associated definitions.
- [Actor](actor.md) — a role through which a person or another system interacts with the system or participates in project work.
- [UseCase](use-case.md) — an action available to an Actor and its outcome, accessed through an Interface.
- [Interface](interface.md) — a contract for interaction at an external or internal boundary.
- [Concept](concept.md) — something meaningful in the project's domain, its meaning and relevant characteristics.
- [Rule](rule.md) — a condition that must hold within the project.
- [Component](component.md) — a part of the system, its purpose and responsibilities.
- [Artifact](artifact.md) — a project file or group of files and their purpose.
- [Scenario](scenario.md) — a concrete situation, the actions taken and the expected outcome.
- [Decision](decision.md) — a choice made for the project and the reasons for that choice.
- [Workflow](workflow.md) — a repeatable sequence of activities, their participants and their intended outcome.
- [Check](check.md) — how an expectation is verified and what counts as success.
- [Procedure](procedure.md) — how to carry out an activity, including its steps and expected result.
- [Condition](condition.md) — a statement that can be true or false.

The library proposal also introduces **System**, **Module** and **Mandate**; see its [architecture terms](../product-model-refinement/canonical-model-proposal.md#40-architekturbegriffe-und-ihre-abgrenzung) and [Mandate direction](../product-model-refinement/canonical-model-proposal.md#66-gewählter-dateiname-und-bevorzugte-kind-richtung).

- [Relationships](relationships.md) — how the terms connect, with simple examples.
- [Tasks example](task-management-example.md) — a small task-management project described with all fourteen terms.
- [Connector example](connector-example.md) — a source-informed pressure test of System, Container and Component; a proposed design, not a compiled model.
- [Consolidated design history](consolidation.md) and [fixed Main comparison](current-main-comparison.md) — origin of the design and product boundaries.

Return to the [design map](../README.md).
