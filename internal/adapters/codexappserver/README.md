# Codex adapter

This package implements Apply's static `AgentProfilePreparer` and a separate admitted single-step `run.Sessions` adapter. Bootstrap currently injects `NewEngineImageReader` into `NewMetadataPreparer`; session construction is an explicit Host operation. The metadata factory receives its executable only from the selected, source-bound Host section. It has no PATH, environment, desktop binary or historical offline fallback.

## Static image metadata

The existing Host envelope remains `markitect.host/v1` with a `codexAppServer` section. The closed section has these required fields:

```yaml
profileVersion: 2
engineExecutable: /usr/bin/docker
imageID: sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
agentdVersion: development
recordedCodexSchemaDigest: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
instructions: []
```

The two digests above are illustrative, not a usable image/profile. Use the actual immutable engine image ID and the expected build-recorded schema digest. `engineExecutable` must be an exact native absolute path on the Host; it names the engine, never a provider executable. `instructions` is an ordered list of `{boundary, path}` coordinates for already captured source files. Missing instructions request a new complete acquisition and remain incomplete; this adapter never reads instruction paths.

### Source-bound execution settings

Version 2 remains metadata-only. Version 3 uses the same `codexAppServer` section in `markitect.host/v1`, retains every version 2 field above and requires these explicit additions. This illustrative section shows the complete shape; its image, schema and model literals establish no runtime availability:

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

All fields are required, including explicit empty instruction strings and network. Numeric fields require YAML numbers, with integer types for byte counts, version and process count. The adapter validates `session` through `NewSessionProfile`; model, credential reference and store are bounded nonempty literals. `maxWireBytes` is a positive integer at most 1 MiB, matching the current Codex transport limit. There are no environment lookups, interpolation, defaults or secret reads. Credential references remain opaque identifiers for the later Host resolver. Base and developer instruction strings are passed literally, including newlines and empty strings. Captured `instructions` remain separate ordered inputs to the existing context renderer; this projection never appends or duplicates them in either string.

The v3 static observation uses `markitect.codex-app-server.image-metadata` contract version 3. Apply's identity binds the whole captured Host document and selected field, captured instruction view, observation contract and actual image metadata. A change to execution settings therefore changes the static identity even when image metadata stays identical. Version 2 retains contract version 1.

`SessionProfileFromPrepared(prepared apply.PreparedProfile) (profile run.Payload, credentialRef string, maxWireBytes int64, err error)` requires complete passed preparation with an identity, exact input-binding echo, v3 contract, Codex provider, selected immutable image and matching recorded versions/schema. It uses the same private configuration decoder as metadata preparation and the existing `NewSessionProfile` factory. It returns only a fresh immutable payload and the captured credential reference and wire limit. The already observed image ID remains at `prepared.Observation().AgentImage.ImageID`. Missing facts, old v2 metadata, foreign contracts and mismatches refuse without partial output. Projection acquires nothing and resolves no credentials.

The Host still must bind those returned facts and the observed image into its admitted SessionPlan and supply the existing runtime credential resolver. This source producer does not wire Core profile selection, establish an execution grant or prove provider/model availability. Fake-reader tests use the actual Apply consumer; they run no Docker, provider, credentials, Apply or smoke.

Preparation resolves the selected image ID, copies only `/opt/markitect/runtime.env` through `Engine.ReadImageFile`, and checks the image identity again. The engine creates a container for copying and removes it without starting it. It does not pull/build an image, start a Session, invoke either provider, probe a CLI version or export a schema.

The supported metadata format is exactly four unique LF-separated records emitted by [the runtime Dockerfile](../../../packaging/runtime/Dockerfile): `codex`, `claude`, `codexSchema` and `agentd`. Values must be bounded nonempty UTF-8 literals without control characters or surrounding whitespace. The recorded Codex version must equal `codex-cli 0.162.0`; the recorded agentd version and Codex schema digest must match the selected profile. The Claude record is retained through the whole file identity, with no assertion that this package supports Claude sessions. Unknown/duplicate records, old profile versions, tags and passthrough fields fail closed.

Apply binds the actual immutable image ID, canonical metadata path, whole-file SHA-256 and size, literal CLI/agentd versions and build-recorded Codex schema digest. The schema digest is a build assertion, not an observed schema inventory or provider handshake. File acquisition is bounded to 1 MiB. Engine errors and cancellation preserve completed observations as incomplete; known mismatches are failures. Neither outcome grants runtime readiness or execution authority.

Tests use the real Apply profile consumer with source fixtures and fake image-reader peers. Metadata bytes are synthetic examples of the repository's runtime format; no third-party schema or proprietary CLI source is copied. Concrete factory tests bind regular files without executing them. These tests provide no real engine, provider, Session, recovery or Apply evidence.

## Pinned protocol codec

`NewCodec` supplies pure in-memory encoding and NDJSON framing for a selected stable Codex App Server 0.162.0 subset. It encodes `initialize`, `initialized`, `thread/start`, a single text input in `turn/start` with optional effort and object output schema, and `turn/interrupt`. It does not implement `run.Sessions`, start processes, read credentials, redact secrets, execute tools or correlate outstanding requests. Encoded instructions are literal caller inputs, including explicit empty strings; the codec selects no runtime policy or instruction sources.

`Feed` accepts fragmented or multiple lines. Each line, including LF, is bounded by the caller's limit, at most 1 MiB; JSON nesting is bounded to 64 compound levels. Invalid UTF-8, unpaired Unicode surrogate escapes, duplicate keys at any depth, trailing JSON values and malformed envelopes are refused. This prevents Go's JSON replacement of malformed surrogate escapes from collapsing distinct wire IDs. A malformed or oversized line latches the decoder's failure and returns earlier complete observations plus a rejected observation. The rejected oversized observation is only a bounded prefix and has `Truncated` set. Later bytes in that fragment are not interpreted; the caller must stop consuming the affected connection. `End` reports incomplete final bytes without supplying an invented LF; `Pending` retains a copied unfinished prefix.

`Message.Raw` returns a copy of the original provider NDJSON bytes, including LF, leading/trailing whitespace and CR when present. `Digest` hashes exactly those bytes, or the explicitly truncated prefix. This boundary is not the engine command, agentd frame headers, reconstructed tool envelopes or a redacted journal representation. The codec provides no replay or dispatch authority. A future Host consumer must redact known secrets before every journal append while retaining the separate true wire identity.

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

The upstream root `LICENSE` at the same commit is Apache-2.0, SHA-256 `d17f227e4df5da1600391338865ce0f3055211760a36688f816941d58232d8dc`. No schema generator, CLI installation, schema export, image build or provider invocation is part of this codec. These source tests establish neither a runtime handshake nor recovery, Session readiness or Apply support.

## One admitted session and step

`NewSessions(engine, resolver)` uses the existing container engine and agentd Session APIs. `NewSessionsWithRuntime(start, helper, resolver)` exposes only those runtime operations for Host injection and source tests. It creates no workspace or image and resolves no ambient environment, executable or credential. The Host first verifies the pinned `codex-cli 0.162.0` image metadata, materializes the candidate and admits the exact workspace, provider home, image ID, credential reference, profile and tools.

`NewSessionProfile(profile SessionProfile) (run.Payload, error)` validates the explicit settings and serializes the closed JSON version 1 as `application/vnd.markitect.codex-session-profile+json` with `utf-8` encoding. Use the returned payload for `SessionAdmission.Profile` and its `Ref()` for `SessionPlan.Profile`. For example, its JSON bytes can be:

```json
{"version":1,"model":"gpt-6-luna","effort":"high","baseInstructions":"","developerInstructions":"","store":"example-store","cpus":1,"memoryBytes":1073741824,"pids":128,"tmpBytes":67108864,"network":""}
```

The model and effort are explicit admitted values; the adapter selects neither. Effort accepts the pinned protocol's `none`, `minimal`, `low`, `medium`, `high` and `xhigh`. The constructor and admission share intrinsic profile validation; the constructor supplies no defaults. This representation is distinct from static metadata. The v3 source projection above translates explicit captured Host configuration into this payload; the Host owns admission and runtime composition. Profile and tools must be valid payloads matching their plan references, and execution requires nonempty admission receipts. Admission requires the exact profile media type and `application/vnd.markitect.tools+json`, both encoded as `utf-8`. Unknown JSON fields and duplicate keys fail closed. RUN's `ToolCatalog(phase)` supplies the closed version 1 catalogue of names, descriptions and input schemas. The adapter registers exactly the admitted catalogue through agentd; it adds no tools or schema of its own. An explicit empty tools list is supported.

The runtime mounts exactly the admitted workspace at `/work` with `NoCopy`, preserving the verified tree even when it is empty and the provider image contains files at that path. Provider home at `/home/agent` retains ordinary initial image copy-up for ownership. The runtime uses user `1000:1000` and the container adapter's hardening. Read-only roles receive a read-only workspace mount. Reviewer sessions require read-only access and cannot resume. Codex runs `/usr/local/bin/codex app-server` with native delegation and unrelated UI features disabled, MCP through `/usr/local/bin/markitect-agentd mcp`, approval policy `never`, and `danger-full-access` inside the hardened container. Resource limits and network come from the admitted profile.

`Open` sends acknowledged `initialize`, `initialized` and either persistent `thread/start` (`ephemeral:false`) or `thread/resume`. The latter names only `Plan.Resume`, with no history or path override; the actual returned ThreadID must match exactly. Incoming handshake bytes are retained in the binding observation. A failed Open retains uncertainty and its runtime handle once a credential helper or container operation was attempted, including failed container creation/start. Credential resolution or validation failure before any such operation returns an error without uncertainty or a binding callback: no provider identity or wire exists to form a valid binding. `Stop` then records certain termination because no runtime operation was invoked. Start reservation settlement belongs to the Host. Retrying the same Session never creates another container.

`Step` verifies the exact admitted input and uses `run.DecodeStepContract` to check the producer's `application/vnd.markitect.step-contract+json` contract before dispatch. It sends one `turn/start` using the bound thread and admitted effort, retains provider NDJSON observations and binds the actual TurnID. Input and contract bytes occur literally in the compiled text input as instructions, never provider `outputSchema`. Every phase's outcome reaches the Host only through its admitted tools. Final answer text remains raw wire evidence, including malformed or report-shaped JSON; it is never projected into `StepObservation.Proposed`. Provider completion, failure and interruption retain distinct terminal reasons without claiming a Host outcome or task acceptance. The obsolete Explore contract and changed or unsupported StepContracts are refused before dispatch.

MCP calls must carry the pinned provider metadata `threadId` and `callId`. Before reaching `SessionIO.Call`, each call must match an actual active-turn `mcpToolCall` item by ID, server, tool and arguments. A call arriving through agentd before that stdout item waits for correlation. The tool must belong to the admitted catalogue. `Call` executes the Host handler and records its closed outcome; an error or absent receipt prevents every answer write. The exact recorded outcome bytes become MCP text, with refusals reported through `isError:true`. The fixed halted-run refusal is never sent. A tool reply's preparation names both Step and Call. The typed agentd tool-call/reply payload is a separate boundary from provider NDJSON and original MCP envelopes.

Every provider NDJSON or tool-reply payload write requires a fresh `PrepareSend` permit with an acknowledged receipt, dispatch permission and exactly matching true byte identity. An actual runtime write is followed by acknowledged `sent`, `not_sent` or `uncertain` observation before provider replies are processed. `sent` establishes the write to the existing agentd transport, not provider consumption or task success. A write error may follow a partial frame; it is uncertain and is never replayed. Known runtime credential bytes and their JSON-escaped representations are redacted from adapter journal payloads while preserving the separate original identity. Redacted observations cannot reproduce the wire. Native thread, turn and call IDs that contain a known credential are refused before binding or a Host callback; uncertainty retains only an absent or previously safe identity. The adapter never substitutes a redacted fake ID. Provider exit signals and returned error text are redacted before observations leave the adapter.

Calls for one Session are serialized. Cancel and await an in-flight Open or Step before calling Stop. After a failed operation the Host stops the session; no recovery path silently resends a turn or reconstructs a provider conversation. `Inspect` checks the supplied known binding handles, then reports `run.ErrObservationLimit`: this adapter has no supported read-only physical/history query. It makes no runtime call. `Stop` reports agentd's actual provider exit code/signal, or retains uncertainty when an exit cannot be established. Cancellation and connection close alone do not prove termination. Start/Stop are typed runtime controls: the current runtime API hides its control request IDs, so the adapter does not invent control envelope bytes or claim their raw identity.

### Runtime credentials

The Host resolves only the admitted `CredentialRef`. `RuntimeCredential.Mode` must explicitly be one of:

- `api-key`: only `APIKey` is populated. It is supplied solely through provider `Start.Env` as `OPENAI_API_KEY`; an explicit Responses API provider configuration uses that environment key. No login-file fallback is selected.
- `dedicated-runtime-login`: only `RuntimeLogin` is populated, from a dedicated runtime login maintained by the Host. The accepted closed `auth.json` shape has `auth_mode:"chatgpt"`, null `OPENAI_API_KEY`, tokens `id_token`, `access_token`, `refresh_token`, `account_id`, and optional `last_refresh`. The adapter strips the refresh token, writes a session copy through a hardened helper's stdin into only the admitted provider home, and uses Codex's file credential store. No credential is read from the desktop or environment. Credential provenance is the resolver's Host obligation; a byte payload cannot authenticate its origin.

The helper uses the exact admitted image and provider-home volume, with no workspace mount. After an actual observed exit, a second hardened helper removes the session copy. Partial installation is remembered for cleanup even if no provider started. If stop is uncertain, the copy is retained; if cleanup fails, the returned error and stop reason explicitly say cleanup is incomplete. The Host must reconcile that retained credential and its lifetime. The adapter does not refresh or rotate a runtime login. These mechanics implement the existing [runtime credential procedure](../../../packaging/runtime/README.md#credentials); they are not a new provider experiment.

The resume and item/call projections were checked against `v2/ThreadResumeParams.json`, `v2/ThreadResumeResponse.json`, and `codex-rs/core/src/mcp_tool_call.rs` at the same pinned upstream commit as the codec table. Synthetic tests cover a complete fake-runtime turn, Host answers/refusals, exact admission/permit binding, partial writes, callback recording failure, foreign-thread calls, cancellation, resume mismatch, read-only mounts, dedicated-login copy/cleanup and cleanup uncertainty. They establish source behavior only, not real provider execution, runtime readiness, guarded Apply or owner acceptance. Existing [runtime probes](../../../docs/validation/runtime-capabilities-20261017.md) keep their own evidence scope.

## Opt-in actual Session adapter smoke

Status: **UNEXECUTED**. The source in `session_smoke_test.go` prepares a bounded real adapter smoke. Compilation and skipped tests establish no runtime or provider result. Existing container CLI probes bypass this adapter and cannot supply this evidence.

The test calls actual `NewSessions(engine, resolver)` with explicit `run.SessionAdmission` and `run.StepAdmission` values, the current `run.ToolCatalog` and `run.NewStepContract`. Its `controlledSessionIO` is explicitly in-memory smoke scaffolding: its synthetic receipts acknowledge that observer only. It establishes no durable journal, Coordinator admission, Core integration, task acceptance, recovery or Apply authority.

A separately authorized execution admits at most three read-only steps under one five-minute context: manager work, manager resume in a new container using the same provider home, then a fresh reviewer Candidate step using a different home. Every step must make the expected Host outcome call with the observed artifact content; final text cannot satisfy it. Resume must retain the actual provider session ID and recall a random token supplied only in the first Host answer. A fresh reviewer must have a different observed provider ID. Reviewer resume remains forbidden by the current contract. Each step requires exact acknowledged wire permits, observed sends, terminal observation, actual Stop and readback of the unchanged one-file candidate. Initial materialization refuses image-populated workspace entries; all subsequent workspace mounts use `NoCopy` and are read-only.

Ordinary CI leaves the boolean command-line activation flag false. Runtime environment variables cannot enable it. The activation check precedes profile, engine or credential access. A future operator must supply all of these explicit flags after a finite runtime grant; no command below is an execution record:

| Flag | Explicit input |
| --- | --- |
| `-markitect-codex-session-smoke` | Boolean opt-in, default false. |
| `-markitect-codex-smoke-engine` | Absolute engine executable path. |
| `-markitect-codex-smoke-image` | Immutable `sha256:` image ID, checked against the engine observation. |
| `-markitect-codex-smoke-profile` | Absolute closed SessionProfile JSON source path. |
| `-markitect-codex-smoke-profile-sha256` | Expected whole source bytes SHA-256, lowercase hex without prefix. |
| `-markitect-codex-smoke-metadata-sha256` | Expected whole image `/opt/markitect/runtime.env` SHA-256, same format; the pinned provider version must also match. |
| `-markitect-codex-smoke-login-volume` | Name of a preexisting dedicated runtime login volume containing auth.json; never desktop login. The helper reads only that volume and the adapter strips refresh authority from its session copy. |

The profile selects Codex Luna (`gpt-6-luna`) with high effort. These inputs bind this smoke only; profile provenance and the finite runtime grant remain Host obligations. Run this package's `TestAdmittedSessionAdapterSmoke` alone with `-count=1`, an external test timeout greater than the five-minute runtime bound plus cleanup, and the explicit flags through `go test ... -args`. Never place credential values in flags, output or committed files. Diagnostic output reports stage failures and source digests without logging credential values or raw provider transcripts. Created smoke volumes are removed with bounded cleanup; incomplete Stop or cleanup fails the test and retains uncertainty.
