# Development coordination

Start engineering work through the root [AGENTS.md](../../AGENTS.md) and [Markitect-first change guide](../markitect-first.md), using a fixed base and the relevant canonical owner. [CONTRIBUTING](../../CONTRIBUTING.md) owns commands and validation; [Architecture](../architecture.md) and [Repository layout](../repository-layout.md#source-repository-layout) explain the source boundaries. The [roadmap](../implementation-plan.md) and [backlog](../work-items/backlog.yaml) own current delivery status. This directory owns implementation coordination.

| Document | Owns |
|---|---|
| [Documentation maintenance](documentation.md) | Placement, navigation, canonical owners and evidence preservation |
| [Shared contracts](shared-contracts.md) | Version-bound seams and coordinator-owned changes; distinguish historical contracts from current project operations |
| [Modules](modules.md) | Final Core/Host/Module ownership, static composition and mechanical dependency laws |
| [Code map](code-map.md) | Every Go package with its layer, purpose and owning document; generated and checked by a test |
| [Architecture consolidation](../validation/clean-architecture-consolidation.md) | Migration decisions, measured dependencies, compatibility and exact gate evidence |
| [Parallel work](parallel-work.md) | Engineering rules inside a work package: ownership, Module and Core change requests, shared files and evidence. The roadmap owns sessions, zones, branch names and integration order. |
| [Adoption boundaries](adoption-boundaries.md) | Earlier Init/Copy Me handoff contracts; current adoption is in the Project workflow |
| [First-wave integration](../validation/parallel-development-wave-1.md) | Explicit assignments, reviewed candidates, gates, modularity findings and deferrals |

For the current model-first product, read [Project workflow](../project-workflow.md) and [Project operations](../project-operations.md). [Vision](../vision.md), [Operating methodology](../operating-methodology.md), [Measurement](../measurement.md) and the [engineering constitution](../engineering-constitution.md) own their respective intent and method boundaries.

The [supplied bundle](../strategy/sources/parallel-development/README.md) and the first-wave record preserve the provenance of the 3 October parallel wave. That wave's parallelizability audit, risk register, baseline and workstream specifications were retired on 10 October 2026. Their lasting points are in the [Product Readiness lessons survey](../work-items/surveys/product-readiness-lessons-20261010.md#earlier-preparation-records-3-october), the risk classifications are in the [risk triage report](../validation/parallel-wave-risk-triage.md), and git history keeps the originals. New parallel work follows the roadmap and the backlog; an old assignment does not dispatch new work or authorize a release.
