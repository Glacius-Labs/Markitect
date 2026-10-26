# Codex native sessions

The adapter implements static `runtimeconfig.AgentProfilePreparer` and the actual `run.Sessions` port. One prepared profile supplies one `Agents` instance. Outer composition chooses profiles; Project and Run do not import Runtime configuration. Source implementation does not establish provider qualification.

## Static preparation

`codexAppServer` in `markitect.runtime/v1` has a closed metadata profile. Version 2 requires `profileVersion`, absolute `engineExecutable`, immutable `imageID`, `agentdVersion`, `recordedCodexSchemaDigest` and ordered `instructions: [{boundary, path}]`. It is metadata-only. Version 3 additionally requires `credentialRef`, `maxWireBytes` (positive, at most 1 MiB) and the closed `session` object: `version: 1`, literal `model`, `effort`, `baseInstructions`, `developerInstructions`, `store`, positive `cpus`, `memoryBytes`, `pids`, `tmpBytes`, and explicit `network` (empty selects the engine default; host networking is refused).

The metadata reader inspects the selected image, copies bounded whole `/opt/markitect/runtime.env` bytes without starting the image, and checks its identity again. Required recorded values are Codex `0.162.0`, agentd version and Codex schema digest. Captured instructions are immutable inputs; a missing source requests complete reacquisition. No environment interpolation or credential resolution occurs here.

`ProfileFromPrepared(runtimeconfig.PreparedProfile)` admits only complete, source-bound v3 observations with the exact contract, provider, image, metadata path, versions, schema and captured instruction list. Static profile identity is distinct from actual native construction.

## Actual Run consumer

`NewAgents(engine, preparedProfile, credentialResolver)` and `NewAgentsWithRuntime(starter, preparedProfile, credentialResolver)` construct the concrete consumer. The injected starter must provide fresh private-home lifetime, actual immutable running-image observation and partial handles on error.

`Open(ctx, run.SessionSpec)` checks the exact prepared Manager, complete checked Project, supported tools and actual workspace location before resolving the selected opaque credential reference. Native execution accepts only an existing private Linux `Volume` plus `Subpath`, mounted directly at `/work` with no copy-up. A local `Directory` is explicitly unsupported. The native factory owns one separate fresh private home and one hardened Manager container. No host credential home is mounted.

One agentd process and one Codex App Server process live until `Close`; the first `Exchange` starts a fresh thread and subsequent messages use it. The native CLI's own wire turn is a protocol fact; Markitect adds no Turn layer and no API-key route.

## Harness, inbox and completion

The pure [harness](../../harness/README.md) supplies versioned rules, checked Project and complete hierarchy, exact Manager/context, mandate and Findings. The native renderer adds selected literal profile instructions and captured runtime instructions. These exact native base/developer strings determine `HarnessDigest`. Optional `TranscriptRoot` is an explicit absolute evidence directory fixed before Open; fresh exclusive transcript files redact login tokens in both calls and native prose.

Launch arguments disable native delegation, apps, memories, image generation, browser and computer use; keep ordinary native shell/file tools; set `project_doc_max_bytes=0`, no project-document fallback filenames and `/work` untrusted. Private `CODEX_HOME` contains only explicitly provisioned account login. These controls require the separately bounded effective-config/conflicting-instruction proof; text construction alone does not establish loader priority.

Nine native MCP tools directly call Run's fixed tool set: `delegate`, `await`, `review`, `ask`, `reply`, `judge`, `check`, `read_notes` and `append_note`. Each has one implementation in Run; the adapter only maps it. Descriptions state each tool's purpose and arguments, not host process states. Findings are objects with exactly one `text` string: Run sets their origin. Ask returns its ID immediately. Await delivers actual parent assignments, questions, answers and fixed child handoffs with their check results. Unknown tools and foreign-thread calls fail. Invalid arguments and Run refusals return to the native provider as tool errors with their reason; callbacks drain before Exchange returns. `read_notes` takes an optional `after` cursor and returns one bounded page with `next`, so every reply fits the transport frame. Exact required keys, casing and field types are validated before any callback, including durable Notes writes.

Each Exchange sends Run's message as the JSON prompt: delivered events and Run's diagnostics. Its end carries no judgment: the Manager judges through `judge`, which Run validates when it is called; final prose is independent of it. After native completion, the adapter uses the pinned experimental `thread/backgroundTerminals/list`, terminates observed terminals, and requires a complete empty readback within the same deadline. Queued `clean` acknowledgment alone is not drain evidence. Fatal native notifications are preserved during handshake and terminal-drain RPC waits as well as during inference. Native writer termination and the limits of detached/untracked background work require the real capability proof before fixed-handoff qualification.

`Observation()` returns only the actual constructed session's `ProfileIdentity`, confirmed `ImageID` and `HarnessDigest`. Failed construction exposes no fabricated observation. `Acted()` reports whether the last Exchange saw any completed native item other than text (`userMessage`, `agentMessage`, `reasoning`) – a command, file change, tool call, web search or subagent – whatever its outcome and also on a native subagent thread; Run uses it only to tell an idle response from one that did something. Close cancels an active Exchange, joins it, observes provider shutdown, clears installed login after observed exit, closes the container and removes only the owned home. Cleanup uncertainty remains an error.

## Transient provider errors

A native `error` notification with `willRetry: false` ends the Exchange with its redacted message, unless its `codexErrorInfo` kind is transient. The closed list of transient kinds is `serverOverloaded` ("Selected model is at capacity"), which ended three Phase C Applies. A new entry needs observed evidence; every other kind, or none, stays final. After a transient error the adapter waits up to 30 s for the failed native turn to end; a turn that does not end ends the Exchange with the cause. What the response had done by then decides what follows:

- Nothing yet: no Run tool call was served, and no native item completed or is still running. The adapter sends the same message again in the same thread. A response gets at most two extra attempts, after waits of 30 s and 120 s. A wait that would end after the Run deadline does not start. Waiting is host time: `run.HostTime` pauses Run's clock of the response.
- Acted: Run tool calls were served or native executions completed. These actions are never repeated. A later continuation must start from the actual thread, workspace and assignment state. Whether a native thread accepts a new turn after such a failure awaits a native probe; until then the Exchange ends with the cause.
- Unclear: a native item started and did not complete, including a Run tool call on its way to the host, or a tool call reached the host after the error or during the wait. That call stays unserved. The Exchange ends with the cause.

Each repeat adds a `retry` record to the transcript. `Agents.ProviderRetries()` counts the repeats across the profile's sessions for the Apply report. A study runner's repeats of whole Applies are separate.

## Login and evidence limits

Only explicit `dedicated-runtime-login` input with `auth_mode: chatgpt`, account ID, ID token and access token is admitted. Mixed API-key authority, desktop mode and malformed login are refused. Refresh authority is removed from the Manager copy; this adapter does not refresh or rewrite the source. Credentials travel only through bounded private agentd initialization, not argv, environment, harness, Notes or product Git. Known tokens and their escaped forms are redacted. Fresh-login, expiry, refresh and fan-out support remain separately qualified.

Pure codec, metadata and controlled-session tests establish source behavior. Current LinuxGit, effective loading, native hierarchy, background-terminal drain, a native repeat after a provider failure, provider support, full Apply, hosted CI and owner acceptance remain separate claims.
