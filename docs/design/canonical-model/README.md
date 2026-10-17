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

Status: the design basis for the rebuild commissioned in [DEC-024](../../concepts/register.md#dec-024-first-reliable-apply-rebuild-and-coordinated-start). The owner decisions include [DEC-025 to DEC-037](../../concepts/register.md#dec-025-package-managers-organize-realization) and [DEC-039](../../concepts/register.md#dec-039-deterministic-manager-process). DEC-039 owns the fixed Manager process and its confirmed refinements, including rejection counting; DEC-037 owns the container boundary. Producer packages refine exact Go names and versioned schemas in review within those decisions. These documents describe target behavior; implementation pull requests establish which parts work. Historical source comparisons retain their stated revision and are not live status. A change of intent goes to its owning document, not into a parallel note.

Read in this order:

| Document | Owns |
| --- | --- |
| [Apply concept](manager-apply-concept.md) | Organization, fixed Manager cycle, semantic review responsibilities, ownership and reporting. |
| [Library delegation walkthrough](manager-apply-library-flow.md) | The order of exchanges in one Apply of M1 and the first refusal cases. |
| [Go architecture](manager-apply-go-architecture.md) | Target packages, responsibilities and dependency direction. |
| [Port contracts](manager-apply-port-contracts.md) | Interfaces between the core and its adapters and the values they exchange. |
| [Configuration](manager-apply-configuration.md) | Execution inputs, override resolution, host profiles, identities, mandates and questions. |
| [Execution](manager-apply-execution.md) | Budget and admission, blocking findings, dependencies, rejection limits, idle policy, run states and outcomes. |
| [Journal](manager-apply-journal.md) | Durable records and storage, the controller lock, answers and continuation runs. |
| [Rebuild plan](manager-apply-rebuild-plan.md) | The container runtime, the Codex and Claude Code session adapters, and what happens to the current source. |
| [Playground](manager-apply-playground.md) | The external two-Apply experiment and its independent assessment. |
| [Acceptance gates](manager-apply-implementation-plan.md) | What each delivery step must prove. |
| [Library M1](library-example/README.md) | The example model, including its [callable interface](library-example/m1/model/library/callable-interface.yaml), and the [independent expectations](library-example/evaluation-contract.md). |
| [North star](manager-realization-north-star.md) | Ideas and variants that were not selected, later operations and open questions. |
| [Architecture review of 17 October 2026](apply-architecture-review-20261017.md) | Dated findings on this design and the proposed approach under the concept freeze ([DEC-040](../../concepts/register.md#dec-040-concept-freeze-until-the-core-runs)). |

Implementations identify the supported part of this target and its evidence. An intentional deviation changes the owning contract in the same pull request; partial implementation does not silently redefine the target. Record shapes, event tags and Go signatures in the design are illustrative unless explicitly identified as executable fixtures; the behavioral requirements remain binding. Exact implemented formats belong to the versioned contracts in source.

| Term | Meaning |
| --- | --- |
| Host | Markitect's deterministic orchestration: preparation, scheduling, validation, records and guarded writes. |
| Manager | The logical owner of one local Package's realization. The root is also an ordinary Manager, with a runtime identity but no canonical root Package. It coordinates the whole model and its aggregate integration, and may write only its owned artifacts and authorized new paths. |
| Assignment | One owner request with its caps; it links its runs. |
| Run | One Apply over fixed inputs, from Prepared to one Finished outcome. |
| Continuation | A further run of the same assignment after a run finished without completing it. |
| Run resume | Reopening the same unfinished run after an interruption; reconcile first and retain its captured run inputs. Provider conversation resume is a separate, capability-bound operation. |
| Task | One Manager responsibility within a run. |
| Attempt | One submission within a task. |
| Rejection | One rejected submission, counted per run, Manager and review purpose. Successful attempts neither use nor reset this allowance. |
| Execution session | One bound execution environment with fixed profile, mounts and authority ceiling. Starting a replacement consumes budget. |
| Conversation | Provider dialogue whose identity may survive a proven execution-session replacement; otherwise use an explicit handoff. |
| Step | One admitted Agent turn bound to an exact workspace version, supplied inputs and permitted effects. |
| Working basis | Captured source plus explicitly consumed accepted deliveries; own unaccepted changes are retained separately. Its version can advance between proven quiescent steps within the same run. |
| Candidate | An immutable artifact tree with its registry revision. |
| Delivery | A Host-accepted candidate with reviewed scope and exact evidence. It can supply a prerequisite while its producer's remaining responsibility stays open. |
| Finding | A recorded concern, blocking or advisory, that only an independent review resolves. |
| Registry | The record of which Manager owns each artifact and which Definitions it realizes. |
| Mandate | The section of a context package that states responsibility, freedom, authority and limits. |
| Final evidence check | The Host's deterministic check that all required evidence is current before publication. |

The [delivery plan](../../implementation-plan.md) owns packages, order and dispatch. The [owner discussion](sources/manager-realization-20261016.md) and the supplied [concept assessment](sources/concept-assessment-20261016.txt) and [refinements](sources/concept-refinements-20261016.txt) are the sources; the refinements are part of the concept. The [working-day reflections](../../concepts/reflections/README.md) describe the intended use and later acceptance scenarios.

<a id="working-vocabulary"></a>

## Standard-library design

Status: the owner selected fourteen Kinds across five Packages in four distribution models, all with body links and empty specs, plus security/configuration/build/test/release/operation coverage. The language is unchanged; actual library files, locks, compiler/consumer validation and releases remain implementation work.

| Document | Owns |
| --- | --- |
| [Standard-library contract](standard-library.md) | Selected scope, Kind wording owners, Package/version policy, relationship boundaries and implementation handoff. Start here. |
| [Architecture/data/deployment companion slice](standard-library-architecture-platform.md) | Six selected Kind definitions and the connected illustrative Web Library. |
| [Project coverage](standard-library-project-coverage.md) | Six selected coverage areas, proposed reference Rules/Scenarios/Checks and open project values. |
| [Data and views](standard-library-data-and-views.md) | Detailed C4/data-access/retry/migration/recovery recommendations; unselected reference technology remains a proposal. |
| [Discussion history](standard-library-history.md) | Original owner statements, intermediate proposals and superseded open-question claims. |

## Earlier vocabulary and examples

The following term pages originated in the earlier product discussion. The eight selected foundation Kind pages now own their consolidated standard-library wording. Other terms and the older [provisional-term record](consolidation.md#arbeitsziel-und-wackelkandidaten) remain historical or deferred vocabulary exploration, not additional first-scope Kinds or open language decisions.

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

The earlier proposal explored **System**, **Module** and **Mandate**; see its [architecture terms](../product-model-refinement/canonical-model-proposal.md#40-architekturbegriffe-und-ihre-abgrenzung) and [Mandate direction](../product-model-refinement/canonical-model-proposal.md#66-gewählter-dateiname-und-bevorzugte-kind-richtung). The selected System contract is now owned by the companion slice above; Module/Mandate are not added to the standard library by those historical examples.

- [Relationships](relationships.md) — how the terms connect, with simple examples.
- [Tasks example](task-management-example.md) — the earlier fourteen-term project exploration; its term set differs from the fourteen selected standard-library Kinds.
- [Connector example](connector-example.md) — a source-informed pressure test of System, Container and Component; a proposed design, not a compiled model.
- [Consolidated design history](consolidation.md) and [fixed Main comparison](current-main-comparison.md) — origin of the design and product boundaries.

Return to the [design map](../README.md).
