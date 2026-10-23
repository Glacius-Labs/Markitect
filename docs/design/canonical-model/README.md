# Canonical model and Apply design

This folder holds the completed canonical language v1 and the design of the Manager organization that realizes a model through Apply.

## Language v1

Status: completed and confirmed by the owner on 16 October 2026. The [normative language specification v1](minimal-language/language-contract.md) is the sole owner of the canonical form. There are no open language decisions within its scope. The Go production compiler is implemented; its source tests and delivery evidence describe conformance coverage and remaining work.

| Document | Role |
| --- | --- |
| [Language specification v1](minimal-language/language-contract.md) | Normative baseline: syntax, semantics, Core, models, locks and canonical bytes. |
| [Diagnostic contract](minimal-language/diagnostic-contract.md) | Separate tool profile for exact coordinates; independent of model validity and canonical bytes. |
| [Language history](minimal-language/language-history.md) and [earlier Core explorations](minimal-language/README.md) | Historical choices, superseded variants and their evidence. |
| [Procedure proposals](minimal-language/procedure-proposals.md) | Unselected, nonnormative context, impact and semantic-index procedures. |
| [Executable Exchange example](minimal-language/exchange-example/README.md) | Input files, bounded probe and reports, including the rejected Command/command vocabulary collision. |

The baseline fixes the uniform definition form, nine Core definitions, values and lists, Unicode and naming rules, identities and imports, folder-based Packages, models and dependencies, locks and canonical bytes. It prescribes no universal project vocabulary.

## Apply design

**Current accepted target, 22 October 2026:** [Agent workspaces and reviewed handoffs](manager-apply-workflow.md), adopted in [DEC-067](../../concepts/register.md#dec-067-agent-workspaces-and-reviewed-handoffs), is the behavioral owner. One Agent has one assignment container and its own working copy; Managers delegate, answer questions, review Git handoffs and integrate upward. There are no domain Turns or mandatory per-write protection/evidence protocols. Read the [rebuild plan](../../implementation-plan.md#apply-rebuild-plan) next. The S3 source and canonical self-model now describe complete fresh Run hierarchies, durable closure and stored-candidate publication. Full native Apply qualification and owner acceptance remain separate. The former Pass implementation is retired source history.

| Document | Owns |
| --- | --- |
| [Accepted Apply workflow](manager-apply-workflow.md) | Target mandates, Agent workspaces/containers, delegation, questions, reviewed Git handoffs, final verification and supersession. |
| [Apply rebuild plan](../../implementation-plan.md#apply-rebuild-plan) | Approved bounded slices, source seams, dependencies, removals, integration and distinct source/native/owner evidence. |
| [Earlier Claude handoff](../../development/claude-apply-implementation-handoff-20261020.md) | Dated DEC-064 assignment, superseded where it conflicts with DEC-067. |
| [Implemented runtime contract](manager-apply-runtime.md) | Earlier implemented behavior and limits. Old names and differing rules are not exceptions to the accepted target. |
| [Prior Apply concept](manager-apply-concept.md#current-agreed-concept--19-october-2026) | Dated 19 October direction and earlier design history; superseded where DEC-064 differs. |
| [Prior Library walkthrough](manager-apply-library-flow.md#current-library-walkthrough) | Earlier exchange sequence; the accepted workflow now contains the updated target walkthrough. |
| [Extract workflow](commands.md#extract-the-existing-stand) | Extract's separately owned target, including [stable-round structure changes](commands.md#keep-the-structure-stable-within-each-extract-round). Shared engine refactoring must preserve its distinct authority. |
| [Closing evidence](../../validation/apply-extract-closeout-20261020.md) | Build-bound observed results and unproved properties; not evidence for DEC-064. |
| [Playground](manager-apply-playground.md) | External two-Apply experiment and independent assessment. |
| [Earlier acceptance gates](manager-apply-implementation-plan.md) | Prior delivery gates; the new handoff supplies the changed acceptance cases. |
| [Library M1](library-example/README.md) | Example model and [outside evaluation contract](library-example/evaluation-contract.md). |
| [North star](manager-realization-north-star.md) | Remaining unselected ideas and later extensions; DEC-064 adopts the conversation's Manager-team direction. |

DEC-067 replaces conflicting DEC-064 Pass/Turn, shared-tree and per-write-enforcement mechanics. Original decisions and evidence keep their dates and scope. Earlier detailed protocols and their failed attempts remain in [Apply design before DEC-056](../../history/apply-design-before-dec-056/README.md). DEC-056 replaced their conflicting mechanics; DEC-058 made every Apply Execute-first; DEC-064 now refines that sequence and supersedes the initial-accounting and owner-question exceptions. Historical records retain their original wording. They cannot override the current target or dispatch new work.

Concrete Go signatures, event identifiers and versioned formats are implementation choices within the accepted behavioral requirements. The [delivery plan](../../implementation-plan.md) owns assignments and shared-file coordination. Implementation, local validation, actual runtime evidence, Main composition and human acceptance remain separate.

## Self-model refinement

[Preparation-to-publication model proposal](reconciliation-model-proposal.md) records the owner-accepted 19 October 2026 refinement: six central meanings, explicit relationships, four behavioral contracts and the preservation/change comparison. It is documentation only; canonical Definitions and implementation are unchanged. The current Apply concept continues to own behavior.

## Later commands

[Commands over the Manager organization](commands.md) owns the owner-accepted direction of [DEC-044](../../concepts/register.md#dec-044-commands-reuse-the-manager-organization): common operation instructions, read-only Verify, Extract through existing model review and integration, and model-focused Optimize. It is a later design, separate from the frozen first-Apply core and from current executable availability. The [north star](manager-realization-north-star.md#later-operations-and-extensions) routes the remaining extensions.

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
