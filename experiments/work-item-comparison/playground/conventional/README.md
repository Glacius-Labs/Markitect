# Conventional execution wrapper

Source preparation only. Both examples are disabled. No Codex, model, provider or
installed App Server was invoked to validate this wrapper. Offline fixtures use
scripted local processes; they cannot prove runtime availability, useful agentic
coding, CLI behavioral equivalence, cost, or human acceptance.

The wrapper controls an ordinary Conventional actor. It passes the supplied user
request unchanged and inherits the native runtime's project instructions, tools,
Skills, configuration and helpers. It introduces no canonical model, generated
guidance, engineering roles, solution planner, final-response schema or automatic
repair. The actor owns its decomposition, helpers, branches, tests and integration.
The wrapper never installs/configures Markitect or mutates the case's source itself.

## Two independent choices

| Controller entrance | Native execution backend | Prepared path |
|---|---|---|
| Wrapper CLI | `codex-cli` | `codex exec --json`; own session continued with `exec resume` |
| Wrapper CLI | `codex-app-server` | stdio initialize, own thread start/resume and ordinary turns |
| Wrapper MCP | `codex-cli` | The same service and CLI backend, accessed with lifecycle tools |
| Wrapper MCP | `codex-app-server` | The same service and App Server backend, accessed with lifecycle tools |

MCP is a controller entrance here; App Server is a native execution backend. They
can be combined. Markitect's own MCP additionally exposes product operations to
an actor; Markitect's own App Server schedules genuine product roles. Those
method mechanisms and their costs remain part of Markitect. This wrapper exposes
only run control and never copies their domain contracts. The normal Codex CLI
user flow remains the scientific reference. No custom App Server protocol server,
deprecated Codex MCP server, SDK backend or Claude backend is supplied here.

Use the same outer entrance/backend in a method pair, or disclose the difference
and restrict interpretation. Matching entrypoints alone does not establish matching
context, permissions, tools or helper behavior. A later bounded transport comparison
may select these paths with method, case and runtime fixed; preregister its question,
order and finite allowance separately. These optional paths do not create an
automatic model × method × transport factorial study or authorize starts.

## Shared lifecycle

`service.py` owns admission and durable attempts. `backends.py` owns native wire
handling. `mcp.py` is a stdio facade; `../conventional_wrapper.py` is the CLI
composition root. Start accepts one ordinary request and returns a wrapper UUID.
Resume accepts that UUID plus the next user request and returns a new attempt UUID
for the same native session/thread. Resume the latest completed continuation, never
an ancestor. Status reads durable state; cancel requests interruption, which is
distinct from a proved terminal state. Native terminal success means the turn
completed; task assessment stays `NOT RUN` and human acceptance remains unestablished.

MCP tools are `conventional_start(prompt)`,
`conventional_resume(run_id, prompt)`, `conventional_status(run_id)` and
`conventional_cancel(run_id)`. Initialize MCP before tool calls. Tool arguments
cannot change cwd, model, permissions, executable or deadlines. Tool results are
wrapper metadata, never a response DTO taught to the implementing actor.

CLI start/resume publish the handle, then remain alive until the native dispatch
finishes. Status/cancel may be called from a separate local controller with the same
configuration. MCP owns its active workers until disconnect; disconnect/Ctrl+C
requests cancellation and performs bounded cleanup of the direct child. The
wrapper reports `ownedProcessScope: direct-child-only`; it does not prove native
helper/descendant termination. One outer turn may execute at
a time per audit store. This does not serialize the actor's own native helpers.

Prospective command shapes, after a later authorized external binding:

```powershell
C:/Python313/python.exe -B experiments/work-item-comparison/playground/conventional_wrapper.py --config ABSOLUTE_CONFIG start --prompt-file ABSOLUTE_PROMPT
C:/Python313/python.exe -B experiments/work-item-comparison/playground/conventional_wrapper.py --config ABSOLUTE_CONFIG resume WRAPPER_UUID --prompt-file ABSOLUTE_NEXT_PROMPT
C:/Python313/python.exe -B experiments/work-item-comparison/playground/conventional_wrapper.py --config ABSOLUTE_CONFIG status WRAPPER_UUID
C:/Python313/python.exe -B experiments/work-item-comparison/playground/conventional_wrapper.py --config ABSOLUTE_CONFIG cancel WRAPPER_UUID
C:/Python313/python.exe -B experiments/work-item-comparison/playground/conventional_wrapper.py --config ABSOLUTE_CONFIG mcp
```

## Frozen inputs and receipts

Configure explicit absolute executable argument array, cwd and separate audit;
model/effort; positive finite turn/request timeouts; safe runtime binding metadata;
and SHA-256 `filePins` including the executable, wrapper CLI/service/MCP/backend
source and any launcher/fixture files. The complete pair freeze still owns case,
setup, evaluator, effective runtime, tools/rights and business-input bindings;
this guard is not complete freeze verification.
No shell command construction, auth probe, alternate provider key, global setting
change, helper-disabling instruction or automatic sandbox bypass is used. Native
approval/user-input requests require a human-capable path; unsupported requests
are preserved rather than automatically accepted. Requested model/effort are not
proof of effective serving. Raw protocol can contain sensitive workspace data;
the external audit belongs on the private research remote, outside actor context.

A future human order must be translated into an external order file with
`execution_authorized: true`, a reviewable `actualOrderRef`, exact configuration
byte SHA-256, prepared lifecycle `runId`, timezone-aware `expiresAt` and positive
`maxTurns`. The config must explicitly enable execution and point to that order.
The guard binds files and finite outer attempts; it does not authenticate a human
or prove native helper/OS-wide budgets. The two example files are not grants.
Timeout examples allow 90 minutes per turn and 10 minutes per protocol request;
the later order owns actual windows and can increase them. Protocol waiting is
bounded separately from healthy long turn execution. Order expiry limits a turn's
remaining wall window. Every accepted attempt, including failed dispatch, consumes
one outer turn. Native executing starts, concurrency/depth, aggregate usage and
method-internal work must be bound/measured separately in the pair freeze.

`AUDIT/conventional-execution` contains immutable config/order/run binding,
copies of config/order, per-attempt `request.json`, raw `events.jsonl`, `result.json`, atomic current
`status.json`, cancellation receipt and continuation link. Parent request/results
are retained. Exact native session/turn IDs, terminal state, partial/unknown usage,
errors and controller timestamps explain the observation. Original pipe/client
wire bytes are retained as base64 with byte length; decoded text is a convenience
view. Request receipts include requested runtime metadata and current prepared-run/
station-control hashes; none establishes loaded/effective context. The controller never
declares product quality or a source snapshot complete from these receipts; use
the existing independent lifecycle snapshot/freeze and common assessment.

A dropped controller may leave a native turn or helper unresolved. A different
process reading a running status reports ownership as unverified and never replays
it. `uncertain`/`needs_input` retain `active.json` and block further dispatch.
There is no automatic orphan cleanup, lock deletion, recovery replay or retry.
Reconciliation requires a later explicit owner decision with native handles and
evidence; this initial wrapper does not implement interactive approvals or an
uncertain-session recovery editor. Preserve the candidate and receipts meanwhile.

## Protocol sources and validation scope

The native backends follow the official [non-interactive Codex guide](https://learn.chatgpt.com/docs/non-interactive-mode)
and [Codex App Server protocol](https://learn.chatgpt.com/docs/app-server).
The MCP facade follows the [MCP lifecycle](https://modelcontextprotocol.io/specification/2025-11-25/basic/lifecycle)
and [tools specification](https://modelcontextprotocol.io/specification/2025-11-25/server/tools).
Source contracts were consulted in Product Integration at
`3523f168f3af11fad19a3bedfaf181f5429d06a5` as interface context only;
no product files were copied, imported, modified or declared installed/ready.

[Wrapper validation](validation.md) binds the exact offline checks and limitations.
The [study contract](../study-contract.md) continues to own scientific conclusions.
