# Project operations

This page is the current operational reference for the model-first `markitect <verb>` source workflow. It is not in the published v0.14.1 CLI. Build/use a candidate from the matching Markitect source checkout; commands and MCP tools below do not change that release pin.

## MCP server and tool groups

Start the repository-local stdio server with `markitect mcp --repo ABSOLUTE_PROJECT_ROOT`. The server fixes Host authority to that root at startup. MCP schemas are typed and closed; write operations enforce the exact authorization, preview digest, compare-and-swap, or durable-identity rules declared by their schemas. Consult `tools/list` for the exact current schema.

The composed project server exposes these operation groups:

- **Inspect:** `model`, `check`, `context`, `impact`, `status`.
- **Work Item and model:** `explore`, `ready`, `edit`, `plan`, `deliver`.
- **Execution lifecycle:** `run`, `resume`, `repair`, `verify`, `apply`.
- **Brownfield:** `adopt`.
- **Setup and views:** `init`, `onboard`, `config`, `doctor`, `docs`.

Read-only inspection does not start a role. Setup and doctor inspect local prerequisites; project operations start inner roles only when explicitly invoked through an authorized run/delivery path. The outer client is not itself an inner Manager.

## Ordinary Work Item

Follow [Project workflow](project-workflow.md): inspect, record a typed exploration, edit canonical model only when needed, check and establish readiness, then deliver or use the staged lifecycle. A preview is not a write. For any preview/write pair, use the digest returned by that operation and do not reconstruct it. `deliver` requires the existing caller's explicit execution authorization and advances the same durable run through Verify, preflight, and guarded Apply. The staged tools preserve each plan/run/candidate/verification identity; use the returned bindings.

`check` returns structural findings and coverage. An error result remains an error; it is not converted into a successful empty result. `check` with `coverage` reports nonconforming coverage as a failed operation while returning its report. Resolve those diagnostics before planning or readiness.

## Why something is in the impact

`impact --explain` adds, for every element of the change impact, the reason it is included, a shortest witness path of model relations from a change, and a class: `change` (must be updated or rechecked) or `context` (only to read). The class is a priority signal. The impact and every plan built from it stay as they are. With `--manager`, the explanation shows only what that Manager may see; the impact itself stays project scope.

`context MANAGER --trace ID` also walks that Manager's knowledge graph, which is built from its context alone, and returns each node it reaches with one shortest witness path. `--direction` and `--depth` bound the walk. A node the Manager cannot see is reported as not found, exactly like a node that does not exist.

## CLI, setup and current roots

The CLI exposes matching operations for shell use. Read the candidate's `markitect help` for exact flags; the MCP operation schemas are the preferred contract for outer agents. `--repo` is the project authority root. For an MCP server, pass an absolute path so its selected root is explicit.

`init`, `onboard`, `edit`, `explore`, `ready`, `adopt`, `config`, `docs`, and `apply` have preview/write or guarded-write semantics as defined by their operation. Do not apply a stale plan. Onboarding writes only repository-local guidance and skills; it does not register MCP, configure global accounts, or authenticate a provider. Setup configures the native Codex App Server runtime by default; `--input` with a setup options record selects per-role profiles, including bring-your-own process executors (see [Provider adapters](provider-adapters.md#role-profiles)). Metered agents require explicit budget weights; unmetered agents record unknown cost. Configured cost is an estimate, not billing enforcement. On Windows, `config` defaults to process-local `--windows-sandbox-backend mxc`; the option may also be supplied explicitly. This setting leaves global Codex configuration and Windows Managed Policy unchanged. Setup applies one `:workspace` permission profile to every generated Manager, independent Review role, and Verifier. Each native role runs in its own owned workspace, and a process executor runs in a fresh empty directory and returns candidate files; Host artifact guards remain authoritative, and Review/Verify must return an empty candidate delta. Helpers inherit their parent profile. `--codex-profile` overrides the shared profile for all generated roles. `doctor` does not start roles.

Native App Server roles inherit the caller's environment through `appServer.environmentMode: inherit`. Markitect binds the effective values into the role fingerprint and receipt digest without exposing them. `Agent.Environment` remains the selected-name policy for declared-check executable resolution, not a filter on the inherited App Server environment. Ordinary process adapters retain explicit allowlists.

Within Manager work and integration turns, `markitect_start_helper` delegates scoped file authoring and returns Host-validated file changes after delivery and cleanup. Its current contract has no report-only result for standalone read-only inspection or analysis. Use normal parent tools and the parent's typed report for that work, or the existing independent Review roles for candidate assessment. Authoring helpers may still inspect context and run checks for their edits. Safe read-only and analysis-helper returns are backlog package RUN-04 in the [backlog](work-items/backlog.yaml); this capability is not implemented.

## Candidate workspaces and recovery

Native work executes in an owned candidate workspace. The bridge binds each candidate to the fixed accepted source, repository identity, overlay, task, and candidate delta. The source working tree is preserved as the service baseline; the fixed accepted project snapshot supplies only the candidate overlay. Apply rechecks freshness and writes only the reviewed delta through guarded Host operations.

A run's observed root and known owned child starts must be terminal before the Host harvests/closes their workspace. Adapter lifecycle accounting remains **partial**: the report records durable Host starts and observed nested starts separately, and observed nested totals are a lower bound. It does not claim an exhaustive child census or proof that no unobserved process exists. Workspace lifecycle evidence is cooperative; it is not OS isolation.

On interruption, inspect `status` and the private durable run/journal. Resume or repair the same run only when the durable state identifies a known safe continuation. Unknown native outcomes are not replayed. Recovery inspects the exact saved provider configuration and already dispatched thread/session/turn; it never starts a new role or resubmits the turn. If ownership, bindings, or terminal state cannot be established, the candidate and journal remain preserved for inspection. A trusted dispatched original turn can be inspected even when an interrupted Host left the outer journal at `prepared`. Known nonterminal Host helper reservations and uncertain helper cleanup block Verify and Apply. Automatic helper-journal reconciliation is not implemented; preserve the private handle and resolve its concrete ownership before proceeding. Generic partial child telemetry alone is not such a blocker.

Agent logs and run journals live in `.markitect/runs/private`, which Markitect creates owner-only: a protected owner-only access list on Windows, mode 0700 elsewhere. It refuses an existing directory that is not owner-only. Known limit for Windows users whose native runs in earlier releases created that directory: those releases gave it the parent folder's access list, so after upgrading every run stops with "existing private log directory … is not owner-only". Delete `.markitect/runs/private` once to continue. The native journals of runs interrupted before that cannot be recovered afterwards; resume such a run with the release that started it before you upgrade.

The [backlog](work-items/backlog.yaml) owns current status, and the [A01 validation record](validation/a01-native-smoke-20261010.md) summarizes the native smoke, its exercise limits and its attempt accounting. Those exercise limits are separate from configured runtime defaults and do not guarantee cost or acceptance. Source-level tests, the recorded native A01 result and post-merge gates retain their own source identities and evidence limits. See the [Product Readiness lessons survey](work-items/surveys/product-readiness-lessons-20261010.md) for known runtime limits and the [documentation map](README.md) for dated validation.
