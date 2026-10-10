# Documentation map

Markitect's current product workflow manages ordinary Work Items through a recursively owned YAML model and typed fixed-root operations. Start with the [project workflow](project-workflow.md), the [Project operations reference](project-operations.md), and the [executable Shop example](../examples/project-world/README.md). The [implementation roadmap](implementation-plan.md) gives a short orientation; the [Product Readiness backlog](work-items/product-readiness/backlog.yaml) owns current statuses and evidence pointers.

The current source is an integration candidate. Actual28 passed the combined A01 native smoke on source `cdd30b0efc540f151404dabe86b022275dc40d83`, including Full Verify, guarded Apply, and same-binary recovery of the original Root turn without replay. Exact-final-head hosted Linux/Windows gates and normal Main integration remain pending. This source-bound result does not establish human semantic acceptance or productivity benefit. Current attempt identities and accounting are recorded in the [acceptance ledger](work-items/product-readiness/evidence/native-acceptance-ledger.yaml) and [progress log](work-items/product-readiness/integration-progress-20261009.md), which preserve prior attempts and evidence boundaries. A02/A03 remain deferred until after Main. Case Studies stay deferred until Markitect works, then proceed only as matched studies with prospective equal resources under the recorded renewal authority; none is starting now. The published v0.14.1 binary preserves earlier Project/Domain commands and does not include this workflow. No release publication is authorized. The [production assessment](production-assessment.md) binds release claims to their original assets.

## Operate and contribute

| Guide | Use it for |
|---|---|
| [Project workflow](project-workflow.md) | Normal Work Items, model decisions, delivery, review, Apply, and recovery |
| [Project operations](project-operations.md) | Current typed MCP tools, CLI equivalents, write boundaries, and runtime limits |
| [Provider adapters](provider-adapters.md) | Connecting outer Codex/Claude clients and understanding the inner Codex App Server boundary |
| [Architecture](architecture.md) | Host composition, ownership, snapshots, candidate workspaces, and evidence limits |
| [Repository layout](repository-layout.md) | Adopting-project organization and the completed Markitect source layout |
| [Usage](usage.md) | Legacy Project/Domain syntax and retained command contracts |
| [Development guide](development/README.md) | Source checks, package boundaries, and development coordination |
| [Contribution checks](../CONTRIBUTING.md) | Required local and hosted validation commands |
| [Executable Shop example](../examples/project-world/README.md) | Model-first example and deterministic fixture behavior |
| [Product Readiness backlog](work-items/product-readiness/backlog.yaml) | Canonical statuses, dependencies, owners, evidence, and renewed per-batch acceptance policy |
| [Integration progress](work-items/product-readiness/integration-progress-20261009.md) | Current integration handoffs and technical findings |
| [P09 readiness contract](work-items/product-readiness/P09-readiness.md) and [acceptance ledger](work-items/product-readiness/evidence/native-acceptance-ledger.yaml) | Final gate contract and actual native-run accounting |

The [Product Vision](vision.md) owns the product thesis; [Operating Methodology](operating-methodology.md) describes the intended method; [Measurement](measurement.md) owns readiness and comparison procedure. These documents do not override source contracts or establish measured benefit.

## Dated evidence and history

The [historical roadmap snapshot](history/implementation-plan-before-product-integration-20261009.md) retains the prior roadmap and its pinned references. Its evidence details remain in the linked dated reports, including [model-first delivery](validation/project-world-delivery.md), [project operations](validation/project-operations-2026-10-09.md), and [native Codex work](validation/project-native-work-2026-10-09.md). Historical release, proof, and comparison records preserve their original scope and outcomes; they are not current readiness status.

The [research map](research/README.md) preserves owner-supplied proposals separately from adopted behavior. The [product design map](design/project-world/README.md) records the accepted project-model direction and its design artifacts.
