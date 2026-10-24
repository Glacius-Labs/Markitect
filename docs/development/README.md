# Development coordination

Start engineering work through [AGENTS.md](../../AGENTS.md), the [working method](working-method.md) and the relevant canonical owner. [CONTRIBUTING](../../CONTRIBUTING.md) owns commands and validation; [Architecture](../architecture.md) and [Repository layout](../repository-layout.md#source-repository-layout) explain the source boundaries. The [delivery plan](../implementation-plan.md) owns current assignments and dependencies; chats and PRs hold execution evidence.

| Document | Owns |
|---|---|
| [V1 bounded assignments](v1-workstream-assignments.md) | Current writer scopes for steps 2 and 3 of the delivery plan: Run contract (D10–D13, D15, D17) and provider scope (D18) |
| [Repository structure](repository-structure.md) | R1 inventory of every file, the proposed target layout, the archive mechanism and the ordered R2 work list; a proposal until the owner accepts it |
| [Concept conformance audit](concept-conformance-audit-20261024.md) | Supplied static audit of Main `6c7d758c` against the V1 concept; origin of A1–A6 and R1–R3, closed by concept 1.0 |
| [V1 counter-review](v1-architecture-audit-20261023.md) and [source archive](v1-source-archive-20261023.md) | Source-bound v0.17 counter-review and the verified pre-cutover archive |
| [Codex Apply conformance handoff](codex-apply-conformance-handoff-20261021.md) | Historical, superseded by V1: bounded correction: material-condition evidence before Implement acceptance, parent integration, independently derived Verify coverage, adjudicated controls and a new finite targeted series; prior series is exhausted |
| [Direct runtime and evaluation handoff](claude-direct-runtime-handoff-20261021.md) | Historical, superseded by V1: provisional direct integration, reusable containers and targeted retest; small scenarios with a plan-plus-Agent comparison; mandatory model-assessment method and integration; closed-Run gap controls; later matched comparison |
| [Claude Apply hardening handoff](claude-apply-hardening-handoff-20261020.md) | Historical, superseded by V1: active counterexample review, tool-produced receipts, previous-proposal dispositions, preserved candidates and a finite Haiku retest; the [earlier implementation handoff](claude-apply-implementation-handoff-20261020.md) is historical |
| [Codex cockpit organization chart handoff](codex-cockpit-org-chart-handoff-20261021.md) | Historical organization chart assignment and startup prompt; retired from V1, no current implementation authority |
| [Claude cockpit implementation handoff](claude-cockpit-handoff-20261020.md) | Historical Cockpit implementation assignment; retired from V1, no current implementation authority |
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
