# P02 — Shared concrete contracts

P02 supplies compiling contracts and a small application seam for independent P03/P04/P05 work. It implements neither an adapter nor a workspace. Keep the current package paths until the parallel streams are integrated; P08 owns any later source move.

## Accepted APIs

| Contract | Current path and API | Boundary |
| --- | --- | --- |
| Operations | internal/host/projectapp/operations.go: Operations{Host, Invoker}; typed Plan/Run/Apply/Preflight/FullVerify/Deliver requests | Explicit root, revision and existing domain types. Plan rejects conflicting explicit revisions. Existing reports, errors, authority and context cancellation pass through. CLI plan/status/read-only repair/preflight already use it. |
| Orchestration | internal/host/projectrun/types.go: Host, Invoker.Run/Fingerprint and domain reports | Existing Plan/Run/Resume/Repair/Review/Verify/PreflightApply/Apply/Deliver own durable state and candidate gates. MCP calls product services rather than CLI subprocesses. |
| Workspace | internal/host/projectworkspace/contracts.go: Service.Prepare/Harvest/Close; Request, Handle, Delta, Change, Limits, NormalizeDelta | Exact repository/root/full base SHA/task/WIP overlay/base inventory digests. Explicit owned paths and excluded foreign scopes; empty allowed scope permits no writes. Service implementation and real history are P03. |
| Delta | Same package, add/modify/delete/rename and Git modes 100644/100755, byte content | Binary-safe, canonical sorted digest, both rename paths authorized, duplicate/alias/control-plane paths rejected, finite count/byte bounds. P03 harvest must validate actual base/final inventories; typed changes alone do not prove existence. |
| Invocation | internal/host/agentexec/invocation.go: PrepareInvocation, DecodeResponse | Fresh identity/nonce, normalized input digest and the same closed role/evidence/native-work validation. Existing process adapter uses these helpers. Output schema does not replace validation. |
| Workspace bridge | agentexec.RunOptions.Workspace, RunResult.Delta; projectrun.Host.Workspaces | Delta comes from Host observation, never model Response. The process adapter rejects a workspace handle before launch. Current projectrun work/review/verifiers reject an unconsumed delta while preserving its receipt. Candidate integration and cleanup are P03/P06. |
| Agent evidence | internal/host/agentexec/lifecycle.go: Receipt.Lifecycle, requested/effective settings, session/turn IDs, root/child start requests, state, event-log digest/accounting | Failed starts count as requests. Nil/partial/unavailable is unknown, not proof of zero helpers or complete telemetry. Current process adapter emits no lifecycle records. P04 observes events; P06 persists/reconciles and enforces aggregate limits. |
| App Server config | internal/host/codexappserver/contracts.go: concrete Config, helper policy, Validate, Digest | Explicit executable/version/model/effort, finite timeout/event bounds and helper-request/depth ceilings. Empty permission profile inherits existing user policy. Semantic config digest alone does not fingerprint executable or instruction bytes. |

The model response stays v1 with UTF-8 candidate files. Do not add delete/binary response fields and then let the existing candidate path ignore them. P03/P06 connect the observed typed delta to the current byte/Delete-aware candidate and guarded Apply machinery. Rename can apply as delete-old plus write-new while retaining rename provenance.

## Implementation ownership after exact source handoff

| Owner assigned by Root | Exclusive new implementation | Integration-owned shared paths |
| --- | --- | --- |
| Fresh P03 workspace chat | Service implementation files/tests under internal/host/projectworkspace; exact repository/history/WIP acquisition | Existing contracts.go/test, projectrun Host/wiring, agentexec DTOs and guarded candidate/Apply integration |
| Fresh P04 App Server chat | Transport/client/session/event files/tests under internal/host/codexappserver | Existing config contract/test, runtime selection, invocation/lifecycle DTOs, durable accounting and aggregate resource gates |
| Fresh P05 MCP chat | internal/host/mcp; protocol schemas/handshake/tool mapping/cancellation and command entrypoint | projectapp operation contracts, shared CLI/service extraction, dependency files and composition roots |
| Fresh Product Integration chat | Shared integration, queue statuses and P06–P10 | Canonical source/artifact owners, dependency changes, runtime/facade wiring, docs, supported-platform gates, actual acceptance ledger and normal PR/Main path |

Each owner starts a separate feature worktree from the same accepted full pushed handoff SHA and returns focused tests/review plus a normal pushed source. Shared-contract changes require integration-owner coordination; do not copy orchestration into adapters. Product IDs remain distinct from MCP task IDs and Codex thread/turn IDs. Recovery preserves completed work and uncertainty; cancellation reaches existing context-aware services and retains durable run handles.

The facade covers the current execution lifecycle. Typed setup, exploration and Brownfield services already exist; P05/integration lifts remaining CLI-local record/orchestration handling into concrete shared application operations before publishing corresponding MCP tools. P02 does not claim complete MCP coverage from ten wrappers.

## Protocol basis and acceptance limits

Read-only inspection used installed Codex CLI 0.162.0 protocol schemas and the [official App Server documentation](https://learn.chatgpt.com/docs/app-server). Fresh Manager/reviewer contexts use distinct threads; persist thread/turn handles and observed collaboration/token events. Missing telemetry remains unknown. Actual initialization, tool work and cancellation need P04/P06 evidence. MCP mapping follows the [2025-11-25 lifecycle](https://modelcontextprotocol.io/specification/2025-11-25/basic/lifecycle), [tools](https://modelcontextprotocol.io/specification/2025-11-25/server/tools) and optional experimental [Tasks](https://modelcontextprotocol.io/specification/2025-11-25/basic/utilities/tasks). These readings and schema exports started no provider jobs.

Acceptance is compiling concrete seams, focused contract tests, independent source review and clear exclusive ownership. Exact results and accepted source are in the [source handoff](source-handoff-20261009.md). P03 onward remains pending fresh-chat dispatch. No additional native allowance was consumed.
