# Candidate model language

This language is used for the documentation exercise only. It does not claim loader support, define a YAML format or change Markitect's accepted product vocabulary.

## Seven kinds

| Kind | Owns | Does not establish |
|---|---|---|
| Concept | Shared domain meaning, properties and valid domain states | A source-language class or implementation file |
| Requirement | A goal, boundary or independently applicable obligation | Implementation or a passing verification result |
| UseCase | One offered action: trigger, input, flow, observable result and failures | A shared obligation copied from another owner |
| Component | A named responsibility boundary and its architecture | An agent, team or compulsory package |
| Artifact | An expected repository output and its content/path commitment | Evidence that the output already exists |
| Check | A verification method and expected observations | An execution result or acceptance decision |
| Decision | A choice, its context, alternatives, rationale and consequences | Another copy of the resulting contract |

The application itself is the root Component. Package title, date and provenance are document metadata; they do not introduce an eighth Project kind.

## Identity, body and status

Every record has an explicit HTML anchor and a stable identity, for example `UC-ADD`. Its heading and body own its definition. Record identities are unique within this example. All records have intent status **proposed** through this package-wide declaration; no record is implicitly accepted. Requirements additionally state whether they are a **goal** or a **must**. Must means obligatory if the specification is accepted, not that it has already been approved.

Decisions state their choice status separately: **selected in this draft** or **deliberately left to implementation**. A selected draft choice is not a human acceptance claim. Application realization is **not implemented in this branch**; application Check execution is **not performed**. Structural checks of these documents and independent review must be reported separately.

Natural language remains necessary. A kind and a resolved edge cannot prove that prose is correct, complete, mutually consistent or useful. Definitions stay cohesive; one sentence does not have to become one record.

## Requirement and UseCase boundary

A Requirement is independently relevant across operations, artifacts or architectural boundaries: durability, safe handling of a corrupt file, scope or documentation maintenance. A UseCase owns its local behavior: a newly created task is open, list output ordering, or what completing an already completed task returns. A Concept owns shared domain meaning, such as the two task states.

A UseCase invokes shared obligations by explicit edges and by referring to their owner when helpful. It does not restate them as a competing rule. Checks may verify a UseCase or Concept directly; a local behavior does not need an artificial Requirement solely to become checkable.

Decision bodies explain why; the subject record owns what was chosen. Artifact bodies define expected repository resources; physical existence is a later realization observation. A development procedure is described by `REQ-CHANGE` and the developer-guide Artifact in this small example; no eighth Workflow kind is needed for the short process.

## Explicit relationships

The edge table in [Relationships](relations.md) is authoritative. Links elsewhere only navigate to definitions. The names below are provisional vocabulary, not CLI commands or a schema.

| Relation | Source kind | Target kind | Meaning |
|---|---|---|---|
| uses | UseCase | Concept | The action uses the concept's domain meaning. |
| applies-to | Requirement | Component, UseCase, Artifact | The requirement constrains that scope. |
| contains | Component | Component | Structural composition; no dependency is implied. |
| depends-on | Component | Component | A capability of the target is required by the source. |
| provided-by | UseCase | Component | The target owns the offered action. |
| owned-by | Artifact | Component | One accountable component owns the planned repository output. |
| realizes | Artifact | Component, UseCase, Requirement, Check | The future artifact is intended to realize this contract or verification method. |
| verifies | Check | Concept, Requirement, UseCase, Component, Artifact | The check examines the specified target within its declared limits. |
| explains | Decision | Concept, Requirement, UseCase, Component, Artifact, Check | The choice explains or deliberately leaves freedom in the target definition. |

No automatic inverse edges are additional facts. `uses` and `explains` do not impose execution order. `applies-to` does not assert that the Requirement has been satisfied. `realizes` describes an intended mapping, not successful realization. Effects on compiler digests, context, impact, invalidation or execution are not decided by this experiment.

## Candidate structural validation rules

- Every edge resolves to declared records of allowed kinds; identities and identical edges are unique.
- `COMP-APP` is the sole root. Every other Component has one parent. Containment and dependency graphs are acyclic; a component does not depend on itself.
- Every UseCase has exactly one `provided-by` target and at least one directly targeting Check.
- Every Artifact has exactly one `owned-by` target. Planned output paths do not overlap between Artifact owners. Implementation-selected manifests must be added to the tooling Artifact's concrete path list before a realization coverage claim.
- Every Requirement has at least one scope and at least one Check, including human inspection for the goal. A Check must state expected observation and limitations; no requirement is considered proven by the presence of an edge.
- Every Component has a planned realizing Artifact. Every record participates in the edge list; the application root is intentionally structural.

These are reviewable rules of the candidate documentation model. The current Markitect check covers the files as Markitect documentation; it does not execute these rules or validate the proposed seven-kind graph. See [the assessment](assessment.md).
