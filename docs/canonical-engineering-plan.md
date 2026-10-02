# Canonical engineering model delivery

Status: implementation in progress for the next release. This plan records the authorized delivery scope; the roadmap remains the owner of release status.

## Outcome

Markitect models explicitly owned engineering knowledge and policy, validates declared structure and relationships, compiles task context, identifies change consequences, and projects or reconciles explicitly configured consumers. Narrative and semantic judgment remain with their authors and reviewers. Project artifacts remain opaque inputs to the core; specialized analysis is an adapter responsibility.

## Release scope

1. A versioned language registry with strict resource schemas, typed references and relation definitions. Domain definitions are fixed inputs loaded before resource validation. Unknown fields, ambiguous identities and invalid extension definitions fail closed.
2. A normalized semantic model with resource origin, source locations, resolved relationships, constraint definitions, a successful-validation status and reproducible input identities. Invalid models retain their diagnostics and cannot enter adapters. The existing AI engineering vocabulary becomes a supplied domain; additional software and delivery concepts exercise the same mechanisms.
3. Relation-specific context, invalidation and cycle behavior. Constraints and selections record their evaluation inputs; additions, removals and changed selection membership invalidate affected evidence conservatively.
4. Explicit domain policy and technology mapping. Human documentation, agent guidance and deterministic checks consume the same structured definitions. Adapter configuration is canonical, owned input and never silently inferred from an external output.
5. Adapter capabilities for projection, observation, planning, application and verification. Capabilities are independently declared. Local output application uses existing controlled writers; external actions require explicit project configuration and a concrete captured plan. Observations do not overwrite desired source. Reconciliation also detects external drift with unchanged model inputs.
6. Executable examples, negative boundary tests, Windows/Linux CI, packaged source/binary checks, updated authoring guidance and a versioned release with source and asset evidence.

## Delivery order

- Establish shared language and adapter contracts, then implement independent kernel and adapter workstreams.
- Integrate parsing, graph, context, impact, formatting and rendering against the normalized model.
- Exercise software policy and delivery policy through documentation, agent context and checks; exercise observed drift and stale-plan rejection.
- Review the integrated candidate independently, fix findings, run complete release gates and publish the next immutable release through the existing process.

## Completion criteria

An adopting project can define a domain without modifying the Markitect core, author typed engineering resources and policies, validate their constraints, obtain bounded context and explainable impact, generate consistent declared outputs, and inspect/apply/verify an explicitly configured reconciliation plan. Results identify their source, domain and adapter inputs. Unsupported observations and incomplete checks remain visibly incomplete. No result implies semantic truth or human acceptance.

Continuous operators, general source-language semantics, automatic natural-language policy formalization, arbitrary source-code execution inside the deterministic evaluator and a hosted plugin marketplace are outside this release. These are not necessary to deliver the canonical-model product.

## Strategic guardrails and validation

The user-provided [strategic analysis](strategy/markitect-strategic-positioning-and-validation.md) is preserved as a discussion source, with its status and provenance recorded in the [strategy index](strategy/README.md). It informs decisions; its market claims and proposed integrations are hypotheses rather than verified product evidence or an automatic feature backlog.

This release follows the small-kernel and incremental-adoption direction: project-owned domains, a finite structural assertion vocabulary, independent consumers of one normalized model, and specialist analysis outside the compiler. The executable policy slices prove technical consistency and boundary behavior. They do not establish reduced engineering effort or market demand.

Before expanding the language or adapter ecosystem, evaluate real policy changes against the prior manual process. Record independently maintained copies, manual synchronization steps, missed updates, the resources and consumers inspected, setup and maintenance effort, and unresolved semantic review. Compare with a composition of existing tools. Generalize only when the observed reduction in synchronization cost justifies the modeling and integration cost. Additional constraint expressiveness requires a concrete policy use case and an explicit CUE/OPA integration assessment.
