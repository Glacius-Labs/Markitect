# Canonical model terminology

Status: working definitions, developed from the product discussion on 15 October 2026.

For continuation in a new session, read the [design handoff](HANDOFF.md) first. It distinguishes selected language rules, proposed index procedures, evidence and the next work.

For the selected language foundation, start with the [consolidated Core language contract](minimal-language/language-contract.md): the common Definition form, Kind and Property, binding body text, compact Properties, stable identities, imports, versions and references. The owner has selected these design rules; the new language is not yet implemented in the product compiler. The [consolidated design baseline](consolidation.md) records their origin and the other open product questions. The [source comparison](current-main-comparison.md) explains the fixed Main boundary.

This area collects the terms used to describe the canonical model. Each term has its own Markdown document explaining its meaning and purpose. The vocabulary grows one term at a time as the project-description needs become clear.

The owner's current starting point is detailed, typed documentation of the intended project. The [working goal and provisional terms](consolidation.md#arbeitsziel-und-wackelkandidaten) distinguish that description from Apply's agent coordination and the resulting artifacts. System, Component versus Module, Artifact, Mandate and Decision are explicitly under reconsideration, with System/Component/Module taking priority. The need for process descriptions is also open. Their presence in this area or an example does not settle their inclusion in the new model.

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
