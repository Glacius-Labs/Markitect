# Codex adapter

This package implements Apply's static `AgentProfilePreparer` and the `run.Agents` turn adapter. Bootstrap injects `NewEngineImageReader` into `NewMetadataPreparer`; constructing `Agents` is an explicit Orchestrator operation. The metadata factory receives its executable only from the selected, source-bound Runtime section. It has no PATH, environment, desktop binary or historical offline fallback.

## Static image metadata

The Runtime profile envelope is `markitect.runtime/v1` with a `codexAppServer` section. The closed section has these required fields:

```yaml
profileVersion: 2
engineExecutable: /usr/bin/docker
imageID: sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
agentdVersion: development
recordedCodexSchemaDigest: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
instructions: []
```

The two digests above are illustrative, not a usable image/profile. Use the actual immutable engine image ID and the expected build-recorded schema digest. `engineExecutable` must be an exact native absolute path on the host machine; it names the engine, never a provider executable. `instructions` is an ordered list of `{boundary, path}` coordinates for already captured source files. Missing instructions request a new complete acquisition and remain incomplete; this adapter never reads instruction paths.

### Source-bound execution settings

Version 2 remains metadata-only. Version 3 uses the same `codexAppServer` section in `markitect.runtime/v1`, retains every version 2 field above and requires these explicit additions. This illustrative section shows the complete shape; its image, schema and model literals establish no runtime availability:

```yaml
profileVersion: 3
engineExecutable: /usr/bin/docker
imageID: sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
agentdVersion: development
recordedCodexSchemaDigest: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
instructions: []
credentialRef: dedicated-runtime-credential
maxWireBytes: 1048576
session:
  version: 1
  model: explicitly-selected-model
  effort: high
  baseInstructions: ''
  developerInstructions: ''
  store: explicit-store
  cpus: 1.5
  memoryBytes: 2147483648
  pids: 192
  tmpBytes: 2097152
  network: ''
```

All fields are required, including explicit empty instruction strings and network. Numeric fields require YAML numbers, with integer types for byte counts, version and process count. The adapter validates `session` as a `SessionProfile`; model, credential reference and store are bounded nonempty literals. `maxWireBytes` is a positive integer at most 1 MiB, matching the current Codex transport limit. There are no environment lookups, interpolation, defaults or secret reads. Credential references remain opaque identifiers for the later Orchestrator resolver. Base and developer instruction strings are passed literally, including newlines and empty strings. Captured `instructions` remain separate ordered inputs to the existing context renderer; this projection never appends or duplicates them in either string.

The v3 static observation uses `markitect.codex-app-server.image-metadata` contract version 3. Apply's identity binds the whole captured Runtime profile document and selected field, captured instruction view, observation contract and actual image metadata. A change to execution settings therefore changes the static identity even when image metadata stays identical. Version 2 retains contract version 1.

`ProfileFromPrepared(prepared apply.PreparedProfile) (Profile, error)` requires complete passed preparation with an identity, exact input-binding echo, v3 contract, Codex provider, selected immutable image and matching recorded versions/schema. It uses the same private configuration decoder as metadata preparation. It returns a `Profile` holding the image ID observed during preparation, the captured credential reference and wire limit and the session settings. Missing facts, old v2 metadata, foreign contracts and mismatches refuse without partial output. Projection acquires nothing and resolves no credentials.

The Orchestrator supplies the credential resolver and builds `NewAgents` from the returned profiles. This source producer does not wire Core profile selection, establish an execution grant or prove provider/model availability. Fake-reader tests use the actual Apply consumer; they run no Docker, provider, credentials, Apply or smoke.

Preparation resolves the selected image ID, copies only `/opt/markitect/runtime.env` through `Engine.ReadImageFile`, and checks the image identity again. The engine creates a container for copying and removes it without starting it. It does not pull/build an image, start a container session, invoke either provider, probe a CLI version or export a schema.

The supported metadata format is exactly four unique LF-separated records emitted by [the runtime Dockerfile](../../../packaging/runtime/Dockerfile): `codex`, `claude`, `codexSchema` and `agentd`. Values must be bounded nonempty UTF-8 literals without control characters or surrounding whitespace. The recorded Codex version must equal `codex-cli 0.162.0`; the recorded agentd version and Codex schema digest must match the selected profile. The Claude record is retained through the whole file identity, with no assertion that this package supports Claude sessions. Unknown/duplicate records, old profile versions, tags and passthrough fields fail closed.

Apply binds the actual immutable image ID, canonical metadata path, whole-file SHA-256 and size, literal CLI/agentd versions and build-recorded Codex schema digest. The schema digest is a build assertion, not an observed schema inventory or provider handshake. File acquisition is bounded to 1 MiB. Engine errors and cancellation preserve completed observations as incomplete; known mismatches are failures. Neither outcome grants runtime readiness or execution authority.

Tests use the real Apply profile consumer with source fixtures and fake image-reader peers. Metadata bytes are synthetic examples of the repository's runtime format; no third-party schema or proprietary CLI source is copied. Concrete factory tests bind regular files without executing them. These tests provide no real engine, provider, turn, recovery or Apply evidence.

## Pinned protocol codec

`NewCodec` supplies pure in-memory encoding and NDJSON framing for a selected stable Codex App Server 0.162.0 subset. It encodes `initialize`, `initialized`, `thread/start`, a single text input in `turn/start` with optional effort and object output schema, and `turn/interrupt`. It does not start processes, read credentials, redact secrets, execute tools or correlate outstanding requests. Encoded instructions are literal caller inputs, including explicit empty strings; the codec selects no runtime policy or instruction sources.

`Feed` accepts fragmented or multiple lines. Each line, including LF, is bounded by the caller's limit, at most 1 MiB; JSON nesting is bounded to 64 compound levels. Invalid UTF-8, unpaired Unicode surrogate escapes, duplicate keys at any depth, trailing JSON values and malformed envelopes are refused. This prevents Go's JSON replacement of malformed surrogate escapes from collapsing distinct wire IDs. A malformed or oversized line latches the decoder's failure and returns earlier complete observations plus a rejected observation. The rejected oversized observation is only a bounded prefix and has `Truncated` set. Later bytes in that fragment are not interpreted; the caller must stop consuming the affected connection. `End` reports incomplete final bytes without supplying an invented LF; `Pending` retains a copied unfinished prefix.

`Message.Raw` returns a copy of the original provider NDJSON bytes, including LF, leading/trailing whitespace and CR when present. `Digest` hashes exactly those bytes, or the explicitly truncated prefix. This boundary is not the engine command, agentd frame headers, reconstructed tool envelopes or a redacted journal representation. The codec provides no replay or dispatch authority. A future Orchestrator consumer must redact known secrets before every journal append while retaining the separate true wire identity.

Request IDs retain separate signed-integer and string forms. Thread and turn IDs have distinct types and come only from the corresponding provider fields. `ReadResponse` requires the caller's expected request ID and method. It validates the selected identity/status projection and required field presence, not the full upstream schemas: unconsumed configuration, thread fields and turn items remain opaque in `Raw`. No result acceptance follows from a provider's `completed` status. `failed` and `interrupted` remain distinct. Unknown notifications and unsupported server requests have explicit kinds and keep their raw bytes; a consumer must resolve their significance rather than discard them as harmless. Unknown turn statuses and inconsistent selected event shapes are refused with retained raw observations.

The wire shapes were checked against the official [`rust-v0.162.0` commit c1382380de69521303b416720a52f42d51af6248](https://github.com/openai/codex/tree/c1382380de69521303b416720a52f42d51af6248/codex-rs/app-server-protocol/schema/json), not current unversioned examples. Paths in the table are relative to that commit's `codex-rs/app-server-protocol/schema/json/`; hashes identify the complete upstream bytes consulted. These are documentation references, not vendored schema files. The tests contain independently authored synthetic examples, not captured provider traffic or upstream schema copies.

| Upstream file | SHA-256 |
| --- | --- |
| `JSONRPCMessage.json` | `6da4cc373a7ccd01f5608ad4496eeb4d44b6a9b0d3158e85ed66b59f3c0b5114` |
| `RequestId.json` | `9c72d59a306d827113020c6d31a14c326d55210d9bd6954f825196a52009f1c7` |
| `ClientNotification.json` | `706cf248d75027c84a3c63348d0ed507182e8eba40069dd17541793de029145a` |
| `v1/InitializeParams.json` | `b55124e7c0f36a1789396daa5578217a92be67f053903696236686e59f6f212e` |
| `v1/InitializeResponse.json` | `62ad689c2cb6379913c1d72749cfd8de5089d35760214123518eb92eef11acc9` |
| `v2/ThreadStartParams.json` | `e9c6d3cc18d049bfbc0249add3808fb8e5a27e1a0b5a423767aafbc3859a9428` |
| `v2/ThreadStartResponse.json` | `7247cd531d3f78174444f8262fd3090d5acf8ab74084547b4e0a831d74e77c9a` |
| `v2/ThreadStartedNotification.json` | `9d4763337276cf6ff56b6ee611edfaa652801e01737bd5fc461353e12aa86505` |
| `v2/TurnStartParams.json` | `583aff13e7a2da43fbb0150e0edc3c73171f374f0aa745dddba256166988a005` |
| `v2/TurnStartResponse.json` | `20e837d07826b9c01578ff40b962ab08f58a355a9984e96226733498514e9bad` |
| `v2/TurnStartedNotification.json` | `74050381ae8a90db4893fc15fc26ef14560762c3d320c3fa8d9df9b1aba1273b` |
| `v2/TurnCompletedNotification.json` | `69413578ec3ecd02691424bdeb9d640c7244b4ca4c5c3ef26f69a2efe7c37db8` |
| `v2/TurnInterruptParams.json` | `6dff382dae73d1dbc58406ed045605f647e7a49660e2540fbd2c6c24d60c5f2b` |
| `v2/TurnInterruptResponse.json` | `531de6be06fe979b5963f249bab82498a175e614bf65ac12fb2e849dfe60bcf1` |

The upstream root `LICENSE` at the same commit is Apache-2.0, SHA-256 `d17f227e4df5da1600391338865ce0f3055211760a36688f816941d58232d8dc`. No schema generator, CLI installation, schema export, image build or provider invocation is part of this codec. These source tests establish neither a runtime handshake nor recovery, turn readiness or Apply support.

## One turn: `run.Agents`

`NewAgents(engine, profiles, resolver)` returns the `run.Agents` adapter over the container engine; `NewAgentsWithRuntime(start, profiles, resolver)` exposes the runtime session operations (start a container, private initialization and cleanup) for Orchestrator injection and fake-runtime tests. `profiles` maps the Runtime profile names a `run.Turn` names to a `Profile{ImageID, CredentialRef, MaxWireBytes, Session}`; `ProfileFromPrepared` builds one from complete source-bound preparation, and both constructors validate every profile. The adapter creates no workspace or image and resolves no ambient environment, executable or credential. It keeps no state between turns.

`Run(ctx, turn, handler)` admits the turn before anything starts: a known profile, plain volume names for workspace and home, a read-only workspace and no writable mount for a Verify turn (which may resume only its own reviewer thread of the same review Pass), distinct clean relative `Turn.Mounts` paths, and offered tools with unique names, descriptions and object schemas. A refused turn starts no container and reports `Stopped`. Otherwise one turn is one container:

- The workspace volume is mounted at `/work` with `NoCopy`, read-only when `turn.ReadOnly`, followed by one `NoCopy` overlay per `turn.Mounts` entry (the same volume's subpath `Path` at `/work/Path`, read-only unless `Writable`); the home volume at `/home/agent`; user `1000:1000`; resource limits and network from the profile. Codex runs `/usr/local/bin/codex app-server` with unrelated UI features disabled and the offered tools served through `/usr/local/bin/markitect-agentd mcp`. Approval policy is `never` and Codex's own sandbox is `danger-full-access` inside the hardened container, so a read-only turn is read-only because its workspace mount is.
- The handshake is `initialize`, `initialized` and either persistent `thread/start` or `thread/resume` naming only `turn.Resume`, whose returned thread must match exactly. One `turn/start` carries `turn.Prompt` and the profile's effort.
- Every Orchestrator tool call is answered through the handler, one at a time, with known secrets removed from its arguments. The call must name an offered tool and carry the turn's own thread ID. The answer text becomes the MCP text; a refused answer is reported to the provider as `isError:true` and does not fail the turn. The turn ends at Codex's terminal status: `completed` is success, `failed` and `interrupted` return an error with the conversation kept. A non-retrying provider `error` notification, or a terminal event with an unanswered tool call, is a protocol error.
- `Conversation` is the thread ID. `Final` contains only an `agentMessage` whose native phase is `final_answer`; commentary and a missing phase supply no final assessment. `Completed` is set only by a successful native Turn completion, separately from container `Stopped` and returned errors. Completed command items retain their native identity, invocation, status, exit and output in `Commands`, with known secrets redacted; a recovered shell error is not itself a failed native Turn. Command observations are not candidate-bound conformance receipts. `Usage` sums the `thread/tokenUsage/updated` reports of the turn (`last`, one per model request): cached input is reported as `CacheReadTokens` and removed from `InputTokens`, `cacheWriteInputTokens` is `CacheCreationTokens`, each report counts one `ProviderTurns`, the model comes from the thread response, and there is no cost. A turn that reports nothing has zero usage.
- When the provider turn ends, the timeout passes or the context ends, the adapter asks agentd to stop the provider's process group and closes the connection, which ends the container. `Stopped` is true only when agentd's exit was observed and `Close` ended the container without having to remove it by force, or when no container was ever started; a failed container creation is uncertain and not stopped. Cancellation and connection close alone do not prove termination. A failed write may have been partial and is never repeated.

`turn.Transcript`, when set, is an absolute path opened in append mode. It receives one JSON line per exchange (`at`, `direction`, `data`): Orchestrator writes, provider output lines, tool calls and replies and the observed exit, each with known secrets redacted. A transcript that could not be written fails the turn.

## Persistent assignments: `run.AssignmentAgents`

`OpenAssignment` starts one container and App Server process and completes only
`initialize` / `initialized`. It sends no `thread/start`, `thread/resume` or
`turn/start` until the first `Message`. This allows the Host to prepare dormant
Manager containers concurrently before giving them assignments. The first
message starts the thread; later messages, including corrections after Verify,
reuse that thread, process, home and mounted workspace until `Close`.

Closing a dormant assignment stops its process and container without reporting a
conversation, completed response or usage. A pre-cancelled first message starts
no thread. Failed thread creation closes the assignment and retains its partial
transcript and cleanup observation; it never starts a turn. These are fake-runtime
source checks, not evidence of provider or Apply execution. The single-turn
`Run` entrypoint still initializes, starts/resumes its thread and executes its
prompt immediately, then closes its container.

### Native delegation is disabled

Managers delegate required responsibilities through Markitect's `delegate_child`
tool, with their own Manager assignments and working copies. In the retained
22 October Library Run04, a native `/root/reminders` child performed work while
the required Markitect `lending/reminders` Manager received no message. Native
Codex delegation must therefore be unavailable in the Manager's tool surface.

Both `Run` and `OpenAssignment`, in either credential mode, launch App Server
with `--disable multi_agent --disable multi_agent_v2 -c agents.enabled=false`.
The [official agent guide](https://learn.chatgpt.com/docs/agent-configuration/subagents)
and [configuration reference](https://learn.chatgpt.com/docs/config-file/config-reference)
document the agent tool switch. The pinned CLI 0.162.0 source at
[`c1382380`, `Config::multi_agent_version_override`](https://github.com/openai/codex/blob/c1382380de69521303b416720a52f42d51af6248/codex-rs/core/src/config/mod.rs)
explains the two additional boundaries: Luna's model metadata selects V2 even
when `features.multi_agent` is false, and an explicitly enabled
`features.multi_agent_v2` takes precedence over `agents.enabled=false`. The launch
overrides all three values, including inherited home configuration.

`TestRuntimeNativeAgentTools` checks the installed CLI and effective App Server
`config/read` values in a fresh, credential-free, `network=none` container. A
local HTTP recorder captures the first model request and returns a synthetic
400; each case has one local `turn/start` and no external provider or inference
call. On image
`sha256:29f746d51945608382f3dc74725800fe27dd42375d41f146939e22a78387d92f`
with CLI 0.162.0, the feature-only and explicit-V2 counterexamples expose
`collaboration.spawn_agent`; the actual launch exposes no native collaboration
tools. Ordinary shell and file tools remain in `functions.exec`'s declaration.
The same session's `mcpServerStatus/list` retains Markitect's `delegate_child`
with its schema. MCP tools are deferred: `functions.exec` documents `ALL_TOOLS`
discovery, while their individual signatures are absent from that first model
declaration. Existing handler regressions separately cover Orchestrator calls
and replies. This is local tool/config evidence, not a real provider or Apply run.

Repeat that bounded offline check for a changed CLI/image:

```powershell
$env:MARKITECT_TEST_ENGINE = 'C:\Program Files\Docker\Docker\resources\bin\docker.exe'
$env:MARKITECT_TEST_RUNTIME_IMAGE = 'sha256:29f746d51945608382f3dc74725800fe27dd42375d41f146939e22a78387d92f'
go test ./internal/adapters/codexappserver -run '^TestRuntimeNativeAgentTools$' -count=1 -timeout=3m -v
```

### Retained native subagent notifications

Correlation of native child notifications remains defensive protocol handling
for older evidence or unexpected provider output; it grants no native tools.

The adapter recognizes native child IDs only from a `subAgentActivity` with
`kind: started` in its bound Manager turn or an already recognized native child.
The retained identity bytes are bounded by the protocol byte limit. Recognized
child thread/turn lifecycle, item and child-local error notifications remain in
the transcript and do not change the Manager's turn ID, Final, commands or usage.
Only the bound Manager's own terminal event can complete its response. Unrelated
thread-start, turn lifecycle and item notifications, mismatched Manager turns
and foreign/unoffered Orchestrator tool calls remain protocol errors. Foreign
usage reports and other unconsumed notifications remain best-effort observations
that are ignored. This is provider event correlation, not semantic
acceptance of the resulting work. Regression tests establish adapter behavior;
a new coordinated provider attempt supplies separate runtime evidence.

`TurnResult.Usage` includes only the bound Manager thread's usage reports.
`ProviderTurns` counts those reports, not all provider requests or native child
work. These fields therefore do not measure total provider consumption when
native subagents run. No child usage is silently added to the Manager; without
billing evidence total cost remains unknown, including when `CostUSD` is zero.

### Runtime credentials

The adapter resolves only the profile's `CredentialRef` through the caller's `CredentialResolver`. `RuntimeCredential.Mode` must explicitly be one of:

- `api-key`: only `APIKey` is populated. It is supplied solely through provider `Start.Env` as `OPENAI_API_KEY`; an explicit Responses API provider configuration uses that environment key. No login-file fallback is selected.
- `dedicated-runtime-login`: only `RuntimeLogin` is populated, from a dedicated runtime login maintained by the Orchestrator. The accepted closed `auth.json` shape has `auth_mode:"chatgpt"`, null `OPENAI_API_KEY`, tokens `id_token`, `access_token`, `refresh_token`, `account_id`, and optional `last_refresh`. The adapter strips the refresh token, sends a bounded copy through the existing agentd connection after the Manager container starts, before its provider starts and uses Codex's file credential store. No credential is read from the desktop or environment. Credential provenance is the resolver's Orchestrator obligation; a byte payload cannot authenticate its origin.

Frame protocol 3 admits only the fixed `/home/agent/.codex/auth.json` path, with no supplied pathname, shell, credential argv or credential environment. The private `.codex` directory is 0700 on Linux, the exclusively created file is 0600, and directory identity checks and rooted handles refuse redirected paths. Existing auth files are never overwritten. Initialization is one-shot; failed initialization prevents provider startup. The private setup frame is excluded from transcripts.

After observed provider exit, the adapter requests private cleanup on that same connection before closing the container. An unacknowledged setup is remembered for cleanup even when no provider started. agentd also retries cleanup after disconnect, stops provider processes first, and returns a technical error if cleanup fails. Only the installed auth file is removed; the Run owner must still remove the private home volume, including other files the provider may have created. No separate auth issue/remove container is started. The adapter continues to strip refresh tokens; safe normal-login expiry, refresh and credential fan-out are unconfirmed. This transport slice is not completed V1 authentication. The existing API-key branch remains until its bootstrap/config removal is coordinated. Known secrets and their JSON-escaped forms remain redacted from errors, `Final` and transcripts.
Synthetic tests cover complete fake-runtime turns, fresh and resumed threads, read-only mounts, Orchestrator answers and refusals, known secrets removed from tool-call arguments, foreign and unoffered tool calls, provider failure, timeout, cancellation, unobserved stop, start failures, admission refusals, transcript redaction, usage, and dedicated-login copy and cleanup. They establish source behavior only, not real provider execution, runtime readiness, guarded Apply or owner acceptance. The shape of `thread/tokenUsage/updated` was read from the pinned upstream schema; it and the `item/completed` agent message are not part of the codec table above. Existing [runtime probes](../../../docs/validation/runtime-capabilities-20261017.md) keep their own evidence scope.

## Opt-in actual turn smoke

Status: **UNEXECUTED**. `agents_smoke_test.go` prepares a bounded real adapter smoke. Compilation and skipped tests establish no runtime or provider result.

`TestAgentsSmoke` calls the real `NewAgents` with a one-tool `report` catalogue and a handler that records the report and answers with a random token. It runs three read-only turns under one five-minute bound: an Execute turn, an Execute resume in a new container with the same home, and a fresh Verify turn with another home. Each turn must report the observed file content through the Orchestrator tool; the resume must repeat the token and keep the conversation ID; the fresh turn must have another one; every turn must observe its container's exit and leave the one-file workspace unchanged.

Ordinary CI leaves the boolean command-line activation flag false. Runtime environment variables cannot enable it. The activation check precedes profile, engine or credential access. A future operator must supply all of these explicit flags after a finite runtime grant; no command below is an execution record:

| Flag | Explicit input |
| --- | --- |
| `-markitect-codex-session-smoke` | Boolean opt-in, default false. |
| `-markitect-codex-smoke-engine` | Absolute engine executable path. |
| `-markitect-codex-smoke-image` | Immutable `sha256:` image ID, checked against the engine observation. |
| `-markitect-codex-smoke-profile` | Absolute closed SessionProfile JSON source path. |
| `-markitect-codex-smoke-profile-sha256` | Expected whole source bytes SHA-256, lowercase hex without prefix. |
| `-markitect-codex-smoke-metadata-sha256` | Expected whole image `/opt/markitect/runtime.env` SHA-256, same format; the pinned provider version must also match. |
| `-markitect-codex-smoke-login-volume` | Name of a preexisting dedicated runtime login volume containing auth.json; never desktop login. |

The profile selects Codex Luna (`gpt-6-luna`) with high effort. Run `TestAgentsSmoke` alone with `-count=1`, an external test timeout greater than the five-minute bound plus cleanup, and the flags through `go test ... -args`. Never place credential values in flags, output or committed files. Created smoke volumes are removed with bounded cleanup.

## Inbox transport technical probe (unexecuted)

`TestInboxNativeProbe` reuses the private-init probe's explicit engine/image/profile/metadata flags, private stdin, source-bound agentd handshake and resource cleanup. Select only this test and enable `-markitect-private-inbox`; do not also enable `-markitect-private-native`. Default source tests never execute either native scenario. A concrete finite coordinator dispatch is required before execution.

This scenario uses one real Codex session and an explicitly synthetic counterpart. It offers only `inbox_ask`, `inbox_await` and `inbox_reply`: Ask returns Q1's correlation ID immediately; the first argument-free Await returns independent counter-question Q2; the native Manager must Reply to Q2 using the original assignment token; only then does the next Await return Answer(Q1). Question IDs and tool descriptions do not reveal the task or answer token. A second message recalls both values without further tool calls. This is a fixed test fixture for the S2 owner's Ask/Reply/Await semantics, not another inbox implementation or a Run port.

Passing requires four ordered MCP calls, the native-authored correct Reply(Q2), completed messages on the same nonempty thread, retained task/answer tokens, no observed command execution, and the existing private-home/container/volume closure checks. After closure, the actual redacted transcript must contain four unique tool calls and matching replies, with every call's native thread metadata equal to both message results. `probe.json` adds an `Inbox` observation identifying the synthetic counterpart, correlations, message/tool thread IDs and whole-transcript SHA256. Partial attempts remain incomplete.

Freeze the actual host source, runner and nonsecret profile before dispatch. An existing immutable runtime image may be reused only when its agentd/frame/provider-launch inputs are unchanged and its exact recorded metadata and protocol are checked; record its original image source separately from the host probe source. No image rebuild or provider execution is part of preparing this test. One real Manager with a synthetic counterpart cannot establish two real crossed Managers, credential fan-out, full hierarchy, S2 Run/Apply, steering, Claude support or owner acceptance.

## Private initialization technical probe (unexecuted)

`TestPrivateHomeNativeProbe` in `private_home_native_test.go` is disabled by default. It is a technical adapter probe using the existing concrete consumer, not a product Run, Root-only implementation or new session contract. Two messages in one native conversation test the private login transport, a developer-instruction canary absent from user prompts, one MCP tool and previous-response recall. A passing record covers only these observations. It does not prove contradictory repository-instruction suppression/precedence, refresh/expiry, safe credential fan-out, full canonical harness/context, a second fresh session, Run/Git/handoff or any complete modeled hierarchy.

Source preparation may compile this runner without enabling it. Freeze the independently reviewed source commit and whole runner executable SHA256. The runtime must be built from that same source with `MARKITECT_VERSION` equal to the full source SHA and frame protocol **3**. Existing images are not assumed compatible. The recipe pins Codex **0.162.0** and Claude **2.1.301**; this probe executes only Codex. A separate Claude normal-login/session runner is still absent, so Claude remains blocked/unqualified rather than certified by Codex.

Proposed future authority is one finite package with two sequential stages; neither has been executed. No second confirmation is required merely because the immutable image ID is produced by its build step. Total wall time is at most fifteen minutes; binding failure aborts without a repair/retry cycle:

1. One runtime image build from the frozen source, with its pinned base-image/package downloads if unavailable, at most nine minutes and no automatic retry. Record the resulting immutable image ID and whole `/opt/markitect/runtime.env` SHA256 using a never-started metadata container; remove that exact metadata container. Record the resulting image/metadata bindings and compare the predeclared source, agentd and CLI versions before enabling inference. Metadata acquisition has a thirty-second limit. The image can be retained for this bounded probe. No credential enters the build context. Failure or missing version/protocol binding ends preparation.
2. One technical Codex Manager container and two exclusive named volumes, two messages (45 seconds each), at most two accepted MCP calls, three minutes total execution and ninety seconds cleanup, with an outer five-minute runner timeout. Profile ceilings are one CPU, 2 GiB RAM, 192 PIDs and 256 MiB temporary storage. This proposal selects Codex `gpt-6-sol`, effort `high`, and the engine default network (explicit empty string). The runner replaces only its store and instruction fields with this probe's bounded fixture and records the effective profile digest. There is no automatic retry or reset. Message/time limits do not establish a monetary hard cap.

The normal-login source must be recorded by mode and exact private path in the separately granted invocation record, outside repository documentation. Feed that file privately to runner stdin; do not put its contents into argv, environment, logs, Notes or bundles. This slice still installs an access-only copy, so a successful account call proves that copy worked at that time, not refresh support.

The native CLI authenticates as the private home's user and retains its normal native tools. Owner-only file permissions isolate it from other users; they do not isolate its auth file from that CLI's own same-UID tools. Probe instructions do not establish such isolation, and this probe makes no credential/tool-isolation claim. The existing token redaction remains active at the adapter's transcript, error and host-tool boundaries.

Compile-only preparation, with an external output path:

```powershell
go test -c ./internal/adapters/codexappserver -o C:\private-probe\codex-private-native.test.exe
Get-FileHash -Algorithm SHA256 -LiteralPath C:\private-probe\codex-private-native.test.exe
```

Within that single granted package, after successful build/metadata preflight and recording the exact bindings, run the frozen executable with this argument template. Source, model, profile, limits and invocation are reviewed before the grant. Resolve the image/metadata placeholders only from its one successful source-bound build, without another approval round:

```text
codex-private-native.test.exe
  -test.run=^TestPrivateHomeNativeProbe$ -test.v -test.timeout=5m
  -markitect-private-native
  -markitect-private-engine="C:\Program Files\Docker\Docker\resources\bin\docker.exe"
  -markitect-private-image=IMMUTABLE_SHA256_IMAGE_ID
  -markitect-private-profile=ABSOLUTE_NONSECRET_SESSION_JSON
  -markitect-private-profile-sha256=WHOLE_PROFILE_SHA256
  -markitect-private-metadata-sha256=WHOLE_RUNTIME_ENV_SHA256
  -markitect-private-agentd=FULL_REVIEWED_SOURCE_SHA
  -markitect-private-evidence=NEW_ABSOLUTE_PRIVATE_EVIDENCE_DIRECTORY
```

On Windows use `Start-Process -WindowStyle Hidden` with `-RedirectStandardInput` pointing to the granted login file, separate stdout/stderr capture paths and the argument list above. Capture the exit code; a nonzero exit overrides a semantic-looking record. Do not invoke the test through an unbounded shell pipeline. The evidence directory must be new. The runner also rechecks metadata using one additional never-started container; at most two never-started metadata containers are created across the package. The runner records nonsecret bindings, protocol, initialization/start/clear/close acknowledgements, attempted volume names/ownership and a redacted transcript. Preserve all attempts, including failed preflight logs.

Stop on input/build binding mismatch, login rejection, lost transport, unexpected/missing tool calls, canary/recall mismatch, timeout or uncertain cleanup. On closure, confirm provider stop and private cleanup acknowledgement, absence of the uniquely labelled container, unchanged ownership of each created volume before removal, no remaining mounts and observed volume absence. The generated auth file is removed by agentd; deleting the private home volume also removes any additional provider-created files. A failed exclusive volume creation may have uncertain effects: retain its attempted name, do not adopt/delete a preexisting volume, report the failure. Never delete resources outside the recorded private ownership. A passing small technical probe does not release all provider-dependent signatures; the remaining instruction/auth and complete-hierarchy gates retain their scope.
