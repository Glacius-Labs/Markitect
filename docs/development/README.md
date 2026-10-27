# Development coordination

Start engineering work through [AGENTS.md](../../AGENTS.md), the [working method](working-method.md) and the relevant canonical owner. [CONTRIBUTING](../../CONTRIBUTING.md) owns commands and validation; [Architecture](../architecture.md) owns the package layers, contracts, invariants and the import gate, and the [repository layout](repository-layout.md) explains where things live. The [delivery plan](../implementation-plan.md) owns current assignments and dependencies; chats and PRs hold execution evidence. Earlier handoffs, assignments, audits and the R1 repository structure record are not kept on Main; the [archive index](../archive.md) lists them and the verified pre-cutover archive, and they cannot dispatch current work.

| Document | Owns |
|---|---|
| [Repository layout](repository-layout.md) | Where everything lives in this repository, where new things go and what enforces it |
| [Cleanup inventory](cleanup-inventory-20261027.md) | R3b inventory of every Go package, command and maintained page as keep, merge, archive or delete, the canonical page per topic and the R3c work list (DEC-076) |
| [Working method](working-method.md) | The owner-adopted practice of explicit learning, complete workflows, focused design review and short milestone accounts, and the scope of evidence |
| [Documentation maintenance](documentation.md) | Placement, navigation, canonical owners and evidence preservation |
| [Code map](code-map.md) | Every Go package with its layer, purpose and owning document; generated and checked by a test |
| [Parallel work](parallel-work.md) | Assigned-package execution, shared-file requests and evidence; the delivery plan owns scopes, dependencies and reservations |

For current product use, read [Project workflow](../guides/project-workflow.md) and [Project operations](../guides/project-operations.md). [Vision](../vision.md#the-delegated-method) owns the method and [Measurement](../guides/measurement.md) its evaluation.
