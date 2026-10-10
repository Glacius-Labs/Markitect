# Development coordination

Start engineering work through the root [AGENTS.md](../../AGENTS.md) and [Markitect-first change guide](../markitect-first.md), using a fixed base and the relevant canonical owner. [CONTRIBUTING](../../CONTRIBUTING.md) owns commands and validation; [Architecture](../architecture.md) and [Repository layout](../repository-layout.md#source-repository-layout) explain the source boundaries. The [roadmap](../implementation-plan.md) and [backlog](../work-items/backlog.yaml) own current delivery status. This directory owns implementation coordination.

| Document | Owns |
|---|---|
| [Documentation maintenance](documentation.md) | Placement, navigation, canonical owners and evidence preservation |
| [Shared contracts](shared-contracts.md) | Version-bound seams and coordinator-owned changes; distinguish historical contracts from current project operations |
| [Modules](modules.md) | Final Core/Host/Module ownership, static composition and mechanical dependency laws |
| [Architecture consolidation](../validation/clean-architecture-consolidation.md) | Migration decisions, measured dependencies, compatibility and exact gate evidence |
| [Parallelizability audit](parallelizability.md) | Consolidation findings and the preserved preparation assessment |
| [Parallel work](parallel-work.md) | Delegation, isolated ownership, validation and integration |
| [Historical baseline](baseline.md) | PR #60 preparation identity and its original gates |
| [Adoption boundaries](adoption-boundaries.md) | Earlier Init/Copy Me handoff contracts; current adoption is in the Project workflow |
| [Risk register](risk-register.md) | Preparation-era classification of the 18 supplied risks |
| [Workstream map](../workstreams/README.md) | Earlier bounded specifications and hard/soft dependencies |
| [First-wave integration](../validation/parallel-development-wave-1.md) | Explicit assignments, reviewed candidates, gates, modularity findings and deferrals |

For the current model-first product, read [Project workflow](../project-workflow.md) and [Project operations](../project-operations.md). [Vision](../vision.md), [Operating methodology](../operating-methodology.md), [Measurement](../measurement.md) and the [engineering constitution](../engineering-constitution.md) own their respective intent and method boundaries.

The preparation records and [supplied bundle](../strategy/sources/parallel-development/README.md) preserve provenance; their illustrative commands and assignments are tied to that wave. New parallel work records its own full base, owned paths, contracts and exit criteria. A map or old assignment does not dispatch new work or authorize a release.
