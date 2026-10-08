# Managed project source candidate

Date: 2026-10-08. Branch: `codex/model-driven-delivery`. Base: `a97cbd5ef3e0b22b9e6397501047a4e01dc90204`.

This is the local source implementation of the user-authorized [project-world design](../design/project-world/README.md). The [workflow](../project-workflow.md) and [Shop fixture](../../examples/project-world/README.md) are its current executable entrypoints. The earlier design specimens are retained as `.yaml.example`; they do not define the current schema. No release, remote push, deployment or adopting-project acceptance is claimed.

## Implemented boundary

- A pure `projectmodel` capability over the unchanged structural Core, with recursive Managers, concept/rule/use-case/architecture/workflow Statements, Artifact expectations and file assignments, Checks and Decisions. Namespace derivation, explicit public contracts and conservative impact remain structural checks.
- Host project loading, scoped inventory, index/context, generated documentation, guarded initialization and exact-digest edits under `.markitect/`. The user or root Manager can change inventory and runtime configuration through an edit; narrower roles cannot broaden those controls.
- Fixed-commit Discovery, evidence-grounded supplied Distillation, explicit clarification and per-scope adoption/defer decisions. Adoption changes model files and the manifest only. Actor and runtime-record claims are explicitly unauthenticated.
- Persisted implementation plans, separate manager processes, top-down delegation, bottom-up candidate integration, actionable obligations/escalations, interruption/resume, independent declared checks, optional AI verification, and exact preflight/apply bindings. Agent proposals cannot change control-plane files or bypass inventory/exclusions.
- Explicit environment selection, bounded protocol and costs, runner/check executable fingerprints, private provider logs and Windows ACL checks. Failed verification attempts are durable and cannot be replayed to reset their budget.

The source-alpha CLI exposes `project init/check/index/context/impact/document/edit/discover/distill/adopt/plan/run/resume/status/verify/apply`. A plan is read-only unless explicitly persisted; it starts no agent. Selected working inputs must equal the fixed committed basis before planning. Model/runtime edits are committed on the feature branch before a new plan. Apply preview supplies exact preconditions; write checks them again under the guarded Host boundary.

## Validation record

Final integrated source gates are pending while the independent real-process lifecycle test is integrated. Focused model, frontend, adoption, provider, CLI and runtime checks have passed during implementation; those earlier runs do not stand in for the final source gates.

## Remaining boundaries

`controlled-local` constrains the product protocol and Apply path; it cannot prevent a separate process with the user's OS authority from modifying the repository or external systems. Requested `isolated` mode fails closed. A verified container/VM launcher, isolated credentials/egress and an independent Apply broker remain future implementation.

The lifecycle evidence uses deterministic fake executors through real subprocess transport. It does not establish provider reliability or autonomous implementation quality. No paid/live Codex or Claude inference was run for this candidate. The installed Claude CLI observed during development was 2.1.233; the restricted adapter requires a compatible version at least 2.1.248 and rejects unsupported versions.

P1 Distillation validates a supplied report; it does not automatically discover business meaning or invoke a provider. Passing compilation and finite checks does not prove every comment, document or implementation fact is consistent, complete or correct. The larger acceptance catalogue in the design remains a requirement list, not a claim that every scenario ran. Hosted Windows/Linux CI, immutable packaging/release gates, OS isolation and human acceptance are separate evidence.
