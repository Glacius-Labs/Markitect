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

`NewAgents(engine, profiles, resolver)` returns the `run.Agents` adapter over the container engine; `NewAgentsWithRuntime(start, helper, profiles, resolver)` exposes only the runtime operations (start a container, run a helper container) for Orchestrator injection and fake-runtime tests. `profiles` maps the Runtime profile names a `run.Turn` names to a `Profile{ImageID, CredentialRef, MaxWireBytes, Session}`; `ProfileFromPrepared` builds one from complete source-bound preparation, and both constructors validate every profile. The adapter creates no workspace or image and resolves no ambient environment, executable or credential. It keeps no state between turns.

`Run(ctx, turn, handler)` admits the turn before anything starts: a known profile, plain volume names for workspace and home, a read-only workspace and no writable mount for a Verify turn (which may resume only its own reviewer thread of the same review Pass), distinct clean relative `Turn.Mounts` paths, and offered tools with unique names, descriptions and object schemas. A refused turn starts no container and reports `Stopped`. Otherwise one turn is one container:

- The workspace volume is mounted at `/work` with `NoCopy`, read-only when `turn.ReadOnly`, followed by one `NoCopy` overlay per `turn.Mounts` entry (the same volume's subpath `Path` at `/work/Path`, read-only unless `Writable`); the home volume at `/home/agent`; user `1000:1000`; resource limits and network from the profile. Codex runs `/usr/local/bin/codex app-server` with native delegation and unrelated UI features disabled and the offered tools served through `/usr/local/bin/markitect-agentd mcp`. Approval policy is `never` and Codex's own sandbox is `danger-full-access` inside the hardened container, so a read-only turn is read-only because its workspace mount is.
- The handshake is `initialize`, `initialized` and either persistent `thread/start` or `thread/resume` naming only `turn.Resume`, whose returned thread must match exactly. One `turn/start` carries `turn.Prompt` and the profile's effort.
- Every Orchestrator tool call is answered through the handler, one at a time, with known secrets removed from its arguments. The call must name an offered tool and carry the turn's own thread ID. The answer text becomes the MCP text; a refused answer is reported to the provider as `isError:true` and does not fail the turn. The turn ends at Codex's terminal status: `completed` is success, `failed` and `interrupted` return an error with the conversation kept. A non-retrying provider `error` notification, or a terminal event with an unanswered tool call, is a protocol error.
- `Conversation` is the thread ID. `Final` contains only an `agentMessage` whose native phase is `final_answer`; commentary and a missing phase supply no final assessment. `Completed` is set only by a successful native Turn completion, separately from container `Stopped` and returned errors. Completed command items retain their native identity, invocation, status, exit and output in `Commands`, with known secrets redacted; a recovered shell error is not itself a failed native Turn. Command observations are not candidate-bound conformance receipts. `Usage` sums the `thread/tokenUsage/updated` reports of the turn (`last`, one per model request): cached input is reported as `CacheReadTokens` and removed from `InputTokens`, `cacheWriteInputTokens` is `CacheCreationTokens`, each report counts one `ProviderTurns`, the model comes from the thread response, and there is no cost. A turn that reports nothing has zero usage.
- When the provider turn ends, the timeout passes or the context ends, the adapter asks agentd to stop the provider's process group and closes the connection, which ends the container. `Stopped` is true only when agentd's exit was observed and `Close` ended the container without having to remove it by force, or when no container was ever started; a failed container creation is uncertain and not stopped. Cancellation and connection close alone do not prove termination. A failed write may have been partial and is never repeated.

`turn.Transcript`, when set, is an absolute path opened in append mode. It receives one JSON line per exchange (`at`, `direction`, `data`): Orchestrator writes, provider output lines, tool calls and replies and the observed exit, each with known secrets redacted. A transcript that could not be written fails the turn.

### Native subagent notifications

A Codex Agent may use native subagents inside its assignment container and
working copy. The owning Manager still reviews and hands off the resulting Git
stand. Native delegation creates no extra Markitect Manager, assignment or outer
message. The launch command no longer force-disables `multi_agent`; no additional
enable flag or configuration format is introduced. The CLI documents `--disable`
as a feature-flag override ([official reference](https://learn.chatgpt.com/docs/developer-commands?surface=cli)).
That flag was present in the retained RunID03 Library attempt even though native
`subAgentActivity` and child turn notifications were observed; its presence did
not establish absence of native delegation.

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
- `dedicated-runtime-login`: only `RuntimeLogin` is populated, from a dedicated runtime login maintained by the Orchestrator. The accepted closed `auth.json` shape has `auth_mode:"chatgpt"`, null `OPENAI_API_KEY`, tokens `id_token`, `access_token`, `refresh_token`, `account_id`, and optional `last_refresh`. The adapter strips the refresh token, writes a copy through a hardened helper's stdin into only the turn's home volume and uses Codex's file credential store. No credential is read from the desktop or environment. Credential provenance is the resolver's Orchestrator obligation; a byte payload cannot authenticate its origin.

The helper uses the exact profile image and the home volume, with no workspace mount. After an observed exit, a second hardened helper removes the copy. A partial installation is remembered for cleanup even if no provider started. If the exit was not observed the copy is retained; if cleanup fails, the returned error and the reason say so. The Orchestrator must reconcile a retained credential. The adapter does not refresh or rotate a login. These mechanics implement the existing [runtime credential procedure](../../../packaging/runtime/README.md#credentials). Known secrets and their JSON-escaped forms are redacted from errors, `Final` and the transcript.

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
