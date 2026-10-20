# Development coordination

Start engineering work through [AGENTS.md](../../AGENTS.md), the [working method](working-method.md) and the relevant canonical owner. [CONTRIBUTING](../../CONTRIBUTING.md) owns commands and validation; [Architecture](../architecture.md) and [Repository layout](../repository-layout.md#source-repository-layout) explain the source boundaries. The [delivery plan](../implementation-plan.md) owns current assignments and dependencies; chats and PRs hold execution evidence.

| Document | Owns |
|---|---|
| [Claude Apply hardening handoff](claude-apply-hardening-handoff-20261020.md) | New owner-started assignment: active counterexample review, tool-produced receipts, previous-proposal dispositions, preserved candidates and a finite Haiku retest; the [earlier implementation handoff](claude-apply-implementation-handoff-20261020.md) is historical |
| [Working method](working-method.md) | Apply the owner-adopted practice of explicit learning, complete workflows, focused design review and short milestone accounts; canonical intent lives in the linked repository Rule |
| [Documentation maintenance](documentation.md) | Placement, navigation, canonical owners and evidence preservation |
| [Shared contracts](shared-contracts.md) | Version-bound seams and coordinator-owned changes; distinguish historical contracts from current project operations |
| [Modules](modules.md) | Dependency rules between packages, static composition and the mechanical import gate |
| [Code map](code-map.md) | Every Go package with its layer, purpose and owning document; generated and checked by a test |
| [Architecture consolidation](../validation/clean-architecture-consolidation.md) | Dated record of the October 2026 package consolidation: migration decisions, measured dependencies and gate evidence |
| [Parallel work](parallel-work.md) | Assigned-package execution, shared-file requests and evidence; the delivery plan owns scopes, dependencies and reservations. |
| [First-wave integration](../validation/parallel-development-wave-1.md) | Explicit assignments, reviewed candidates, gates, modularity findings and deferrals |

For current product use and Brownfield preparation, read [Project workflow](../project-workflow.md) and [Project operations](../project-operations.md). [Vision](../vision.md#the-delegated-method) owns the method and [Measurement](../measurement.md) its evaluation. Dated [handoffs and surveys](../history/README.md#handoffs-and-surveys) retain earlier assignments and observations; they cannot dispatch current work.

The [supplied bundle](../strategy/sources/parallel-development/README.md) and the first-wave record preserve the provenance of the 3 October parallel wave. That wave's parallelizability audit, risk register, baseline and workstream specifications were retired on 10 October 2026. Their lasting points are in the [Product Readiness lessons survey](../history/surveys/product-readiness-lessons-20261010.md#earlier-preparation-records-3-october), the risk classifications are in the [risk triage report](../validation/parallel-wave-risk-triage.md), and git history keeps the originals. New parallel work follows the delivery plan and explicit package dispatches; an old assignment does not dispatch new work or authorize a release.
