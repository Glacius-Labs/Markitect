# P04 native App Server transport handoff — 9 October 2026

Implementation commit: **ac6a04d4b1b2912b84b4e7d9fb4dd848762352bc**, based on **1495e1be7b0046711531fa42c8407fba67b8e814**. Branch: `codex/product-app-server-20261009`. This document is exclusively P04-owned. Integration owns the central queue, shared DTOs, runtime selection, workspace services, aggregate limits and final product acceptance.

The source adapter and focused protocol tests are ready for wiring. This is not native acceptance, Main readiness, an installed release, or human acceptance. No real model/provider turn, acceptance job, new key, installation, release, merge or protection bypass was performed by P04. Protocol fixtures use the Go test executable as a fake server; they exercise actual process/stdio handling without running Codex inference.

## API and ownership

`internal/host/codexappserver.NewAdapter(Config, Options) (*Adapter, error)` returns an immutable configured adapter. Its `Run(context.Context, agentexec.Config, agentexec.Request, agentexec.RunOptions)` and `Fingerprint(agentexec.Config)` signatures satisfy `projectrun.Invoker`. Every Run opens its own owned process and fresh `thread/start` context; it never shares a Manager conversation or forks history. Manager/reviewer scheduling remains the existing Host's responsibility.

Run requires an owned `RunOptions.Workspace` with a real available CWD and BaseSHA equal to the explicit request SourceRevision. It uses that CWD for the native process, thread and turn. The supplied workspace service remains responsible for ownership validation, Git/history, observing full byte deltas and cleanup. The adapter neither creates a second controller nor harvests model-authored deltas. Response validation uses the shared `PrepareInvocation`/`DecodeResponse` binding; `WorkspaceMode` is empty for the full Host workspace contract. The earlier scoped NativeWork contract is not reinterpreted as binary/delete support.

Shared invocation configuration must have the same absolute native executable, provider version, model and timeout as the adapter Config. Args must be empty; nonempty ModelOptions are rejected, because explicit adapter Config owns effort/profile/helper settings. Shared stdout/stderr/runtime-file/environment limits are validated through `agentexec.Fingerprint`. Its current ten-minute per-invocation timeout bound still applies; the central 1,800-second job allowance does not silently change that contract. `InputRoots`, `TempParent` and `PrivateLogDirectory` do not select additional App Server workspaces or private logs. Host workspace observation and the explicit callbacks supply those integration responsibilities.

`Options` has these synchronous integration hooks:

| Hook | Contract |
| --- | --- |
| `BeforeStart(ctx, RoleStartRequest) error` | Reserve the root role request durably before `thread/start`. Failed reservations dispatch no thread. Native collab requests are observed later and do not call this hook before dispatch. |
| `OnHandle(ctx, RecoveryHandle) error` | Persist a copied, nonce/input/config/workspace-bound handle after thread creation, before turn dispatch, and after turn-ID observations. A failed pre-dispatch journal write sends no turn. |
| `OnEvent(ctx, Event) error` | Persist private exact incoming wire bytes: notifications, server requests, and `rpc/response` including effective approval/sandbox values. Incoming source evidence can contain sensitive project content. A synthetic `markitect/instructionSources/observed` entry has Params but no server Wire. |
| `DynamicTools`, `HandleToolCall(ctx, ToolCall) (ToolResult,error)` | Explicit function tools owned by Host. `item/tool/call` passes owned thread/turn/call IDs, selected tool and arguments. Host validates arguments, reserves helper starts and executes any independently scoped helper. |
| `MaxToolCalls`, `ToolTimeout` | Mandatory finite tool-call count and handler timeout when tools are selected. Duplicate call IDs are rejected. Specifications and limits are included in Fingerprint. |

Hooks must honor their context and must not recursively use the same live Client. Separate Adapter.Run calls create independent connections. A dynamic handler that starts an independent helper must use the Host's existing task/scope/ownership rules and aggregate ledger. Do not count a root helper twice by reserving both in the handler and in a second unconditional BeforeStart callback; Integration owns consistent reservation identity. The transport adds no Manager/helper task graph.

`Adapter.Recover(ctx, sharedConfig, RecoveryHandle)` reopens only a trusted, exact Host-journaled invocation using `thread/resume`, then reads its exact saved turn with `thread/read`. It sends no new user input and never calls `turn/start`. A missing turn ID after a lost start reply, missing turn or running turn stays uncertain. Preserve the original attempt's receipt/start journal when adding recovery observations: recovery's partial receipt is not replacement evidence that earlier starts did not occur. Never accept recovery handles from model output or untrusted external input.

## Protocol, executable and permissions

The actual local CLI reported **codex-cli 0.162.0**. The normal npm `codex.ps1` entrypoint wraps Node; execution requires the direct installed native `codex.exe`, whose local path was resolved under `@openai/codex-win32-x64/vendor/x86_64-pc-windows-msvc/bin`. No credential file was read. Version/schema inspection alone does not prove authentication or account/model access.

The following schemas were generated read-only from that installation. Their identities are bound into the adapter protocol fingerprint:

| Generated v2 bundle | SHA-256 |
| --- | --- |
| `codex app-server generate-json-schema --out DIR` | `0bf5254bede109d4ae03ce2e81372e4c93a30b359c0749ec7dce7a9382a7f857` |
| Same command with `--experimental` | `5c0ee37a723e4672108aa5c68b0c63cd540e17d408ca2f8ac1e7ebf76e737ac2` |

There is no invented protocol-version handshake field. Every Run/Recover probes `--version` with a finite bound and rejects mismatches. Fingerprint binds provider/protocol, adapter Config, shared executable bytes, declared runtime pins, selected environment contract and dynamic tool specs/limits. Receipt records executable, command, runtime-file and selected-environment digests; it never serializes environment values. Candidate mutation is rechecked before success.

The adapter implements initialize/initialized, fresh thread creation, normal turn start, streamed item/turn/settings/usage events, interruption, owned resume and history inspection. The method/field shapes were checked against the generated schemas and the [official App Server reference](https://learn.chatgpt.com/docs/app-server). It inherits the caller's normal authentication/configuration and native shell/Git/test/helper capability; it injects no broad sandbox/approval flags, replacement base instructions or helper-disabling feature flags.

An empty PermissionProfile inherits existing user settings. A selected nonempty named profile uses the generated experimental `permissions` field and requires the returned `activePermissionProfile.id` to match before a turn starts. Experimental capability is enabled only for explicit profile selection or explicit dynamic tools. The actual approvalPolicy/sandbox values are available as raw RPC response evidence through OnEvent. Shared SessionSettings has no separate approval/sandbox fields; absence of profile provenance remains unknown. Unknown approval, authentication, input or tool server requests fail closed with a user-decision error; the adapter never fabricates approval, answers or refreshed tokens.

Server-reported instruction paths are checked as absolute regular bounded files: at most 64 paths, 4 MiB each and 16 MiB combined, with context checks and post-turn stability verification. The server reports paths, not loaded byte hashes. The callback labels the observed current-byte digests separately; Effective.InstructionDigest remains empty rather than falsely proving the exact bytes loaded by Codex. Declared instruction/runtime pins must still be selected by Integration when constructing the candidate runtime.

## Lifecycle, bounds and native helper limits

Thread, session and turn IDs remain separate. Start responses alone do not prove inference; terminal success requires the matching `turn/completed` status and a shared-protocol-valid Response. Failed requests, interruption, malformed responses and bounded overflow remain errors with receipts. A lost request/transport or unconfirmed timeout after dispatch is `unknown`/ErrUncertain. The adapter attempts a bounded turn/interrupt, then closes its owned Windows Job Object or Linux process group. Process cleanup failure stays uncertain. Unsupported process-tree platforms reject startup.

Event bytes are finite across the connection, not per line; stdout response, stderr retention, wall time, dynamic-tool arguments/results/calls/time and instruction inputs also have finite bounds. No automatic retry or uncertain-turn replay exists. OnHandle must be connected for durable recovery; without it only the returned receipt is retained. Host should persist OnEvent before allowing measured starts.

The [official configuration reference](https://learn.chatgpt.com/docs/config-file/config-reference) defines `agents.max_concurrent_threads_per_session` as concurrent spawned children, excluding the primary thread. The adapter sends the configured helper limit directly, with no extra root slot. An earlier bare binary-string observation of `max_depth` did not establish a supported config key; **the adapter does not send `agents.max_depth`**. Depth is only an observed limit here.

Native `collabAgentToolCall` spawn/resume events count observed requests, including failed requests, without double-counting item/started and item/completed. Receiving thread IDs are not mislabeled session IDs: session identity stays unknown until the actual child thread event reports it. Parent/session links, child states and usage wire events are retained when available. Root receipt Usage is explicitly `codex-app-server/thread-total`, not aggregate helper/cost accounting. Missing usage is nil/unknown; native accounting is always partial.

**Native total helper starts and depth cannot be guaranteed before dispatch by this installed App Server protocol.** Concurrent limits do not bound lifetime failed/nested starts. Observed over-limit events trigger failure/interruption, but do not retroactively prevent the dispatched start. Host-reserved dynamic helper calls provide a concrete pre-launch seam for Integration's P06 ledger; they do not intercept arbitrary native collab starts. A02 needs an honestly enforceable finite central plan, rather than relabeling post-hoc observations as a hard 64-start guarantee. This limitation was reported once to Overseer and materially clarified to Integration.

## Checks, review and remaining wiring

On implementation commit ac6a04d4:

| Check | Result |
| --- | --- |
| `go test ./internal/host/codexappserver ./internal/tooling/architecture -count=1 -timeout=2m` | Passed; adapter 5.363 s, architecture 0.330 s |
| `go vet ./internal/host/codexappserver` | Passed |
| `git diff --check` | Passed |
| `go test ./... -run '^$' -count=1 -timeout=3m` | Passed before the final narrow identity/profile/concurrency corrections; compilation only, not a full suite |
| `go test -race ./internal/host/codexappserver` | NOT RUN: this environment has CGO disabled, and Go rejected the race flag |
| Fixed BASE check/context | BASE structural check passed; engineering-change context acquired from the fixed revision |
| Artifact accounting | Failed only for the new P04 paths, which are not yet declared in Integration-owned markitect-artifacts.yaml; exact additions sent to Integration |

Fixtures cover successful terminal receipts, failed/interrupted turns, helper failed requests, malformed wire/output, incompatible version/effective model, RPC failure, event overflow, approval requests, lost dispatch, timeout with/without confirmed interruption, root reservation refusal, pre-dispatch persistence failure, explicit profile confirmation, differing thread/session IDs, child session resolution, bounded/duplicate dynamic tools, tools in fingerprint, unsupported Args and owned recovery without turn replay.

Two bounded read-only developer agents independently inspected installed protocol/configuration and reviewed transport correctness. Review fixes addressed effective-rights source evidence, instruction bounds/loaded-byte uncertainty, fresh root/child identity, pre-dispatch persistence failure and uncertain recovery cleanup. These reviews are development evidence, not product role starts or independent product acceptance.

Production additions are client.go, session.go, events.go, evidence.go, process.go, process_tree.go and platform process guards, recovery.go and tools.go; focused fixtures are adapter_test.go and client_test.go. Existing contracts.go/config tests, shared source contracts, dependencies and the central queue were not edited. Windows/Linux process ownership follows the existing agentexec guard pattern without changing its files.

Integration must declare all thirteen added package files under `bounded-agent-execution-host` plus this handoff in the canonical artifact manifest, select/configure the runtime, wire private callbacks/ledger and workspace harvesting, preserve uncertainty in the Host, and reconcile the native-helper quota limitation. Hosted CI, Linux runtime tests and A01/A02/A03 real authenticated native acceptance remain separate required gates. No native acceptance quota was consumed by P04. Full functional correctness/usability and human acceptance remain unproved.
