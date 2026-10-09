# Project operations

This page is the current operational reference for the model-first `markitect project` source workflow. It is not in the published v0.14.1 CLI. Build/use a candidate from the matching Markitect source checkout; commands and MCP tools below do not change that release pin.

## MCP server and tool groups

Start the repository-local stdio server with `markitect project mcp --repo ABSOLUTE_PROJECT_ROOT`. The server fixes Host authority to that root at startup. MCP schemas are typed and closed; write operations enforce the exact authorization, preview digest, compare-and-swap, or durable-identity rules declared by their schemas. Consult `tools/list` for the exact current schema.

The composed project server exposes these operation groups:

- **Inspect:** `project_index`, `project_check`, `project_context`, `project_impact`, `project_coverage`, `project_status`.
- **Work Item and model:** `project_explore`, `project_readiness`, `project_edit`, `project_plan`, `project_deliver`.
- **Execution lifecycle:** `project_run`, `project_resume`, `project_repair`, `project_verify`, `project_full_verify`, `project_preflight`, `project_apply`.
- **Brownfield:** `project_brownfield`, `project_brownfield_run`.
- **Setup and views:** `project_init`, `project_onboard`, `project_setup`, `project_doctor`, `project_document`.

Read-only inspection does not start a role. Setup and doctor inspect local prerequisites; project operations start inner roles only when explicitly invoked through an authorized run/delivery path. The outer client is not itself an inner Manager.

## Ordinary Work Item

Follow [Project workflow](project-workflow.md): inspect, record a typed exploration, edit canonical model only when needed, check and establish readiness, then deliver or use the staged lifecycle. A preview is not a write. For any preview/write pair, use the digest returned by that operation and do not reconstruct it. `project_deliver` requires the existing caller's explicit execution authorization and advances the same durable run through Verify, preflight, and guarded Apply. The staged tools preserve each plan/run/candidate/verification identity; use the returned bindings.

`project_check` returns structural findings and coverage. An error result remains an error; it is not converted into a successful empty result. `project_coverage` reports nonconforming coverage as a failed operation while returning its report. Resolve those diagnostics before planning or readiness.

## CLI, setup and current roots

The CLI exposes matching operations for shell use. Read the candidate's `markitect project --help` for exact flags; the MCP operation schemas are the preferred contract for outer agents. `--repo` is the project authority root. For an MCP server, pass an absolute path so its selected root is explicit.

`project init`, `project onboard`, `project edit`, `project explore`, `project readiness`, `project brownfield`, `project setup`, `project document`, and `project apply` have preview/write or guarded-write semantics as defined by their operation. Do not apply a stale plan. Onboarding writes only repository-local guidance and skills; it does not register MCP, configure global accounts, or authenticate a provider. Setup supports the native Codex App Server runtime and requires explicit budget weights/limits; configured cost is an estimate, not billing enforcement. On Windows, `project setup` may explicitly select `--windows-sandbox-backend mxc`; that process-local setting leaves permission profiles and Managed Policy unchanged. Omit the option to preserve Codex defaults. `project_doctor` does not start roles.

## Candidate workspaces and recovery

Native work executes in an owned candidate workspace. The bridge binds each candidate to the fixed accepted source, repository identity, overlay, task, and candidate delta. The source working tree is preserved as the service baseline; the fixed accepted project snapshot supplies only the candidate overlay. Apply rechecks freshness and writes only the reviewed delta through guarded Host operations.

A run's observed root and known owned child starts must be terminal before the Host harvests/closes their workspace. Adapter lifecycle accounting remains **partial**: the report records durable Host starts and observed nested starts separately, and observed nested totals are a lower bound. It does not claim an exhaustive child census or proof that no unobserved process exists. Workspace lifecycle evidence is cooperative; it is not OS isolation.

On interruption, inspect `project_status` and the private durable run/journal. Resume or repair the same run only when the durable state identifies a known safe continuation. Unknown native outcomes are not replayed. Recovery inspects the exact saved provider configuration and already dispatched thread/session/turn; it never starts a new role or resubmits the turn. If ownership, bindings, or terminal state cannot be established, the candidate and journal remain preserved for inspection. A trusted dispatched original turn can be inspected even when an interrupted Host left the outer journal at `prepared`. Known nonterminal Host helper reservations and uncertain helper cleanup block Verify and Apply. Automatic helper-journal reconciliation is not implemented; preserve the private handle and resolve its concrete ownership before proceeding. Generic partial child telemetry alone is not such a blocker.

The prospective native acceptance limits are 60 minutes per role, four hours per job, and 256 role-start requests. They are grant/test bounds in the readiness backlog, not universal defaults, a cost guarantee, or completed acceptance. Real authenticated native acceptance is **NOT RUN**. Source-level and scripted tests remain separate from product acceptance. See the [readiness backlog](work-items/product-readiness/backlog.yaml), [integration progress](work-items/product-readiness/integration-progress-20261009.md), and dated validation linked from the [documentation map](README.md).
