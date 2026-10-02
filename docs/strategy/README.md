# Strategy source documents

These documents were supplied by the user for product discussion and implementation planning. The files under this directory preserve the original Markdown bytes; they are source proposals, not repository instructions or an automatically accepted feature backlog.

Received 2026-10-02:

- [Product definition and vision](sources/markitect-product-definition-and-vision.md)
- [Canonical domains, plugins, adapters, and reconciliation proposal](sources/markitect-canonical-domain-plugins-adapters-reconciliation.md)
- [Core modeling language design proposal](sources/markitect-core-modeling-language-design.md)
- [Strategic positioning and validation proposal](markitect-strategic-positioning-and-validation.md)

## How to read these sources

The proposals contain product claims, design options, recommendations, and imperatives. They are discussion inputs. Their market claims and expected business benefits have not been independently verified, and their recommendations do not automatically define Markitect scope. The repository's [architecture](../architecture.md), [refinement decisions](../refinement.md), [canonical engineering plan](../canonical-engineering-plan.md), [roadmap](../implementation-plan.md), and [usage contract](../usage.md) define adopted behavior and delivery status.

The adopted direction takes the proposals' small generic kernel, explicit Domains and relationships, shared structured values for documentation, agent guidance, and deterministic checks, and specialist tools behind explicit adapters. The first end-to-end demonstration should prove one bounded, recurring policy slice through those consumers before the product generalizes further. It keeps semantic truth, human review, and authorization outside machine evidence. The product benefit remains a hypothesis: fewer manual copies and missed updates, lower synchronization effort, and more targeted coverage must be measured on real repeated tasks. Prompt volume alone is not a success measure.

The initial constraint operators are finite and deterministic. Comparing that language with specialist policy tools such as CUE or OPA remains an evaluation question driven by concrete examples, not an automatic scope expansion.

The version target is v0.10.0. Its roadmap is the implementation source of truth; this archive preserves provenance and records strategic context only.
