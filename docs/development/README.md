# Development coordination

Start with the canonical [product vision](../vision.md), [Architecture](../architecture.md), the [engineering constitution](../engineering-constitution.md), [Usage](../usage.md), [roadmap](../implementation-plan.md) and [CONTRIBUTING](../../CONTRIBUTING.md). This directory owns implementation coordination, not another product vision or constitution.

| Document | Owns |
|---|---|
| [Baseline](baseline.md) | Integrated source identity, required evidence and published/source distinction |
| [Shared contracts](shared-contracts.md) | Current frozen seams and coordinator-owned changes |
| [Modules](modules.md) | Final Core/Host/Module ownership, static composition and mechanical dependency laws |
| [Architecture consolidation](../validation/clean-architecture-consolidation.md) | Migration decisions, measured dependencies, compatibility and exact gate evidence |
| [Parallelizability audit](parallelizability.md) | Actual subsystem independence, limitations and hotspots |
| [Parallel work](parallel-work.md) | Delegation, isolated ownership, validation and integration |
| [Adoption boundaries](adoption-boundaries.md) | Init, Copy Me and adopter handoffs; current versus proposed contracts |
| [Risk register](risk-register.md) | Evidence-based classification of the 18 supplied risks |
| [Workstream map](../workstreams/README.md) | Bounded future packages and hard/soft dependencies |
| [First-wave integration](../validation/parallel-development-wave-1.md) | Explicit assignments, reviewed candidates, gates, modularity findings and deferrals |

Preparation does not authorize implementation of the listed workstreams, a consumer migration or a release. The [supplied bundle](../strategy/sources/parallel-development/README.md) preserves design provenance; its illustrative commands and syntax are not shipped contracts.
