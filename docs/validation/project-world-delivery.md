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

The final implementation source is `950332a9bc621153e70372f5f97a28b3365644c4`, checked locally on Windows with Go 1.27.1. The subsequent report and plan correction are prose-only changes; they do not acquire a new full-suite result from this run.

| Gate | Tested source | Result |
|---|---|---|
| `go test ./... -count=1 -timeout=30m` | `950332a` | PASS, 52 packages with tests, including the architecture import gate and historical compatibility suites |
| `go vet ./...`, `go mod verify`, native CLI build | `950332a` | PASS |
| Schema freshness, root format, fixed root `check` and engineering-change `context`, impact from the recorded base | `950332a` | PASS |
| Managed-artifact accounting and configured hooks/pipelines | `950332a` | PASS; one pipeline and six declared check references |
| Contribution examples: eight checks, five formats, four models and two contexts | `c57a1de8068a0006e8bfd44da98dfc91e9b1bb40` | PASS; these example inputs are unchanged by the final compatibility fixes |
| Codex and Claude adapter protocol suites | `c57a1de` | PASS, 20 Codex and 6 Claude tests; deterministic protocol tests, no provider inference |
| Native initialization and Shop walkthrough | `c57a1de` | PASS, described below; this is evidence from that candidate binary |

The integrated `projectrun` tests use the real `agentexec` subprocess transport. They exercise separate manager invocations, delegation and integration, interruption and resume without replay, literal check execution, stale-apply rejection and successful guarded Apply. Negative subprocess cases cover foreign paths, stale response nonces, out-of-scope proposals and failed integration. The full suite also includes fixed-commit Brownfield discovery, contradictory evidence and clarification, deferred-scope exclusion, reviewed runtime setup and refusal to plan over uncommitted selected inputs.

The native CLI walkthrough initialized a fresh unborn feature branch: preview created no files and write created only Markitect-owned files. A separate committed Shop copy passed `project check/index/context/document` and all seven Python/SQLite behavior tests. An exact-digest model edit was previewed, applied and committed, then impact reported one changed definition, eight affected Statements, six Managers, five files and one check, with no unknown scope. These finite fixture results establish the demonstrated flow, not arbitrary project completeness.

The earlier full run at `c57a1de` failed in legacy controller/inference tests. The final source restores the historical nil-environment-allowlist fingerprint while retaining actual environment receipts; explicit allowlists in the new project path still bind their effective values. Legacy inference now chooses an absent private log leaf beneath its configured container so `agentexec` can create the required ACL without weakening checks or altering container permissions. Those regressions pass in the final full run. Two earlier coordinator-interrupted runs are not counted as passing evidence.

An independent final documentation review found the overstated container-launcher plan entry, which was corrected to match the implemented boundary. The final prose candidate is checked separately with fixed `check/context/impact`, artifact/module accounting, local document links and `git diff --check`. This report records standalone gates; it is not a new whole-Project `verify` receipt.

Local raw logs are retained outside repository inputs under `%TEMP%/markitect-project-world-validation/`, including `integrated-full-tests.log`, `failed-c57a1de-full-tests.log` and the `final-*.log` files. They are local evidence, not published CI artifacts.

## Remaining boundaries

`controlled-local` constrains the product protocol and Apply path; it cannot prevent a separate process with the user's OS authority from modifying the repository or external systems. Requested `isolated` mode fails closed. A verified container/VM launcher, isolated credentials/egress and an independent Apply broker remain future implementation.

The lifecycle evidence uses deterministic fake executors through real subprocess transport. It does not establish provider reliability or autonomous implementation quality. No paid/live Codex or Claude inference was run for this candidate. The installed Claude CLI observed during development was 2.1.233; the restricted adapter requires a compatible version at least 2.1.248 and rejects unsupported versions.

P1 Distillation validates a supplied report; it does not automatically discover business meaning or invoke a provider. Passing compilation and finite checks does not prove every comment, document or implementation fact is consistent, complete or correct. The larger acceptance catalogue in the design remains a requirement list, not a claim that every scenario ran. Hosted Windows/Linux CI, immutable packaging/release gates, OS isolation and human acceptance are separate evidence.
