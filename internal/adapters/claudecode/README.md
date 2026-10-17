# Claude Code session adapter and stream codec

This package implements a bounded `run.Sessions` adapter and a pure in-memory codec for selected Claude Code **2.1.301** stream-JSON shapes. The adapter uses the existing container runtime and agentd MCP transport. It has no CLI/bootstrap wiring and establishes no real provider or Apply acceptance.

## Captured Host profile

`NewMetadataPreparer(ImageReaderFactory)` prepares the `claudeCode` section selected by Apply. The closed adapter-local format uses `profileVersion: 1`; that number is independent of the Codex format and the session payload version. All fields below are required. No metadata-only or legacy format supplies execution settings.

```yaml
claudeCode:
  profileVersion: 1
  engineExecutable: /usr/bin/docker
  imageID: sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
  agentdVersion: development
  instructions: []
  credentialRef: dedicated-runtime-token
  maxWireBytes: 1048576
  session:
    version: 1
    model: explicitly-selected-model
    effort: high
    systemPrompt: "Explicit replacement system prompt."
    builtinTools: [Read, Write]
    store: explicit-store
    cpus: 1.5
    memoryBytes: 2147483648
    pids: 192
    tmpBytes: 2097152
    network: ''
```

The example image ID and model are placeholders for explicit owner configuration. The engine executable must be a canonical native absolute path. `NewEngineImageReader` binds that path through the existing container engine. Preparation inspects the immutable image, copies the complete regular `/opt/markitect/runtime.env` file from a never-started container through a reader bounded to 1 MiB, and inspects the image again. It checks the pinned literal `claude=2.1.301 (Claude Code)` and configured `agentd` build in the packaging format's four unique LF records (`codex`, `claude`, `codexSchema`, `agentd`). The Claude observation binds all actual metadata bytes, the exact image and Claude provider; `RecordedCodexSchemaDigest` remains absent. The other provider's records are retained in that byte identity and do not establish Claude capabilities. Preparation starts no image, session or provider and resolves no credential.

`instructions` is an ordered list of captured `{boundary, path}` locations. Missing captures request a new complete acquisition. Their bytes follow the existing organization rendering path; they are never concatenated into `systemPrompt`. Unknown or duplicate fields, wrong types, absent execution fields and invalid existing `SessionProfile` settings fail before the image reader is created. Acquisition failures retain observed partial evidence and remain incomplete.

`SessionProfileFromPrepared(prepared, stepReadOnly)` returns the existing session payload, literal credential reference and wire limit only after complete source-bound preparation. The Host passes the actual `run.ProfileRequest.ReadOnly` and binds the observed image into its admitted plan. The projection rechecks the selected Claude section, exact input binding, contract, provider, image, versions and instruction locations. Writing steps retain exactly the configured builtin list. Read-only steps receive its intersection with `Read`; a configured `Write` alone or an explicit empty list therefore yields an explicit empty list. Projection does not mutate the prepared writing profile. `NewSessionProfile` remains the intrinsic validation authority, and session admission retains its role, read-only, resume and catalogue checks. Model, effort, prompt, tools, credential and resource values come only from these captured fields; static preparation grants no execution authority.

The metadata and projection tests use the actual Apply consumer with synthetic captured model/profile inputs and controlled image readers. They establish source behavior, including source identity and writing/read-only/writing isolation. They provide no real engine, credential, provider, Apply or owner acceptance evidence.

## One admitted step per container

`NewSessions` receives an explicitly configured runtime engine and a Host credential resolver. `NewSessionsWithRuntime` exposes the same narrow runtime surface for controlled fake evidence. A Session creates one container and sends exactly one admitted user message. Failed Open and attempted Step entries remain retained; repeating either cannot recreate a container or replay a user message. Calls for a Session are serialized; cancel an active Open/Step before Stop.

The closed profile media type is `application/vnd.markitect.claude-session-profile+json`, encoding `utf-8`. Its exact fields are `version` (1), `model` (explicit nonblank), `effort` (`high`), `systemPrompt` (explicit nonblank), `builtinTools` (an explicit array containing only distinct `Read` and/or `Write`), `store`, positive `cpus`, `memoryBytes`, `pids`, `tmpBytes`, and `network`. The tools payload is `application/vnd.markitect.tools+json`, with `version:1` and an explicit `tools` array of agentd `ToolSpec` values. Unknown fields, duplicate JSON keys, invalid Unicode, mismatching payload references and invalid plans refuse before container dispatch. A read-only plan cannot enable `Write`; reviewers require a read-only workspace and a fresh provider conversation.

`NewSessionProfile(SessionProfile) (run.Payload, error)` validates those explicit execution settings and serializes the closed v1 profile with that media type and encoding. The factory and admission reader share intrinsic validation. No model, effort, prompt, builtin tools or resource limit is inferred; an explicit empty builtin array is valid, while an absent array is refused. CPU limits must also be finite. Admission separately checks the plan, role, read-only restriction, resume policy and exact Host tool catalogue. The factory can prepare `Read` for a read-only plan or `Write` for a writing plan; it neither selects a profile for a phase nor translates Host metadata into execution settings. A Host must supply those settings and select a compatible profile for the actual plan.

The container mounts exactly the admitted workspace at `/work` with copy-up disabled, so provider image files cannot populate an empty verified workspace. Persistent provider home at `/home/agent` retains default copy-up for initial ownership. The container uses the recorded image digest and explicit resource limits, and runs as `1000:1000`. The provider executable is `/usr/local/bin/claude`, with no PATH lookup. The stream-JSON, strict MCP, explicit tool allowlist, `dontAsk`, explicit model and high effort invocation follows the [existing runtime probes](../container/README.md). Persistence stays enabled. `--setting-sources ""` excludes user/project/local settings; `--system-prompt` supplies the recorded replacement prompt, using the [official CLI flags](https://code.claude.com/docs/en/cli-reference). Those two additional controls have source and fake invocation evidence here; the previously recorded provider probe did not exercise this adapter invocation.

The Host resolves exactly `CredentialRef` to `RuntimeCredential.Token`, a dedicated long-lived Claude runtime token. The adapter does not read credentials, environment, desktop auth or login files. The token travels only as `CLAUDE_CODE_OAUTH_TOKEN` in `frame.Start.Env`, never container environment, argv, profile or RUN input. Resolver provenance remains a Host obligation. Known token bytes and their JSON escaped spelling are redacted from adapter diagnostics and journal wire payloads; the actual wire hash and byte length remain separate from the permitted redacted bytes. Host tool arguments and answers containing the known token are refused. Typed runtime Start/Stop envelopes belong to the runtime and are not claimed as captured raw provider wire.

Open sends the selected initialization control request only after an exact acknowledged send permit and records actual send observation before reading its response. The matching successful control response establishes `Ready`, with no provider conversation identity. Step validates exact profile/plan/input/contract references and binding, then sends the exact recorded input and output catalogue inside a literal instruction wrapper. The adapter validates the actual RUN `StepContract` with `run.DecodeStepContract` before user dispatch, including Explore, work, integration, reply and both reviewer phases. The exact admitted Host tool catalogue is registered unchanged. Step outcomes reach the Host only through acknowledged Host tool calls. Final text, including a JSON report or a provider `success`, supplies no proposal or accepted outcome; terminal observations leave `Proposed` empty.

Step binds the conversation only from actual `system/init` with the pinned version. A resumed conversation must report exactly the requested session ID. Missing, differing or unacknowledged init observations retain uncertainty. The [RUN guide](../../run/README.md) explicitly withholds proven runtime resume init-ID binding: the runtime continuity probe demonstrated conversation content, not this init-ID check. Matching and mismatching synthetic init events establish source behavior only; a separately authorized bounded real probe remains required before relying on resume capability.

`ProviderStep` holds an explicitly named `claude-single-step-<digest>` **adapter handle**, never a native Claude turn ID. It is established only after the actual init has been durably bound, using that wire observation, the admitted single user frame and the exclusive Session channel. Its distinguishing evidence is the retained actual init, one container, one admitted user write and terminal result on that same physical provider stream. A result UUID, assistant UUID, optional user echo or requested session ID supplies no native turn proof. Host Step IDs and this adapter handle remain distinct. No history, replay or cross-container step equivalence follows from the handle.

Host tool calls arrive only through that container's agentd connection. Their connection-local agentd IDs become explicitly named `claude-channel-call-N` adapter correlation handles; they are not asserted to be native Claude call IDs. The tool must occur in the admitted catalogue, actual session binding and first step observation must be acknowledged, and a Host answer with an acknowledged receipt must exist before a reply is prepared. Host outcome bytes are carried unchanged as MCP text; ordinary refusals set `isError`. A halted-run refusal is never written. Duplicate calls are not resent, and pending calls or unacknowledged writes prevent a terminal claim. No tool callback executes a child Session.

Inspect validates exact recorded session/step handles and receipts and returns `run.ErrObservationLimit`. It has no supported read-only exact-step history query, sends nothing and treats memory as no runtime evidence. Stop reports the runtime's actual provider exit or uncertainty; cancellation, result completion and closing the container alone do not establish exit. The provider home survives the container.

The fake tests exercise lifecycle, exact invocation and input, Host answers/refusals, permits, append failures, no replay, resume/version mismatches, cancellation, stop uncertainty and credential redaction. They run no Docker, provider, credential, Apply or release experiment.

## Selected API and wire shapes

`NewCodec(limit)` bounds each frame, including its actual LF, to between 1 byte and 1 MiB. Use each codec from one consumer. It retains no request map or message history.

`EncodeInitialize(ControlRequestID)` emits a `control_request` with the supplied `request_id` and an `initialize` request subtype, with no optional hooks, agents or SDK capabilities. `EncodeUser(UserInput)` emits one literal text user message with `parent_tool_use_id:null`. UUID and requested session ID are omitted unless the caller supplies them. Explicit `ClientComposed:true` emits the published `client_composed:true`; otherwise that field is omitted. Empty text is a literal caller input, never an automatically inserted bootstrap prompt. String inputs are checked for UTF-8 validity and aggregate input size before JSON encoding; encoded output must also fit the LF-inclusive limit. Each encoding returns fresh bytes.

`Feed` yields these distinct observations:

| Kind | Selected validation and projection |
| --- | --- |
| `ControlResponseMessage` | `response.subtype`, nonempty `response.request_id`, success object payload when present, or error string. Unknown response subtypes remain unsupported. |
| `SystemInitMessage` | `system/init` with nonempty `session_id`, `uuid` and `claude_code_version`. `Init()` returns these observations. |
| `ThinkingTokensMessage` | Closed six-key `system/thinking_tokens` informational envelope with finite numeric estimates and nonempty UUID/session IDs; no outcome or state projection. |
| `UserMessage`, `AssistantMessage` | Message object, matching role and string or array user content; assistant content is an array. Content blocks remain opaque. Assistant UUID/session ID are required; optional user UUID/session ID are checked when supplied. |
| `StreamEventMessage` | Object event, UUID/session ID and parent tool-use ID envelope. Event contents remain opaque. |
| `RateLimitEventMessage` | Closed four-key informational envelope with opaque object `rate_limit_info` and nonempty UUID/session IDs; no outcome or state projection. |
| `ResultMessage` | Known result subtype, boolean `is_error`, session ID and result UUID; success result text or error-subtype errors array. Optional text, errors, structured output and terminal reason are retained by `Result()`. |
| `UnsupportedRequest` | Server control request with nonempty ID and subtype; payload remains raw. No reply or execution API. |
| `UnsupportedMessage` | Other message types and system/result/control-response subtypes, with original bytes visible. |

The user/assistant/stream-event projection checks the required `parent_tool_use_id` as null or a nonempty string. These are selected projections, not full SDK schema validation. Unknown sibling fields, content blocks, tool data, account details, usage and other unconsumed data remain in the owning raw observation. The full JSON, including opaque fields, still passes the framing guards below. Known malformed selected shapes yield `RejectedMessage` and a decoder error, never a successful fallback.

`ReadInitializeResponse(expected)` requires the exact supplied control request ID. A success response must contain `commands`, `agents` and `models` arrays, `output_style` string, `available_output_styles` string array and `account` object. Array contents and account contents remain opaque. The optional `claude_code_version` must be nonempty text. An error response returns `ControlError` and its error text; it is not a successful acknowledgment. A general success acknowledgment without this selected payload fails the initializer projection. Projection errors do not alter the retained message or decoder state.

An initialization reply can precede the first user message. It supplies **no observed provider session ID**, and this codec supplies neither a RUN `Ready` nor a `Bind` receipt. Only the separately decoded `system/init` projection supplies a session-binding observation; the adapter above consumes it. A configured ID is still only `RequestedSessionID` until the adapter confirms it against the actual observation. Version text is an observation, not an enforced runtime pin or handshake proof; preparation and the adapter must enforce the selected image/profile/version. Repeated or differing session observations require adapter reconciliation.

`ControlRequestID`, `RequestedSessionID`, `ObservedSessionID`, `UserMessageUUID`, ordinary `MessageUUID` and `ResultUUID` are separate named types. The codec generates none of them and provides no Host-step or provider-step conversion. Published optional `user_message_uuid` and `user_message_uuids` echoes remain in raw bytes. The published declarations describe batching, multiple consumed prompts and absent echoes; no one-to-one Host-step mapping follows from a result UUID or an optional echo.

`rate_limit_event` is the pinned TypeScript SDK's four-key informational envelope: `type`, object `rate_limit_info`, and nonempty `uuid` and `session_id`. The payload stays opaque. The adapter accepts it only after acknowledged `system/init`, with the same observed session ID; it changes no session/step state and supplies no outcome. Its exact raw bytes remain in the bounded wire observation, including the existing cumulative `MaxWireBytes` bound. A real 2.1.301 subscription-token smoke reported this type and four top-level keys; the detailed envelope comes from the exact published `sdk.d.ts` hash recorded below. The regression uses independently authored synthetic events. The corrected adapter still needs the separately authorized real smoke rerun.

Known result subtypes are `success`, `error_during_execution`, `error_max_turns`, `error_max_budget_usd` and `error_max_structured_output_retries`. `is_error` remains independent: **a `success` subtype can have `is_error:true`**, and does not establish successful work. `terminal_reason` is uninterpreted published status text, including unfamiliar future values; the codec infers no additional outcome from it. A result observation establishes neither task acceptance nor quiescence, process termination, successful cancellation, inspectable exact-step history or recovery. Unknown result subtypes remain unsupported and cannot be read as `ResultObservation`.

`system/thinking_tokens` is approximate thinking progress, not authoritative billed usage. A real Claude Code 2.1.301 smoke reported it after a Read tool result with exactly `type`, `subtype`, `estimated_tokens`, `estimated_tokens_delta`, `session_id` and `uuid` ([failure report](https://github.com/Glacius-Labs/Markitect/pull/388#issuecomment-6241420190)). The pinned `SDKThinkingTokensMessage` declaration in the exact `sdk.d.ts` below declares both estimates as numbers, without an integer or sign constraint. The selected codec accepts finite JSON numbers and only these six observed keys; the SDK's optional `user_message_uuid` extension remains outside this bounded shape. The adapter requires acknowledged init and the same session ID, leaves session/step state and outcomes unchanged, and retains the event in the existing bounded, credential-redacted wire observation. Unknown system subtypes and malformed selected shapes still refuse. Synthetic codec and fake-session regressions cover this correction; successful source checks do not establish a corrected real provider run.

## Framing and immutable evidence

`Feed` accepts fragmented or multiple frames, including fragments within UTF-8 characters. A line includes its LF in the byte limit. CRLF and surrounding JSON whitespace are accepted without normalization. `Message.Raw()` returns a copy of the exact provider NDJSON bytes, including LF and any CR. `Digest()` is SHA-256 of exactly those bytes. Returned result structured bytes, errors and pending bytes are independent copies.

Every frame rejects invalid UTF-8, unpaired UTF-16 surrogate escapes, duplicate decoded keys at every depth, trailing JSON values, malformed JSON and more than 64 compound object/array levels. Escaped and literal spellings of the same key are duplicates. Valid surrogate pairs and literal U+FFFD remain valid; malformed escapes are refused before Go's JSON decoder can replace them and collapse distinct wire identities.

On malformed or oversized input, `Feed` returns preceding complete messages plus one rejected observation and an error. Failure permanently latches the decoder. Later bytes in that fragment are ignored; subsequent `Feed` calls return the same error and no new messages, and `End` returns that error. `Pending` returns a copy of only the unfinished or rejected frame prefix. It never includes ignored later input. A large caller fragment is scanned without cloning it wholesale before applying the line bound; retained pending bytes are limited to the configured limit. Returned message batches grow with caller input; there is no hidden accumulated history.

Oversized rejected observations have `Truncated:true` and contain at most the configured limit. Their digest identifies **only this prefix**, never the complete rejected frame. A complete malformed frame retains its original bounded bytes with `Truncated:false`.

`End` requires a real LF boundary. An unfinished frame returns and latches `ErrIncomplete`, retains the pending bytes and does not append an invented LF. The codec performs no redaction. The adapter must redact known secrets before every journal append while keeping true wire identity separate from permitted journal bytes. Codec evidence excludes engine commands, agentd frame headers and any reconstructed or redacted payload. None of these hashes grants dispatch, replay or recovery authority.

## Pinned primary sources and licensing

The official [Python SDK `v0.2.168`](https://github.com/anthropics/claude-agent-sdk-python/tree/a39c96edf8cc414e640620ff3360f17475db0bd9) annotated tag `ec03b9c11724eca17294dc43428e27e9e2ada0d9` resolves to commit `a39c96edf8cc414e640620ff3360f17475db0bd9`. Its `_version.py` records SDK `0.2.168`; `_cli_version.py` records bundled CLI `2.1.301`. The following complete consulted file bytes were hashed and independently matched to the Git blobs in that exact commit tree. Paths are relative to its repository root.

| Python source file | SHA-256 |
| --- | --- |
| `LICENSE` | `cebdde8a8fb9ee59e5eaaed19578bf8085aa7047562259c31f38f225c26f6812` |
| `pyproject.toml` | `3df959710f676c88552780b17cda4246112246d4f20bc51cf0c708743e848e9c` |
| `src/claude_agent_sdk/_version.py` | `2453d4fc95e75369c0eed5a15a98ae000dcfc7c350969744f5fb324571d2bb8c` |
| `src/claude_agent_sdk/_cli_version.py` | `370ddef0d1fe851fd81356ae97d13e1599eb4a539752958eecbdf813a3af58be` |
| `src/claude_agent_sdk/_internal/query.py` | `0562517de949f4431177f204fe84cda3265f495d0eb0a949b9ced96cc5bd8276` |
| `src/claude_agent_sdk/_internal/client.py` | `cbc59cdb774a28781245a0cb676543d1914e061657a1e99ec7574143f309948a` |
| `src/claude_agent_sdk/_internal/message_parser.py` | `febb1aee19e47e03433d89cd4b1f8c5636950e206df77e4f4ca2738b0c900393` |
| `src/claude_agent_sdk/_internal/transport/subprocess_cli.py` | `23c0b35acfcff5426255f1f17433e66fc7f8f6d5b8f339e93e085233c569e11c` |
| `src/claude_agent_sdk/types.py` | `3cdbd7040925238df636973e760aee941a61afbc0139fa014e254f797e1551ab` |

The Python repository's root license is MIT. These references establish selected control envelopes, user and response shapes and their pinned version relationship; they are not copied SDK implementation or a complete CLI schema.

The official published [TypeScript SDK `0.3.301` package metadata](https://unpkg.com/@anthropic-ai/claude-agent-sdk@0.3.301/package.json) independently declares `claudeCodeVersion:"2.1.301"`. Only static metadata, README and [published declarations](https://unpkg.com/@anthropic-ai/claude-agent-sdk@0.3.301/sdk.d.ts) were consulted, with whole-file hashes also checked against the versioned CDN file-integrity metadata. No SDK bundle, package archive or platform executable was downloaded or run.

| Published TypeScript file | SHA-256 |
| --- | --- |
| `package.json` | `78ac8f8708081485005016f2b0aae8ff0f7845ee25478ad2e58b8645a92f7b5a` |
| `sdk.d.ts` | `26f9a0c91ad64e2020f0d3734932302e3cc09685bddf2b2f9ef60839ce278773` |
| `README.md` | `786e57ced73510c752e2e15901d290c7c22786e4791d8a2e2d96991a9a97c192` |

The official TypeScript source tag `v0.3.301` points to `b0fc766997199a49f90c37f5ec056936c80c9491`. The published declarations are not proven bound to that Git tree and are identified here as published static declarations. That tree's [LICENSE.md](https://github.com/anthropics/claude-agent-sdk-typescript/blob/b0fc766997199a49f90c37f5ec056936c80c9491/LICENSE.md), SHA-256 `728158fd1037143fad6907e8fa34804177e598b7326519503fe83cafdef849e6`, states all rights reserved and Commercial Terms. Its [README](https://github.com/anthropics/claude-agent-sdk-typescript/blob/b0fc766997199a49f90c37f5ec056936c80c9491/README.md), SHA-256 `d7f8965afea6198e0448bb8a3f3e0806d4dd6cfb3696cab21b1ee36faaef2065`, confirms those terms. No open-source license is inferred for the TypeScript SDK or proprietary CLI.

No upstream SDK source, schema or protocol catalogue is vendored. Tests contain independently authored synthetic messages, not copied fixtures or captured provider traffic. This package adds no runtime dependency or SDK wrapper. The codec tests establish encoding, framing and selected projections; the Session tests establish controlled fake lifecycle behavior. Neither establishes a real provider handshake, runtime resume identity, recovery, runtime image support, Apply or release acceptance.

## Opt-in actual Session adapter smoke

Status: **UNEXECUTED**. The source in `session_smoke_test.go` prepares a bounded real adapter smoke. Compilation and skipped tests establish no runtime or provider result. Existing container CLI probes bypass this adapter and cannot supply this evidence.

The test calls actual `NewSessions(engine, resolver)` with explicit `run.SessionAdmission` and `run.StepAdmission` values, the current `run.ToolCatalog` and `run.NewStepContract`. Its `controlledSessionIO` is explicitly in-memory smoke scaffolding: its synthetic receipts acknowledge that observer only. It establishes no durable journal, Coordinator admission, Core integration, task acceptance, recovery or Apply authority.

A separately authorized execution admits at most three read-only steps under one five-minute context: manager Explore, manager resume in a new container using the same provider home, then a fresh reviewer Candidate step using a different home. Every step must make the expected Host outcome call with the observed artifact content; final text cannot satisfy it. Resume must retain the actual provider session ID and recall a random token supplied only in the first Host answer. A fresh reviewer must have a different observed provider ID. Reviewer resume remains forbidden by the current contract. Each step requires exact acknowledged wire permits, observed sends, terminal observation, actual Stop and readback of the unchanged one-file candidate. Initial materialization refuses image-populated workspace entries; all subsequent workspace mounts use `NoCopy` and are read-only.

Ordinary CI leaves the boolean command-line activation flag false. Runtime environment variables cannot enable it. The activation check precedes profile, engine or credential access. A future operator must supply all of these explicit flags after a finite runtime grant; no command below is an execution record:

| Flag | Explicit input |
| --- | --- |
| `-markitect-claude-session-smoke` | Boolean opt-in, default false. |
| `-markitect-claude-smoke-engine` | Absolute engine executable path. |
| `-markitect-claude-smoke-image` | Immutable `sha256:` image ID, checked against the engine observation. |
| `-markitect-claude-smoke-profile` | Absolute closed SessionProfile JSON source path. |
| `-markitect-claude-smoke-profile-sha256` | Expected whole source bytes SHA-256, lowercase hex without prefix. |
| `-markitect-claude-smoke-metadata-sha256` | Expected whole image `/opt/markitect/runtime.env` SHA-256, same format; the pinned provider version must also match. |
| `-markitect-claude-smoke-token-env` | Name of the environment variable holding a dedicated long-lived runtime token; no ambient token fallback. Only the explicit resolver reads its value after activation. |

The profile selects Claude Haiku (`claude-haiku-5-5`) with high effort and exactly `builtinTools:["Read"]`. These inputs bind this smoke only; profile provenance and the finite runtime grant remain Host obligations. Run this package's `TestAdmittedSessionAdapterSmoke` alone with `-count=1`, an external test timeout greater than the five-minute runtime bound plus cleanup, and the explicit flags through `go test ... -args`. Never place credential values in flags, output or committed files. Diagnostic output reports stage failures and source digests without logging credential values or raw provider transcripts. Created smoke volumes are removed with bounded cleanup; incomplete Stop or cleanup fails the test and retains uncertainty.
