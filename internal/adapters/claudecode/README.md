# Claude Code stream codec

This package supplies pure in-memory encoding and NDJSON framing for selected Claude Code **2.1.301** stream-JSON shapes. It does not implement `run.Sessions` or perform I/O. It launches no CLI, reads no credentials, executes no tools and creates no provider, Host, step or message identities. Container transport, Host tool authority, readiness receipts, session binding, exact step correlation, history inspection and recovery belong to later adapters and Host consumers.

## Selected API and wire shapes

`NewCodec(limit)` bounds each frame, including its actual LF, to between 1 byte and 1 MiB. Use each codec from one consumer. It retains no request map or message history.

`EncodeInitialize(ControlRequestID)` emits a `control_request` with the supplied `request_id` and an `initialize` request subtype, with no optional hooks, agents or SDK capabilities. `EncodeUser(UserInput)` emits one literal text user message with `parent_tool_use_id:null`. UUID and requested session ID are omitted unless the caller supplies them. Explicit `ClientComposed:true` emits the published `client_composed:true`; otherwise that field is omitted. Empty text is a literal caller input, never an automatically inserted bootstrap prompt. String inputs are checked for UTF-8 validity and aggregate input size before JSON encoding; encoded output must also fit the LF-inclusive limit. Each encoding returns fresh bytes.

`Feed` yields these distinct observations:

| Kind | Selected validation and projection |
| --- | --- |
| `ControlResponseMessage` | `response.subtype`, nonempty `response.request_id`, success object payload when present, or error string. Unknown response subtypes remain unsupported. |
| `SystemInitMessage` | `system/init` with nonempty `session_id`, `uuid` and `claude_code_version`. `Init()` returns these observations. |
| `UserMessage`, `AssistantMessage` | Message object, matching role and string or array user content; assistant content is an array. Content blocks remain opaque. Assistant UUID/session ID are required; optional user UUID/session ID are checked when supplied. |
| `StreamEventMessage` | Object event, UUID/session ID and parent tool-use ID envelope. Event contents remain opaque. |
| `ResultMessage` | Known result subtype, boolean `is_error`, session ID and result UUID; success result text or error-subtype errors array. Optional text, errors, structured output and terminal reason are retained by `Result()`. |
| `UnsupportedRequest` | Server control request with nonempty ID and subtype; payload remains raw. No reply or execution API. |
| `UnsupportedMessage` | Other message types and system/result/control-response subtypes, with original bytes visible. |

The user/assistant/stream-event projection checks the required `parent_tool_use_id` as null or a nonempty string. These are selected projections, not full SDK schema validation. Unknown sibling fields, content blocks, tool data, account details, usage and other unconsumed data remain in the owning raw observation. The full JSON, including opaque fields, still passes the framing guards below. Known malformed selected shapes yield `RejectedMessage` and a decoder error, never a successful fallback.

`ReadInitializeResponse(expected)` requires the exact supplied control request ID. A success response must contain `commands`, `agents` and `models` arrays, `output_style` string, `available_output_styles` string array and `account` object. Array contents and account contents remain opaque. The optional `claude_code_version` must be nonempty text. An error response returns `ControlError` and its error text; it is not a successful acknowledgment. A general success acknowledgment without this selected payload fails the initializer projection. Projection errors do not alter the retained message or decoder state.

An initialization reply can precede the first user message. It supplies **no observed provider session ID**, and this codec supplies neither a RUN `Ready` nor a `Bind` receipt. Only the separately decoded `system/init` projection supplies a session-binding observation. A configured ID is still only `RequestedSessionID` until a future consumer confirms it against the actual observation. Version text is an observation, not an enforced runtime pin or handshake proof; preparation and a later adapter must enforce the selected image/profile/version. Repeated or differing session observations require later adapter reconciliation.

`ControlRequestID`, `RequestedSessionID`, `ObservedSessionID`, `UserMessageUUID`, ordinary `MessageUUID` and `ResultUUID` are separate named types. The codec generates none of them and provides no Host-step or provider-step conversion. Published optional `user_message_uuid` and `user_message_uuids` echoes remain in raw bytes. The published declarations describe batching, multiple consumed prompts and absent echoes; no one-to-one Host-step mapping follows from a result UUID or an optional echo.

Known result subtypes are `success`, `error_during_execution`, `error_max_turns`, `error_max_budget_usd` and `error_max_structured_output_retries`. `is_error` remains independent: **a `success` subtype can have `is_error:true`**, and does not establish successful work. `terminal_reason` is uninterpreted published status text, including unfamiliar future values; the codec infers no additional outcome from it. A result observation establishes neither task acceptance nor quiescence, process termination, successful cancellation, inspectable exact-step history or recovery. Unknown result subtypes remain unsupported and cannot be read as `ResultObservation`.

## Framing and immutable evidence

`Feed` accepts fragmented or multiple frames, including fragments within UTF-8 characters. A line includes its LF in the byte limit. CRLF and surrounding JSON whitespace are accepted without normalization. `Message.Raw()` returns a copy of the exact provider NDJSON bytes, including LF and any CR. `Digest()` is SHA-256 of exactly those bytes. Returned result structured bytes, errors and pending bytes are independent copies.

Every frame rejects invalid UTF-8, unpaired UTF-16 surrogate escapes, duplicate decoded keys at every depth, trailing JSON values, malformed JSON and more than 64 compound object/array levels. Escaped and literal spellings of the same key are duplicates. Valid surrogate pairs and literal U+FFFD remain valid; malformed escapes are refused before Go's JSON decoder can replace them and collapse distinct wire identities.

On malformed or oversized input, `Feed` returns preceding complete messages plus one rejected observation and an error. Failure permanently latches the decoder. Later bytes in that fragment are ignored; subsequent `Feed` calls return the same error and no new messages, and `End` returns that error. `Pending` returns a copy of only the unfinished or rejected frame prefix. It never includes ignored later input. A large caller fragment is scanned without cloning it wholesale before applying the line bound; retained pending bytes are limited to the configured limit. Returned message batches grow with caller input; there is no hidden accumulated history.

Oversized rejected observations have `Truncated:true` and contain at most the configured limit. Their digest identifies **only this prefix**, never the complete rejected frame. A complete malformed frame retains its original bounded bytes with `Truncated:false`.

`End` requires a real LF boundary. An unfinished frame returns and latches `ErrIncomplete`, retains the pending bytes and does not append an invented LF. The codec performs no redaction. A later Host consumer must redact known secrets before every journal append while keeping true wire identity separate from permitted journal bytes. Codec evidence excludes engine commands, agentd frame headers and any reconstructed or redacted payload. None of these hashes grants dispatch, replay or recovery authority.

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

No upstream SDK source, schema or protocol catalogue is vendored. Tests contain independently authored synthetic messages, not copied fixtures or captured provider traffic. This package adds no runtime dependency or SDK wrapper. Its source tests establish encoding, framing and selected projections only; they establish no real provider handshake, Session readiness, recovery, runtime image support, Apply or release acceptance.
