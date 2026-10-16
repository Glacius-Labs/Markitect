# Canonical model: completed v1 design baseline

Status: v1 language design completed and confirmed by the owner on 16 October 2026.

The [normative language specification v1](minimal-language/language-contract.md) is the sole owner of the completed canonical form. There are no open language decisions within its selected scope. Production implementation in Go and a broader conformance corpus remain separate work; bounded design probes do not establish full product support.

The linked histories preserve the derivation and superseded choices. Project vocabulary and the unselected procedure proposals remain outside the completed language baseline.

| Document | Role |
| --- | --- |
| [Language specification v1](minimal-language/language-contract.md) | Completed normative v1 design baseline: syntax, semantics, Core, models, locks and canonical bytes. |
| [Diagnostic contract](minimal-language/diagnostic-contract.md) | Separate tool profile for exact coordinates; independent of model validity and canonical bytes. |
| [Language history](minimal-language/language-history.md) and [earlier Core explorations](minimal-language/README.md) | Historical choices, superseded variants and evidence of their recorded scope. |
| [Procedure proposals](minimal-language/procedure-proposals.md) | Unselected, nonnormative context, impact and semantic-index procedures. |
| [Manager realization north star](manager-realization-north-star.md) | Working direction and open variants for one Manager per Package, configurable teams, artifact registration and simple context delivery. |
| [Manager organization and reliable Apply](manager-apply-concept.md) | Developed proposal resolving those variants into organization, context, ownership, integration, verification and reporting, with a library walkthrough. |
| [Implementation plan for reliable Apply](manager-apply-implementation-plan.md) | Two real library Applies around a model change, Go and Playground design, Agent communication and a clean rebuild without compatibility obligations. |
| [Supplied Apply review and resolution](review-resolution-20261016.md) | Preserved assessment sources, five refinements, superseded migration advice and the next concrete design work. |
| [Small library model M1](library-example/README.md) | Canonical YAML from the confirmed business rules, a small ordinary vocabulary, explicit Package links and bounded structural evidence. |
| [Go architecture outline for reliable Apply](manager-apply-go-architecture.md) | Ports and Adapters responsibilities, idiomatic Go package proposal and treatment of current source. |
| [Library delegation walkthrough](manager-apply-library-flow.md) | Root, lending, fresh review, rework and integration traced against M1, with concrete refusal cases; design rather than an executed run. |
| [Apply port contracts](manager-apply-port-contracts.md) | Proposed Go interfaces, value ownership and behavioral contracts for acquisition, sessions, workspaces, checks, journal and publication. |
| [Executable Exchange example](minimal-language/exchange-example/README.md) | Current input files, bounded probe and reports; includes the explicitly rejected Command/command vocabulary collision. |
| [Consolidated design history](consolidation.md) and [fixed Main comparison](current-main-comparison.md) | Origin of the design and product boundaries. |

The language baseline fixes the uniform definition form, nine Core definitions, values and lists, Unicode and naming rules, identities and imports, folder-based Packages, models and dependencies, locks and canonical bytes. It does not prescribe a universal project vocabulary.

The term documents below retain their status as working vocabulary definitions developed from the earlier product discussion. The [working goal and provisional terms](consolidation.md#arbeitsziel-und-wackelkandidaten) record those vocabulary questions and distinguish intended project descriptions from Apply's agent coordination and resulting artifacts. Their presence here does not make them built-in Core Kinds or open decisions in the completed language design.

The [language discussion and bounded Connector probes](minimal-language/README.md) preserve the earlier alternatives and their evidence. They are the derivation of the current contract. The terms below are possible vocabulary contents; they do not predetermine a universal built-in Kind catalogue. Special requirements and roles fields remain outside the selected Core.

## Terms

- [Project](project.md) — the project's name, purpose and associated definitions.
- [Actor](actor.md) — a role through which a person or another system interacts with the system or participates in project work.
- [UseCase](use-case.md) — an action available to an Actor and its outcome, accessed through an Interface.
- [Interface](interface.md) — an interaction point through which an Actor uses UseCases.
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

The library proposal also introduces **System**, **Module** and **Mandate**; see its [architecture terms](../product-model-refinement/canonical-model-proposal.md#40-architekturbegriffe-und-ihre-abgrenzung) and [Mandate direction](../product-model-refinement/canonical-model-proposal.md#66-gewählter-dateiname-und-bevorzugte-kind-richtung). Together, the two lines name seventeen terms whose need and boundaries are being reviewed.

## Relationships and example

- [Relationships](relationships.md) — how the terms connect, with simple examples and explanations of their different purposes.
- [Tasks example](task-management-example.md) — a small task-management project described using all fourteen terms and their relationships.
- [Connector architecture and request/reply example](connector-example.md) — local source-informed pressure test of System/Container/Component, topic selection, response matching and remaining vocabulary gaps; proposed design, not a compiled model or accepted new schema.

Return to the [design map](../README.md).
