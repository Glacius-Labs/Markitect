# Development coordination

Start engineering work through [AGENTS.md](../../AGENTS.md), the [working method](working-method.md) and the relevant canonical owner. [CONTRIBUTING](../../CONTRIBUTING.md) owns commands and validation; [Architecture](../architecture.md) and the [repository layout](repository-layout.md) explain the source boundaries. The [delivery plan](../implementation-plan.md) owns current assignments and dependencies; chats and PRs hold execution evidence. Earlier handoffs and assignments are not kept on Main; the [archive index](../archive.md) lists them, and they cannot dispatch current work.

| Document | Owns |
|---|---|
| [Repository layout](repository-layout.md) | Where everything lives in this repository, where new things go and what enforces it |
| [Repository structure](repository-structure.md) | R1 inventory of every file, the accepted target layout (DEC-071), the archive mechanism and the R2 work list |
| [V1 bounded assignments](v1-workstream-assignments.md) | Writer scopes for steps 2 and 3 of the delivery plan: Run contract (D10–D13, D15, D17) and provider scope (D18) |
| [Concept conformance audit](concept-conformance-audit-20261024.md) | Supplied static audit of Main `6c7d758c` against the V1 concept; origin of A1–A6 and R1–R3, closed by concept 1.0 |
| [V1 counter-review](v1-architecture-audit-20261023.md) and [source archive](v1-source-archive-20261023.md) | Source-bound v0.17 counter-review and the verified pre-cutover archive |
| [Working method](working-method.md) | The owner-adopted practice of explicit learning, complete workflows, focused design review and short milestone accounts, and the scope of evidence |
| [Documentation maintenance](documentation.md) | Placement, navigation, canonical owners and evidence preservation |
| [Modules](modules.md) | Dependency rules and invariants of packages, static composition and the mechanical import gate |
| [Code map](code-map.md) | Every Go package with its layer, purpose and owning document; generated and checked by a test |
| [Parallel work](parallel-work.md) | Assigned-package execution, shared-file requests and evidence; the delivery plan owns scopes, dependencies and reservations |

For current product use, read [Project workflow](../guides/project-workflow.md) and [Project operations](../guides/project-operations.md). [Vision](../vision.md#the-delegated-method) owns the method and [Measurement](../guides/measurement.md) its evaluation.
