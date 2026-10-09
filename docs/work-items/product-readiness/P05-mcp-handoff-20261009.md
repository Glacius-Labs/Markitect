# P05 MCP adapter checkpoint — 9 October 2026

Owner: fresh P05 chat, branch `codex/product-mcp-20261009`, isolated `product-mcp` worktree. Base: `1495e1be7b0046711531fa42c8407fba67b8e814`. Implementation checkpoint: `e0f727520ccfc2b10b9f613235967ff0b220b7c9`. The publication commit adds this handoff; its full remote SHA is reported in the material completion callback. This is a module checkpoint, not closure of integrated P05 or product readiness.

Classification: implementation of the already accepted P05 intent. Existing canonical sources, CLI composition, shared contracts, dependencies and central backlog are unchanged.

## Files and API

- `internal/host/mcp/tools.go`: `New(root string, projectapp.Operations) (*Server, error)`, deterministic tool inventory, direct typed operations, sanitized structured results and partial durable reports.
- `internal/host/mcp/schema.go`: JSON schemas from actual DTO JSON tags, recursive closed struct inputs, required-field/unknown-field/type/null/duplicate-key checks. Dictionaries remain typed dictionaries. JSON Schema reflects nullable Go slices/maps/pointers and custom duration/time representations.
- `internal/host/mcp/stdio.go`: `Server.Serve(ctx context.Context, in io.Reader, out io.Writer) error`, newline-delimited local JSON-RPC, initialization, capability negotiation, tools/list, tools/call, ping, cancellation and disconnect handling.
- `internal/host/mcp/mcp_test.go`: bounded protocol fixtures and shared Host selection/error tests. No provider execution.

Integration composes the existing Host/Invoker and explicit root, calls `mcp.New`, then `Serve` with protocol-only stdout. Close input on external process shutdown so a blocking reader unblocks. Write diagnostics to stderr. No adapter command, subprocess bridge, provider runner or alternate workflow engine exists here.

`Register[T,R](server, name, description, mutation, func(context.Context,T)(R,error))` supports additional **actual shared application operations** during composition before Serve starts. Integration owns the new setup/explore/readiness/Brownfield seams and can bind them with this API or request bounded P05 bindings once concrete signatures are available. Do not expose a generic action dispatch or a placeholder implementation as a live product tool. Composition must bind the selected root itself; do not put a caller-selectable root in additional tool DTOs.

## Current tools

| Tool | Shared operation | Writes/executes |
| --- | --- | --- |
| project_plan | Operations.Plan | May persist authorized plan; caller explicitly supplies executeAuthorized |
| project_run | Operations.Run | Executes durable plan |
| project_resume | Operations.Resume | Continues same run |
| project_repair | Operations.Repair | Existing repair semantics |
| project_status | Operations.Status | Read-only |
| project_verify | Operations.Verify | Executes checks and persists evidence |
| project_full_verify | Operations.FullVerify | Executes checks; write selects persistence |
| project_preflight | Operations.PreflightApply | Read-only exact Apply inputs |
| project_apply | Operations.Apply | Existing complete candidate/review/freshness guards |
| project_deliver | Operations.Deliver | Existing durable delivery orchestration |

Tool inputs are the existing operation request DTOs; run-family inputs use `runId`, preflight also uses `candidateId`. Repository selection is fixed at server construction. Tool arguments cannot override root/repo/revision selection outside the actual operation DTO. Apply requires all seven shared guard inputs. Authority is existing caller authority; neither MCP annotations nor protocol success grants broader authority.

Results are closed envelopes `{operation, data?, diagnostic?}` inside both `structuredContent` and serialized JSON text. Partial reports retain product run/candidate IDs on Host failure. Raw Host errors are not forwarded; diagnostic codes include host_rejected, cancelled, busy and encoding_failed, with a bounded recovery instruction. Nested raw stdout/stderr/error/command/executablePath/persistedPath/root values are blanked before either representation is returned. These normalized views are not digest recomputations or byte-identical raw CLI reports; digests identify the original Host records, which remain locally stored. Product-relative ownership paths, status, findings, review facts and durable IDs remain data. No claim is made that arbitrary project-authored/model-authored prose is secret-free.

## Protocol and lifecycle boundaries

The adapter follows the official [2025-11-25 lifecycle](https://modelcontextprotocol.io/specification/2025-11-25/basic/lifecycle), [tools](https://modelcontextprotocol.io/specification/2025-11-25/server/tools) and [cancellation](https://modelcontextprotocol.io/specification/2025-11-25/basic/utilities/cancellation) contracts. Version negotiation accepts 2024-11-05, 2025-03-26, 2025-06-18 and 2025-11-25; an unsupported requested version receives the current version for client negotiation. Only tools capability is advertised, without listChanged. The fixed small inventory needs no pagination; nonempty cursors are rejected. Resources and experimental MCP Tasks are not advertised because no current shared contract requires them.

Long calls use the original MCP request context, while product handles belong to existing durable Host state. Plan returns the handle used by run/status/resume/repair. Pending long calls allow independent status and ping requests. Cancellation reaches context-aware Host operations and returns a structured failure/partial report. Plan, Status, Preflight and Apply have synchronous shared signatures and retain their existing interruption limits. An MCP request ID never replaces a product run ID or provider thread/turn ID. There is no adapter-only persisted job store or false process-restart recovery claim.

At most 16 concurrent active tool requests are admitted; mutations on this server are rejected as busy while another mutation runs. This is admission control, not cross-process locking or a substitute for Host freshness guards. Input frames are bounded at 4 MiB. Disconnect cancels pending contexts and waits for Host calls to settle. Only advertised tools are available; setup/explore/Brownfield are rejected explicitly until real shared bindings are added.

## Checks and independent review

- `go test ./internal/host/mcp -count=1 -timeout=2m` passed after all fixes (1.044 s). Tests cover handshake/required client metadata, tool discovery, unknown/unavailable methods/tools, closed root/unknown/case/null/missing/duplicate/nested inputs, complete Apply input shape, active cancellation, status during work, concurrent mutation rejection, sanitized errors and partial durable handles, all existing lifecycle output DTO schemas and unsigned integer boundaries.
- `go vet ./internal/host/mcp` and `git diff --check` passed.
- `go test ./internal/tooling/architecture -count=1 -timeout=2m` passed (0.759 s).
- Fixed-source `markitect check` and `impact` passed at implementation SHA against the stated base. `markitect-check-artifacts` passed, but its explicit inventory does not list the new mcp files: Integration must add their canonical artifact bindings/owners when incorporating this module and handoff. This pass does not establish coverage of these new files.
- Independent bounded developer source review examined protocol, authority, schema, delegation and sensitive output. Required initialize metadata and raw nested report exposure were fixed and re-reviewed as addressed. A later uint64 validation edge was fixed with a focused regression. No measured product roles were started by implementation or review.

The actual Git fixture test reaches the same real projectapp/projectrun selection path as direct Host calls and preserves exact root/base selection before an injected frontend failure. Handshake/cancellation fixtures are protocol stubs, not native success. Existing Host guard tests remain the authority for stale base/run/candidate/review/concurrent checkout semantics; this adapter does not copy those engines. Full CLI/MCP lifecycle parity on an integrated executable and actual Shop fixture remains Integration/P09 work.

## Integration remaining

Add the CLI entrypoint and local provider configuration guidance through Integration-owned composition; register the actual newly extracted setup/discovery/model/check/Brownfield operations; add canonical artifact ownership for the five new paths; run focused integrated fixture parity and supported-platform CI. Runtime/native acceptance, human acceptance, installed release availability and Main integration are not proved by this checkpoint. No Main merge, release, provider job, purchase, credential change, old Classic/Government work or private evidence access occurred.
