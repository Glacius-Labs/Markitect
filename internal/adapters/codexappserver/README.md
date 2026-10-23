# Codex native sessions

The adapter implements static `runtimeconfig.AgentProfilePreparer` and the actual `run.Sessions` port. One prepared profile supplies one `Agents` instance. Outer composition chooses profiles; Project and Run do not import Runtime configuration. Source implementation does not establish provider qualification.

## Static preparation

`codexAppServer` in `markitect.runtime/v1` has a closed metadata profile. Version 2 requires `profileVersion`, absolute `engineExecutable`, immutable `imageID`, `agentdVersion`, `recordedCodexSchemaDigest` and ordered `instructions: [{boundary, path}]`. It is metadata-only. Version 3 additionally requires `credentialRef`, `maxWireBytes` (positive, at most 1 MiB) and the closed `session` object: `version: 1`, literal `model`, `effort`, `baseInstructions`, `developerInstructions`, `store`, positive `cpus`, `memoryBytes`, `pids`, `tmpBytes`, and explicit `network` (empty selects the engine default; host networking is refused).

The metadata reader inspects the selected image, copies bounded whole `/opt/markitect/runtime.env` bytes without starting the image, and checks its identity again. Required recorded values are Codex `0.162.0`, agentd version and Codex schema digest. Captured instructions are immutable inputs; a missing source requests complete reacquisition. No environment interpolation or credential resolution occurs here.

`ProfileFromPrepared(runtimeconfig.PreparedProfile)` admits only complete, source-bound v3 observations with the exact contract, provider, image, metadata path, versions, schema and captured instruction list. Static profile identity is distinct from actual native construction.

## Actual Run consumer

`NewAgents(engine, preparedProfile, credentialResolver)` and `NewAgentsWithRuntime(starter, preparedProfile, credentialResolver)` construct the concrete consumer. The injected starter must provide fresh private-home lifetime, actual immutable running-image observation and partial handles on error.

`Open(ctx, run.SessionSpec)` checks the exact prepared Manager, complete checked Project, supported tools and actual workspace location before resolving the selected opaque credential reference. Native execution accepts only an existing private Linux `Volume` plus `Subpath`, mounted directly at `/work` with no copy-up. A local `Directory` is explicitly unsupported. The native factory owns one separate fresh private home and one hardened Manager container. No host credential home is mounted.

One agentd process and one Codex App Server process live until `Close`; the first `Exchange` starts a fresh thread and subsequent messages use it. No old domain Turn, Pass, assignment scheduler or API-key execution route remains. The native CLI's own wire turn is a protocol fact.

## Harness, inbox and completion

The pure [harness](../../harness/README.md) supplies versioned rules, checked Project and complete hierarchy, exact Manager/context, mandate and Findings. The native renderer adds selected literal profile instructions and captured runtime instructions. These exact native base/developer strings determine `HarnessDigest`.

Launch arguments disable native delegation, apps, memories, image generation, browser and computer use; keep ordinary native shell/file tools; set `project_doc_max_bytes=0`, no project-document fallback filenames and `/work` untrusted. Private `CODEX_HOME` contains only explicitly provisioned account login. These controls require the separately bounded effective-config/conflicting-instruction proof; text construction alone does not establish loader priority.

Native MCP tools directly call Run's `delegate`, `await`, `review`, `ask`, `reply`, `read_notes` and `append_note` callbacks. Ask returns its ID immediately. Await delivers actual parent tasks, questions, answers and fixed child handoffs; replies are authored through the same native conversation. Unknown tools and foreign-thread calls fail. Tool refusals are returned to the native provider; callbacks drain before Exchange returns.

Each final answer is the explicit decision JSON `{ "decision": "submitted", "findings": [] }`, with `approve`, `reject` and `model_blocker` as the other decision values. After native completion, the adapter uses the pinned experimental `thread/backgroundTerminals/list`, terminates observed terminals, and requires a complete empty readback within the same deadline. Queued `clean` acknowledgment alone is not drain evidence. Native writer termination and the limits of detached/untracked background work require the real capability proof before fixed-handoff qualification.

`Observation()` returns only the actual constructed session's `ProfileIdentity`, confirmed `ImageID` and `HarnessDigest`. Failed construction exposes no fabricated observation. Close cancels an active Exchange, joins it, observes provider shutdown, clears installed login after observed exit, closes the container and removes only the owned home. Cleanup uncertainty remains an error.

## Login and evidence limits

Only explicit `dedicated-runtime-login` input with `auth_mode: chatgpt`, account ID, ID token and access token is admitted. Mixed API-key authority, desktop mode and malformed login are refused. Refresh authority is removed from the Manager copy; this adapter does not refresh or rewrite the source. Credentials travel only through bounded private agentd initialization, not argv, environment, harness, Notes or product Git. Known tokens and their escaped forms are redacted. Fresh-login, expiry, refresh and fan-out support remain separately qualified.

Pure codec, metadata and controlled-session tests establish source behavior. Historical private-init and Inbox probe binaries/evidence remain tied to their named pre-replacement commits; their old Turn-based test consumers were retired. Current LinuxGit, effective loading, native hierarchy, background-terminal drain, provider support, full Apply, hosted CI and owner acceptance remain separate claims.
