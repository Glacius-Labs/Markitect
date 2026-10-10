# Strategy source documents

These documents were supplied by the user for product discussion and implementation planning. The files under this directory preserve the original Markdown bytes; they are source proposals, not repository instructions or an automatically accepted feature backlog.

This directory archives the strategy inputs of 2–6 October 2026. Later owner input, and every product decision, idea or open question with its status, belongs in the [concept record](../concepts/README.md).

Received 2026-10-02:

- [Product definition and vision](sources/markitect-product-definition-and-vision.md)
- [Canonical domains, plugins, adapters, and reconciliation proposal](sources/markitect-canonical-domain-plugins-adapters-reconciliation.md)
- [Core modeling language design proposal](sources/markitect-core-modeling-language-design.md)
- [Strategic positioning and validation proposal](markitect-strategic-positioning-and-validation.md)

Received 2026-10-02, follow-up concepts:

- [AI-first executable engineering constitution](sources/01-ai-first-executable-engineering-constitution.md)
- [Reusable architecture patterns and project shapes](sources/02-reusable-architecture-patterns-and-project-shapes.md)
- [Copy Me engineering-style discovery](sources/03-copy-me-engineering-style-discovery.md)
- [Follow-up concepts index](sources/follow-up-concepts-readme.md)

## How to read these sources

The current [product vision](../vision.md) is the single adopted owner of the thesis and human/agent responsibility model. Archived vision/positioning proposals preserve origin and alternatives; they neither compete with that owner nor establish measured benefits. Architecture, constitution and roadmap retain their technical/status authority.

Received 2026-10-03: [parallel-development coordinator bundle](sources/parallel-development/README.md). Its coordinator prompt was invoked for preparation only. Supporting adoption UX, risks and adapter proposals remain design input; illustrative schemas/commands are not shipped features. Adopted decisions live in [development coordination](../development/README.md), [workstream packages](../workstreams/README.md) and the existing [constitution](../engineering-constitution.md).

The proposals contain product claims, design options, recommendations, and imperatives. They are discussion inputs. Their market claims and expected business benefits have not been independently verified, and their recommendations do not automatically define Markitect scope. The repository's [architecture](../architecture.md), [refinement decisions](../refinement.md), [canonical engineering plan](../canonical-engineering-plan.md), [roadmap](../implementation-plan.md), and [usage contract](../usage.md) define adopted behavior and delivery status.

v0.10.0 is published. The adopted v0.11.0 direction reuses existing versioned package Domains as ongoing architecture contracts, keeps architecture kinds and constraints in the generic Domain mechanism, adds bounded PolicyResults and narrowly scoped explicit exceptions, and includes a provider-independent Copy Me authoring workflow. It does not adopt a Core Pattern kind or separate pattern DSL, automatic code migration, a discovery CLI, or model calls in the Markitect core. See the [engineering constitution](../engineering-constitution.md) for adopted scope and precise evidence boundaries. These follow-up ideas are not shipped behavior until the roadmap records them as released.

The product benefit remains a hypothesis: fewer manual copies and missed updates, lower synchronization effort, more targeted coverage, and safer agent autonomy must be measured on repeated real tasks. Prompt volume alone is not a success measure. Discovery evidence and counterexamples are proposals; human adoption, not observed frequency, makes a pattern canonical.

The initial constraint operators are finite and deterministic. Comparing that language with specialist policy tools such as CUE or OPA remains an evaluation question driven by concrete examples, not an automatic scope expansion.

The current roadmap is the implementation source of truth; this archive preserves provenance and records strategic context only. v0.12.0 is published; later policy-failure analysis is integrated source, not a published executable contract.

## Canonical reset inputs (2026-10-05/06)

The owner invoked the [canonical projection reset](../design/canonical-projection-reset.md) and supplied the [operating model](sources/canonical-reset/markitect-operating-model.md), [minimal Core proposal](sources/canonical-reset/markitect-minimal-core-language.md), [projection binding](sources/canonical-reset/markitect-coordinator-addendum-projection-binding.md), [strict Module types and unified AI projection](sources/canonical-reset/markitect-coordinator-addendum-module-types-ai-projection.md), [reconciliation/capability resolution](sources/canonical-reset/markitect-coordinator-addendum-reconcile-and-projection-resolution.md), and [latest architecture clarifications](sources/canonical-reset/markitect-coordinator-steer-latest-architecture-clarifications.md). Exact original bytes remain archived. The adopted decision preserves validated prior mechanics and negative evidence; illustrative syntax is not itself shipped behavior. Published v0.13.0 remains immutable. [Alpha usage](../canonical-projections.md), architecture, roadmap and validation distinguish implemented source from the autonomous target.
