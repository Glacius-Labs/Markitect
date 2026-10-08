# R1 app-server schema contract

This packet freezes protocol shapes for the separately authorized `s1-common-runner-read-tool-20261008-r1` smoke. It does not launch the pinned CLI, create an Actor, contact a model/provider, or perform a protocol call.

The governing grant is the complete `authorization-grant.json` in this packet. It binds the source coordination-state file and pointer, the approved one-Actor/one-thread/one-turn/one-native-tool scope, exact permission/model/cwd gates, event and resource limits, zero retries, stop rules, and claim boundary. The packet `profile.json`, `expected.json`, and `input-binding.json` freeze the exact prospective argv/RPC sequence, public command/prompt/file bindings, and external expected sentinel. The sentinel is not included in the prompt or this schema contract.

## Pinned schema archive

The schema archive is `experiments/government-comparison/evidence/policy-compatibility/run-1/protocol-schema.zip`, SHA-256 `cd24042eec4696f6b73368cfc6ce930726f3e10c28bae63cdbd1b2f502ff1d97`. `schema-contract.json` pins only 26 relevant member hashes: the five available request schemas, six response schemas, 13 allowlisted notification payloads, `v2/ErrorNotification.json` for the explicit error stop, and `ServerNotification.json` as the source of the finite 83-method map. The 83-entry `topServerNotification` map is method/definition metadata; unused payload schemas are not loaded or pinned. The grant's four required thread/turn member hashes match this packet.

The pinned member list is:

```text
ServerNotification.json
v1/InitializeParams.json
v1/InitializeResponse.json
v2/AgentMessageDeltaNotification.json
v2/CommandExecutionOutputDeltaNotification.json
v2/ConfigReadParams.json
v2/ConfigReadResponse.json
v2/ConfigRequirementsReadResponse.json
v2/ErrorNotification.json
v2/ItemCompletedNotification.json
v2/ItemStartedNotification.json
v2/ReasoningSummaryPartAddedNotification.json
v2/ReasoningSummaryTextDeltaNotification.json
v2/ReasoningTextDeltaNotification.json
v2/ThreadClosedNotification.json
v2/ThreadStartParams.json
v2/ThreadStartResponse.json
v2/ThreadStartedNotification.json
v2/ThreadStatusChangedNotification.json
v2/ThreadTokenUsageUpdatedNotification.json
v2/TurnCompletedNotification.json
v2/TurnInterruptParams.json
v2/TurnInterruptResponse.json
v2/TurnStartParams.json
v2/TurnStartResponse.json
v2/TurnStartedNotification.json
```

`requestSchemas` and `responseSchemas` give archive member paths, not copied schemas. The archive contains no `configRequirements/read` request-params member; the frozen RPC sends that method without `params`, and the response is validated against `v2/ConfigRequirementsReadResponse.json`. `initialized` is the only client notification and is sent with `{}`; no other client notification is in scope. `turn/interrupt` is stop-only for the known owned active turn and is never continuation.

`topServerNotification` binds the archive's full finite method union to its schema member or, for the two fuzzy-file-search variants without standalone members, to the corresponding definition in `ServerNotification.json`. `notifications` is the strict permitted subset for this smoke. A method being present in the union only means its shape is known; it does not make that method acceptable. Every method outside `notifications` stops the attempt. The explicit stop list calls out JSON-RPC/server errors, warnings, model/auth events, settings/permission changes, and server-request resolution; the default stop rule covers all other non-allowlisted methods, including approvals and tool-routing surfaces.

## Event and result contract

For thread creation, validate the `thread/start` response and any `thread/started` notification against their pinned members. Bind `thread.id`; require the response's exact requested model, provider, cwd, approval policy and effective read-only report before sending a turn. The active permission profile, if present, must be `:read-only` with no unexplained parent. If absent/null, preserve unknown provenance and continue only when the explicit effective sandbox report is read-only and does not report network access. A notification is not independent evidence of OS enforcement.

For the single `turn/start`, bind its response `turn.id` to `turn/started` and require the same `threadId`; the started status must be `inProgress`. Every turn/item/output/usage notification must carry the same thread and turn IDs. Item lifecycle events bind `item.id`; `item/commandExecution/outputDelta.itemId` must match the one permitted command item. Reject a second commandExecution item or any other tool-bearing item type.

The recognized item types are only `userMessage` (fixed-input echo; compare if surfaced, do not store arbitrary prompt text), `reasoning` (discard text), `agentMessage` (bounded final-answer candidate), and `commandExecution` (the one ordinary shell tool item). For commandExecution, retain only bounded fields needed to compare exact frozen command, actor cwd, item ID, status, exit code, source and action summary. The exact command string and correlated result are primary; `commandActions` is explicitly best-effort classification in the schema. Success needs item completion with status `completed`, exit code `0`, the exact frozen command/cwd, observed bounded output matching the external expected value, and a completed agentMessage with the fixed public JSON shape. Then require `turn/completed` for the same IDs with status `completed` and no error. `interrupted`, `failed`, an error event, mismatched IDs, missing tool evidence, or output mismatch is a terminal non-success. A final assistant self-report by itself is insufficient.

`item/agentMessage/delta` and `item/commandExecution/outputDelta` have required `delta`, `itemId`, `threadId`, and `turnId`; they are intermediate, not terminal proof. `item/started` requires `item`, `startedAtMs`, `threadId`, `turnId`; `item/completed` requires `completedAtMs`, `item`, `threadId`, `turnId`. `thread/tokenUsage/updated` requires `threadId`, `turnId`, and `tokenUsage.last`/`total`. Keep the latest cumulative usage observation; never sum repeated totals. Reasoning text and summary deltas are schema-checked then discarded, not logged. Enforce the grant's frame, raw-byte, notification, item, output and observed-token limits; the token threshold is retrospective, not an in-flight cap.

The actual ordinary task tool item is `type: commandExecution`; its streamed result uses `item/commandExecution/outputDelta`. `command/exec/outputDelta` is a distinct legacy notification, while `process/outputDelta` and `process/exited` are separate process-stream methods. None are accepted as a substitute. Do not call command/exec, process, thread shell-command, filesystem, approval, or client-side tool RPCs.

## Ordering and stop semantics

The archived JSON Schemas specify payload shape and required fields, not a state machine or event ordering. With multiplexed JSON-RPC, a `thread/started` or `turn/started` event and subsequent item/output events may be read before the matching start response. A collector may hold only bounded, schema-valid pending lifecycle events and bind them once the RPC response supplies the IDs; any mismatch, excess pending buffer, or inconsistent lifecycle stops. Deltas can precede item completion and never establish finality. The successful terminal predicate is built from correlated completed item and turn evidence, not transport arrival order.

`thread/status/changed` is accepted only for the owned thread and recorded exactly; it is not permission evidence. `thread/closed` is conditional and tolerated only after the positively completed turn. The pinned schema does not assert that status or close notifications arrive in a particular order; an event that would reopen or extend a terminal turn stops. On the first unexpected event/error/deviation, do not send an approval, tool-execution, or alternate-route response. At most one `turn/interrupt` may be sent only to stop a known active owned turn, followed by bounded cleanup.

No schema fact here proves that the OS enforces read-only access, that only one provider request is used, or that reported serving identity equals a provider's actual serving identity. The eventual claim remains limited to the one observed public-file read on the authorized route.
