# Claude Code turn adapter and stream codec

This package implements the `run.Agents` port for Claude Code **2.1.301** and a pure in-memory codec for selected stream-JSON shapes. The adapter uses the existing container runtime and agentd MCP transport. It has no CLI/bootstrap wiring and establishes no real provider or Apply acceptance.

## Captured Host profile

`NewMetadataPreparer(ImageReaderFactory)` prepares the `claudeCode` section selected by Apply. The closed adapter-local format uses `profileVersion: 1`; that number is independent of the Codex format and the `session.version`. All fields below are required. No metadata-only or legacy format supplies execution settings.

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

`ProfileFromPrepared(prepared)` returns a `Profile` (`ImageID`, `CredentialRef`, `MaxWireBytes`, `Session`) only after complete source-bound preparation. It rechecks the selected Claude section, exact input binding, contract, provider, image, versions and instruction locations, then validates the result. Model, effort, prompt, tools, credential and resource values come only from the captured fields; static preparation grants no execution authority. `Session` is the closed `SessionProfile`: `version` (1), `model` and `systemPrompt` (explicit nonblank), `effort` (`low`, `medium` or `high`), `builtinTools` (an explicit array, possibly empty, of distinct names from `Read`, `Write`, `Edit`, `Glob`, `Grep`, `Bash`), `store`, positive finite `cpus`, positive `memoryBytes`, `pids` and `tmpBytes`, and `network` (never `host`). Nothing is inferred; invalid or absent values refuse.

The metadata and projection tests use the actual Apply consumer with synthetic captured model/profile inputs and controlled image readers. They establish source behavior, including source identity. They provide no real engine, credential, provider, Apply or owner acceptance evidence.

## One turn per container

`NewAgents(engine, profiles, credentials)` receives the container engine, the named `Profile` values and a Host `CredentialResolver`; `NewAgentsWithRuntime` takes a `RuntimeStarter` instead, for controlled fake evidence. `(*Agents).Run(ctx, run.Turn, run.ToolHandler)` resolves the turn's profile by name and validates the turn before any container starts: an unknown profile, a malformed turn or tool, a missing credential or an oversized prompt is an error with no container, reported as `Stopped` with `Reason` "no container started: ..." because there is nothing to stop. A verify turn must be read-only and never resumes.

One turn is one container named `markitect-<turn ID>` from the profile's image, labelled with the profile's store, run as `1000:1000` with workdir `/work`. The turn's workspace volume is mounted at `/work` with copy-up disabled, read-only when `Turn.ReadOnly`; the home volume is mounted at `/home/agent`. Resources and network come from the profile. A read-only turn drops `Write` and `Edit` from the builtin tools; the rest stay, since the mount is read-only.

The provider is `/usr/local/bin/claude` with stream-JSON in and out, a strict MCP config that starts `markitect-agentd mcp`, `--tools`, `--allowedTools` (builtin tools plus `mcp__markitect__<tool>` for every turn tool), `dontAsk`, the profile's model and effort, `--setting-sources ""`, `--system-prompt`, and `--resume <id>` when `Turn.Resume` is set (see the [existing runtime probes](../container/README.md) and the [official CLI flags](https://code.claude.com/docs/en/cli-reference)). The Host resolves exactly `CredentialRef` to `RuntimeCredential.Token`, a dedicated long-lived runtime token. It travels only as `CLAUDE_CODE_OAUTH_TOKEN` in `frame.Start.Env`, never in argv, container environment or profile. The adapter reads no ambient credentials. The known token is also removed from tool-call arguments before they reach the handler.

Run sends the initialize control request, waits for its successful acknowledgment, then sends the prompt as one user message. Host tool calls arrive through the container's agentd connection and are answered serially through the handler as an MCP text result, with `isError` when the answer is `Refused`. A call for a tool outside the turn's list gets a JSON-RPC error. The turn ends at the result message. `TurnResult.Conversation` is the `system/init` session ID; when resuming, a different observed ID is accepted and recorded. `Final` is the result text; `Usage` comes from init (model), the assistant messages as they arrive (tokens, each message counted once by its ID) and the result message (tokens, `total_cost_usd`, `num_turns`), whose totals replace the streamed sum. A turn that ends without a result message (timeout, cancellation, crash) or whose result carries no usage reports the streamed sum, a lower bound without cost, and says "usage partial: ..." in `Reason`. A result with `is_error` returns its usage and an error.

Before returning in every case, Run stops the container through the runtime (`Stop`, then `Close`). `TurnResult.Stopped` is true only when the provider exit was actually observed by `Stop` and `Close` ended the container without having to remove it by force (`container.ErrForcedClose`); otherwise it reports false with an error. `Turn.Timeout` and cancellation of `ctx` end the turn the same way and return the partial result with the context error. `Reason` names how the turn ended and what was observed of the exit.

When `Turn.Transcript` is set (an absolute path; parent directories are created), Run writes JSON Lines of every provider stream line, sent message and Host tool call and answer. The known token and its JSON-escaped spelling are redacted from the transcript and from every returned error and reason. The token is never logged.

The fake tests exercise exact spec, invocation and environment for execute and read-only verify turns and resume, tool round trips and refusals, usage and conversation extraction, timeout, cancellation and uncertain stop, validation, and redaction. They run no Docker, provider, credential, Apply or release experiment.

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

An initialization reply can precede the first user message. It supplies **no observed provider session ID**. Only the separately decoded `system/init` projection supplies the session observation; the adapter above records it as the conversation. A configured ID is still only `RequestedSessionID` until observed. Version text is an observation; the adapter requires `2.1.301` in the init and the acknowledgment, and preparation enforces the selected image version.

`ControlRequestID`, `RequestedSessionID`, `ObservedSessionID`, `UserMessageUUID`, ordinary `MessageUUID` and `ResultUUID` are separate named types. The codec generates none of them and provides no Host-step or provider-step conversion. Published optional `user_message_uuid` and `user_message_uuids` echoes remain in raw bytes. The published declarations describe batching, multiple consumed prompts and absent echoes; no one-to-one Host-step mapping follows from a result UUID or an optional echo.

`rate_limit_event` is the pinned TypeScript SDK's four-key informational envelope: `type`, object `rate_limit_info`, and nonempty `uuid` and `session_id`. The payload stays opaque. The adapter treats it as provider progress: it changes no turn state and supplies no outcome. A real 2.1.301 subscription-token smoke reported this type and four top-level keys; the detailed envelope comes from the exact published `sdk.d.ts` hash recorded below. The regression uses independently authored synthetic events. The corrected adapter still needs the separately authorized real smoke rerun.

Known result subtypes are `success`, `error_during_execution`, `error_max_turns`, `error_max_budget_usd` and `error_max_structured_output_retries`. `is_error` remains independent: **a `success` subtype can have `is_error:true`**, and does not establish successful work. `terminal_reason` is uninterpreted published status text, including unfamiliar future values; the codec infers no additional outcome from it. A result observation establishes neither task acceptance nor quiescence, process termination, successful cancellation, inspectable exact-step history or recovery. Unknown result subtypes remain unsupported and cannot be read as `ResultObservation`.

`system/thinking_tokens` is approximate thinking progress, not authoritative billed usage. A real Claude Code 2.1.301 smoke reported it after a Read tool result with exactly `type`, `subtype`, `estimated_tokens`, `estimated_tokens_delta`, `session_id` and `uuid` ([failure report](https://github.com/Glacius-Labs/Markitect/pull/388#issuecomment-6241420190)). The pinned `SDKThinkingTokensMessage` declaration in the exact `sdk.d.ts` below declares both estimates as numbers, without an integer or sign constraint. The selected codec accepts finite JSON numbers and only these six observed keys; the SDK's optional `user_message_uuid` extension remains outside this bounded shape. The adapter treats it as provider progress and records it in the transcript. Unknown system subtypes are ignored as unsupported messages; malformed selected shapes still refuse. Synthetic codec regressions cover this correction; successful source checks do not establish a corrected real provider run.

## Framing and immutable evidence

`Feed` accepts fragmented or multiple frames, including fragments within UTF-8 characters. A line includes its LF in the byte limit. CRLF and surrounding JSON whitespace are accepted without normalization. `Message.Raw()` returns a copy of the exact provider NDJSON bytes, including LF and any CR. `Digest()` is SHA-256 of exactly those bytes. Returned result structured bytes, errors and pending bytes are independent copies.

Every frame rejects invalid UTF-8, unpaired UTF-16 surrogate escapes, duplicate decoded keys at every depth, trailing JSON values, malformed JSON and more than 64 compound object/array levels. Escaped and literal spellings of the same key are duplicates. Valid surrogate pairs and literal U+FFFD remain valid; malformed escapes are refused before Go's JSON decoder can replace them and collapse distinct wire identities.

On malformed or oversized input, `Feed` returns preceding complete messages plus one rejected observation and an error. Failure permanently latches the decoder. Later bytes in that fragment are ignored; subsequent `Feed` calls return the same error and no new messages, and `End` returns that error. `Pending` returns a copy of only the unfinished or rejected frame prefix. It never includes ignored later input. A large caller fragment is scanned without cloning it wholesale before applying the line bound; retained pending bytes are limited to the configured limit. Returned message batches grow with caller input; there is no hidden accumulated history.

Oversized rejected observations have `Truncated:true` and contain at most the configured limit. Their digest identifies **only this prefix**, never the complete rejected frame. A complete malformed frame retains its original bounded bytes with `Truncated:false`.

`End` requires a real LF boundary. An unfinished frame returns and latches `ErrIncomplete`, retains the pending bytes and does not append an invented LF. The codec performs no redaction. The adapter redacts known secrets before anything is written to the transcript. Codec evidence excludes engine commands, agentd frame headers and any reconstructed or redacted payload.

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

No upstream SDK source, schema or protocol catalogue is vendored. Tests contain independently authored synthetic messages, not copied fixtures or captured provider traffic. This package adds no runtime dependency or SDK wrapper. The codec tests establish encoding, framing and selected projections; the turn tests establish controlled fake lifecycle behavior. Neither establishes a real provider handshake, runtime resume identity, runtime image support, Apply or release acceptance.

## Opt-in real turn smoke

Status: **UNEXECUTED**. `TestClaudeTurnSmoke` in `agents_smoke_test.go` runs one real read-only verify turn through `NewAgents`: a temporary volume holds one file, one Host tool `report` is offered, and the agent must read the file and report its exact content. The test checks the report, the observed conversation, usage, an observed stop and a token-free transcript, and removes its volumes. Compilation and a skipped test establish no runtime or provider result.

It runs only with explicit command-line flags; no environment variable enables it, and the activation check precedes any engine or credential access:

| Flag | Explicit input |
| --- | --- |
| `-markitect-claude-turn-smoke` | Boolean opt-in, default false. |
| `-markitect-claude-smoke-engine` | Absolute engine executable path. |
| `-markitect-claude-smoke-image` | Immutable `sha256:` image ID, checked against the engine observation. |
| `-markitect-claude-smoke-profile` | Absolute closed `SessionProfile` JSON file; it must offer `Read`. |
| `-markitect-claude-smoke-token-env` | Name of the environment variable holding a dedicated runtime token. |

Run the test alone with `-count=1`, an external timeout above its five-minute bound, and the flags through `go test ... -args`. Never place credential values in flags, output or committed files.
