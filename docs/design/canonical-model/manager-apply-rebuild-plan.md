# Agent runtime, session adapters and Apply rebuild plan

This plan owns the Agent runtime, the session adapters, their session mapping and the map of what happens to the current source. The [Go architecture](manager-apply-go-architecture.md) and [port contracts](manager-apply-port-contracts.md) define the target; the [delivery plan](../../implementation-plan.md) owns the order of work. Architectural correctness and clear responsibilities take priority over saving implementation effort.

Historical replacement baseline: Main `bc4b49e76f9b9e4b3ec98033a0b8c73d5013b310`. The source replacement map refers to that revision; it is not current implementation status. The runtime sections specify target obligations. Evidence in implementation PRs and capability reports establishes each supported behavior separately; process inspection, protocol framing and a successful resume case do not together prove all interruption or workspace-transition cases.

## Container runtime

Every Agent session and every project check runs in its own Linux container ([DEC-037](../../concepts/register.md#dec-037-agent-sessions-and-checks-run-in-linux-containers)). The Host profile names the runtime image by its engine image ID, the content digest of the image (`sha256:...`), which also exists for locally built images. The image contains:

- the session helper `markitect-agentd`;
- the pinned Codex and Claude Code CLIs;
- the project's toolchain.

A project builds its image on the product's base image, so sessions and checks see the same tools as Linux CI ([DEC-013](../../concepts/register.md#dec-013-linux-first-for-tests-and-the-playground)). Real Apply requires Docker or a compatible engine; controlled fake sessions in tests need none. The [handshake record](../../validation/runtime-handshake-20261017.md) holds the evidence behind this choice.

| Concern | Rule |
| --- | --- |
| Authority | Access follows the role, fixed execution authority ceiling and admitted step, not the Manager label alone. Until every initial Explore assessment is present, all artifact volumes are mounted read-only and the Host denies artifact-mutating tools and workspace operations. A writable temporary directory grants no artifact write authority. After the barrier, an admitted writer gets its owned workspace writable; a reviewer gets its frozen subject read-only, and a check gets a disposable copy. Independent expectations never enter a writer's container. Model, configuration and registry remain protected inputs: their mutation fails at Freeze, like any write outside claims. No container receives the owner's working tree, engine socket or another host path. |
| Isolation | The provider's own sandbox is off, because the container is the boundary: Codex runs with `danger-full-access`, Claude Code with an explicit tool allowlist. Every container runs as a non-root user with all capabilities dropped, `no-new-privileges` and a read-only root filesystem; only explicitly writable mounts, provider home and temporary space are writable. A fresh conversation starts with an empty provider home. A proven conversation continuation uses only its exact retained, Host-bound state, never unrelated desktop state. User configuration, skills, plugins, memories and MCP servers reach a session only when the profile supplies them. |
| Process lifetime | agentd runs the provider in its own process group and stops that group first, politely and then forcibly; stopping the container ends every remaining process. agentd stops the provider when its connection to the Host closes, so a crashed Host leaves no writer running. The profile bounds CPU, memory, process count and output size. |
| Volumes and recovery | Each Manager conversation has its own provider-home volume, which outlives every container; each new session of that conversation, and `Inspect`, mount it in a new container, and every review gets a fresh one. Every container and volume carries the label of its Host store. On start the Host reconciles labeled containers it does not control: it stops them and records the observed termination before an unfinished run continues. A run's volumes are kept while the run is unfinished and deleted when it finishes, after their evidence is in the journal and the state store. |
| Files | Workspaces live on Linux volumes that the runtime owns. Inputs enter as Git objects of the captured commit. Candidates leave as Git objects into the Host's private state store, never into the owner's `.git`. Only guarded publication writes to the owner's working tree, and it writes each file the way Git checks it out in that repository, applying its attributes, line-ending settings and executable bits. Windows short names, case aliases, reparse points, line-ending conversion and lost file modes therefore never occur inside a session. |
| Credentials | A runtime credential is an API key or a long-lived token that the Host profile references, never a copied desktop login. Providers rotate refresh tokens, so a copy can sign the desktop out. The Host sends the value to agentd in the start control message, never as container environment or command argument, and agentd gives it only to the provider process. The journal records that message with the credential's reference and kind instead of its value, and the Host redacts known secret values from every payload before appending it. Where the CLI allows, as with Codex's shell environment policy, the provider's tool processes do not inherit the credential. The Agent can still reach it, so a runtime credential should be scoped and spend-capped. Codex subscription logins for concurrent sessions are [OQ-010](../../concepts/register.md#open-questions). |
| Network | Sessions need egress to their provider. The first version does not restrict other egress; an allowlist is a later extension. |

`markitect-agentd` is the container's entrypoint and the only program the Host talks to. It has three jobs:

- run the provider CLI as its child;
- serve the Host tool catalogue to that CLI: the CLI starts `markitect-agentd mcp` as its stdio MCP server, and that bridge connects back to the main agentd over a Unix socket inside the container;
- carry four channels over the container's single stdio connection: the provider's own protocol, the provider's diagnostic output, Host tool calls, and control (start, stop, failures and exit status).

There is no network port and no host path, so the same transport works on every host operating system. agentd adds no policy. It relays bytes, reports exits and observation limits, and refuses a Host whose frame-protocol version differs from its own; a product release therefore needs a new image only when that protocol changes.

## Session adapters

Two adapters implement `run.Sessions`:

- Codex App Server in `internal/adapters/codexappserver`;
- Claude Code in stream-JSON mode in `internal/adapters/claudecode`.

Each role profile selects one; the [configuration](manager-apply-configuration.md#project-execution-file) states how reviewer and Manager profiles relate. Markitect owns the organization, scheduling, context, authority, review policy and accepted results; the CLI supplies the coding workflow inside one session.

Host tools reach both CLIs through MCP. Claude Code has no other client-handled tool mechanism, and Codex's dynamic tools are experimental. One protocol keeps one tool implementation, and a further CLI with MCP support can become an adapter without new tool code.

| Alternative | Treatment |
| --- | --- |
| Codex dynamic tools | Not used. They work in `codex-cli 0.162.0` but are gated as experimental, and Claude Code has no counterpart. |
| Sessions directly on the host operating system | Not supported for real Apply. They inherit user configuration and need per-system process, path and sandbox handling ([handshake record](../../validation/runtime-handshake-20261017.md)). |
| Existing native App Server adapter | Keep the transport framing and recovery mechanisms; replace the invocation, prompt and report design. |
| Existing `codex exec` process adapter and Claude wrapper | Remove. They exchange one invocation and a final report instead of the Host session protocol. |
| Direct model API with a new coding loop | A possible later adapter. It would make Markitect responsible for a coding loop and its tools. |
| External exchange packets | Remove; outside the first real Apply. |

The choice rests on the session boundary and Host interaction, not on the amount of existing Codex code. A new adapter must demonstrate the new contracts even where old tests pass.

## Session lifetime and protocol mapping

One Host execution session owns one container and its agentd; its provider conversation is identified separately. The same execution session can serve several known steps, including a quiescent workspace-basis advance supported by the runtime and step contract. It may sit idle while the Manager waits. A changed working basis alone should not force conversation loss. A changed mount/access plan, profile or runtime requires a newly bound execution environment, not a live unrecorded remount. Continuing a known conversation there requires demonstrated provider resume support and exact retained state; otherwise start a visibly new conversation with an evidence-linked handoff. Local and integration reviews always receive fresh conversations. There is no shared container pool or provider-native Manager hierarchy.

Quiescence means no unresolved tool, background writer or external effect can race the workspace transition. Turn completion, cancellation acknowledgement or a stopped foreground process alone is insufficient. Keep the old basis, own delta and transition intent until actual materialization and recorded confirmation agree. If the runtime cannot establish quiescence while retaining the process, stop and reconcile the affected execution environment; use only proven conversation continuation or an explicit handoff afterward. Process suspension by itself does not invalidate old file handles or caches. Provider hidden context is not proof that it adopted a new basis.

For a basis transition, first close admission of new affected steps and tool effects, then reconcile all already admitted actions and establish an effective artifact-write barrier that lasts through materialization and confirmation. Bind the observation to the exact execution session, step and workspace version. agentd's process census is supporting evidence only: the provider itself may write or start another process, and a Host-side action can outlive its caller. Delayed callbacks cannot gain current authority by arriving after the census. If the runtime cannot prove the barrier with a live provider, observe termination of the environment and reconcile remaining Host/external effects before switching the basis. Stopping a container alone does not resolve an already dispatched remote effect. An interrupted transition retains its old basis, own changes and pending effect records until actual state is reconciled.

Logical Managers do not require permanently resident containers. Follow the [finite idle policy](manager-apply-execution.md#idle-execution-and-resumption): after proven quiescence keep at most the bounded warm set, then stop by deadline or deterministic eviction. Retained provider-home bytes are only one input to resume, not proof that the pinned provider can restore the conversation or adopt the new workspace. Resume needs exact identity and capability evidence; otherwise hand off explicitly. All replacement starts remain charged. This target requires implementation and evidence; current process inspection, stop or provider framing alone does not establish it.

The protocol table below describes fresh startup. It does not specify or prove provider resume, permission changes, quiescent workspace updates or their recovery; those need bounded capability evidence before dependent implementation. The [step-basis contract](manager-apply-execution.md#step-basis-and-continued-conversations) owns admission and retained accounting. The earlier whole-session candidate binding remains the current source boundary until its versioned replacement is implemented.

| Host port operation | Codex App Server | Claude Code |
| --- | --- | --- |
| `Open` | Start the container with the role's mounts, including the plan's provider home. agentd starts `codex app-server`; the adapter initializes, starts a fresh thread with the profile's settings and the Host MCP server or resumes the plan's thread with `thread/resume`, and durably binds the thread ID to the preallocated Host session ID. | Start the container with the role's mounts, including the plan's provider home. agentd starts `claude -p --input-format stream-json --output-format stream-json` with the profile's tool allowlist and only the Host MCP server, adding `--resume` with the plan's session when continuing; the adapter durably binds the provider session ID from the init event. |
| `Step` | Send one turn with the exact compiled input and output contract, route tool calls and events to `SessionIO`, and return the terminal turn and proposed result. | Send one user message, route tool calls and events to `SessionIO`, and return the `result` event and proposed result. |
| `Inspect` | Read only the journal-bound thread and exact turn through the pinned version's history methods. | Read the session transcript in the provider-home volume for the exact step. |
| `Stop` | Interrupt the turn where possible, then stop the container and report observed termination or uncertainty. | The same. |

Exact request shapes come from the pinned CLI version: the generated App Server schemas for Codex and the stream-JSON events for Claude Code. Provider request, thread, turn and session IDs are not interchangeable with Host identities. A Host tool call is bound through the session's channel; the provider's call identifiers are recorded as observations.

Operation propagation is guaranteed by the Host through every direct parent-child edge, independently of Agent output. A Manager acts only on delivered messages and can yield while awaiting children or a named dependency. The Host waits for the terminal observation and resolves active or uncertain activity before dispatching another writer. It later resumes the Manager with a recorded result or answer. No tool callback executes a child session recursively or holds a request open for the whole child workflow. Exact message contracts follow the [fixed process](manager-apply-concept.md#tasks-and-delegation); there is no Agent plan or child-operation rewrite.

Provider-native spawning, independent helpers and automatic review workflows are disabled in every profile. Every organizational Agent is created through Host scheduling and counted in its budget and history. Local coding tools remain available within the workspace. Their activity cannot become trusted project-check evidence without the `Checks` port; an Agent's report of a passing command is not a receipt. Markitect's reviewers use fresh ordinary role sessions and its own review contract; they do not inherit an implementer's conversation through a provider review shortcut.

Known Manager conversations may continue across explicit working-basis versions only through the confirmed step transition. Execution profile, mounts and authority ceiling stay fixed within one execution session; changes require a newly bound environment with proven conversation resume or a recorded handoff. An unprovable workspace or outstanding effect blocks progress rather than being hidden by session replacement. Cancellation does not establish successful termination. Unknown activity blocks affected handoff, even if a final message was received.

## Bind the runtime image before real work

An executor profile binds:

- the runtime image digest and the adapter kind;
- the CLI version inside the image;
- the provider's generated protocol schemas, where the provider has them;
- model and effort, effective provider settings and tool schemas;
- the credential reference.

The image records its CLI versions and schema digests in a metadata file when it is built. Preparation reads the image ID and that file through the engine without running the image: an image inspection plus copying the file out of a created, never started container. It never reads a host executable. Model selection remains configuration. Both demonstration Applies use the same frozen product build, image and profiles.

Earlier pins are candidates, not obligations. Those are `codex-cli 0.162.0`, which the deleted source required, and the desktop build `0.162.0-alpha.17.2`, which the offline metadata adapter recorded. Before real work, each adapter passes a bounded handshake smoke with the chosen image: start, one Host tool call, one file change observed by the Host, termination. The smoke repeats for every new image, and metadata never stands in for it.

The profile makes implicit context explicit: instructions, provider settings, environment policy and enabled tools and features. The clean container home removes ambient configuration. The profile still disables provider features that would act outside the Host, such as native spawning, memories and app connectors, and records the observed instruction sources. A build that cannot control or observe a required input is rejected instead of claiming a complete transcript.

## Adapter implementation boundary

`internal/adapters/container` is the concrete engine client. It handles images, volumes, containers, limits and the agentd framing on stdio. The session adapters, `checks` and the workspace implementation use it, and it knows no tasks, models or reports. Each session adapter starts as one cohesive package with `adapter.go`, `session.go`, `codec.go` and `recovery.go`, splitting only for a real responsibility; `codec.go` owns the provider's wire shapes. No adapter contains library rules, canonical Kind dispatch, review policy or old Project report branches.

Session protocols use typed Host tools for adjacent Manager messages, targeted reads, claims and registry proposals, questions, allowed checks and work or review submission. The Host itself guarantees operation propagation; an Agent tool does not decide participation. The adapter decodes provider messages and calls `SessionIO`; `run` validates meaning, authority and state. Prompt and context construction belong to `organization` and `run`, and the adapter adds only the provider envelope. There is no `agentexec.Invocation` compatibility envelope around the new records.

The old [client send path](https://github.com/Glacius-Labs/Markitect/blob/bc4b49e76f9b9e4b3ec98033a0b8c73d5013b310/src/internal/runtime/codexappserver/client.go) recorded outgoing requests after writing them. Replace that ordering: durably acknowledge the complete prepared bytes before writing, then record the observed send outcome or uncertainty. This covers tool responses and notifications, not only requests with IDs. A preparation record is not proof of delivery.

The old [tool handler](https://github.com/Glacius-Labs/Markitect/blob/bc4b49e76f9b9e4b3ec98033a0b8c73d5013b310/src/internal/runtime/codexappserver/tools.go) rejected duplicate call IDs. In the new design the Host owns durable deduplication and returns the existing reply for the same bound call and content. Different content under the same ID fails. The adapter must not consume an identity before the Host can reconcile a timed-out call.

Recovery keeps the existing rule: inspect the exact known turn and never automatically resend a lost dispatch. Missing handles or inconclusive history stay unknown. Neither a new container nor a new provider session repairs uncertainty about an earlier writer.

## Source replacement map

“Retain mechanisms” means transplant a coherent implementation and its relevant failure tests when it fits the new contract. It does not require retaining package paths, exported APIs, wire formats or old storage layouts. Rewriting a mechanism is preferable when extraction would introduce old domain dependencies. The [code map at that revision](https://github.com/Glacius-Labs/Markitect/blob/bc4b49e76f9b9e4b3ec98033a0b8c73d5013b310/src/internal/tooling/architecture/codemap.yaml) bounds the package inventory used below.

| Current source family | Target and disposition |
| --- | --- |
| `src/internal/core`, excluding `snapshot` | **Replace.** `internal/model` implements complete canonical v1, its values, diagnostics and canonical bytes. Remove the old Schema/envelope identities and compiler contract; do not wrap them in a second v1 frontend. |
| `src/internal/core/projectmodel`, including `projectvalue`, `projectbriefing`, `projectcoverage` | **Delete the old family.** Move relevant rules into independently designed `organization` and `realization`. Remove the fixed Manager/Statement/Artifact catalogue and shared Project DTO. Explicit context and accounting stay; their old representations do not. |
| `src/internal/core/snapshot` | **Retain the small fixed-tree idea and suitable comparisons** in `internal/snapshot`. Review identity/mode rules against the new source and candidate bindings; keep I/O out. |
| `src/internal/infrastructure/source` | **Retain suitable hardened acquisition mechanisms** in `adapters/gitrepo`. New `apply.Source` returns fixed new-model inputs, explicit dependencies and a complete census instead of loading the old Project. |
| `src/internal/infrastructure/guardedwrite` | **Retain suitable checked-write mechanisms** behind `apply.Publisher` in `adapters/gitrepo`. Add exact registry/tree binding, final read-back and inspectable partial/unknown publication. Preserve stale and path-substitution counterexamples. |
| `src/internal/infrastructure/recordstore` | **Replace as the journal backend** with `adapters/journal`, a SQLite store with one stream per repository. Retain suitable private-directory/identity/lease ideas and failure cases; verify the new driver's actual guarantees. Delete the closed four-record layout/codecs and old lifecycle coupling. |
| `src/internal/runtime/codexappserver` | **Rebuild** as `adapters/codexappserver` inside the container runtime. Candidate reuse: bounded framing, event observation, configuration fingerprinting, approval refusal and exact-turn recovery. Replace session orchestration, prompt/report codecs, outgoing recording order, duplicate handling, host process launch and old subject-alias machinery. |
| `src/internal/runtime/agentexec` | **Delete the old public execution protocol and runner.** Take only useful environment and fingerprinting mechanics and failure cases. Process lifetime moves into containers: `adapters/container` and `markitect-agentd` serve the session adapters and project checks; they know no tasks, models or reports and introduce no generic port catalogue. Host-side process trees, Windows job objects and short-name handling are not carried over. |
| `src/internal/runtime/projectexecutor` | **Delete.** Its transport router, helper bridge and separate native journal are replaced by bootstrap wiring, `run.Sessions`, Host tools and one authoritative run journal. |
| `src/internal/runtime/projectrun` | **Replace the runtime, not its outer DTO.** `run` owns tasks, scheduling, budgets, session messages, reviews, rejection accounting and accepted deliveries; `apply` owns aggregate reconciliation/publication; `realization` owns registry, candidates and composition rules. Preserve meaningful behavioral counterexamples. No broad Host callback struct or Plan/Report compatibility facade remains. |
| `src/internal/runtime/projectworkspace` and `projectrun` workspace bridges | **Retain useful private materialization/harvest/cleanup mechanisms** in `adapters/gitrepo`; replace contracts with immutable candidates, Host claims and full protected-input observation. Remove old runtime-to-workspace bridges after the new consumer is wired. |
| `src/internal/runtime/projectconfig`, `projectsetup` | **Replace.** Organization owns effective Manager/team policy and profile references; `run` owns budgets and execution policy; adapter setup owns typed provider configuration. Source acquisition captures all affecting bytes. Remove old manifest/schema setup and transport-specific configuration duplicated across layers. |
| `src/internal/runtime/exchangecli`, `src/cmd/markitect-exchange-executor` | **Delete**, together with exchange protocol fixtures and active documentation. Future external work uses a new demonstrated need and the current ports. |
| `src/internal/tooling/codexrunner`, `clauderunner` | **Delete** wrappers, private fixtures and their old protocol checks. The production Agent paths are the Codex App Server and Claude Code adapters in containers; controlled fake sessions remain test fixtures. |
| `src/internal/application/projectwork`, `projectcoverage` | **Replace** old acquisition/edit/render/coverage use cases with new model preparation, realization and adapter responsibilities. No adoption of old authoring edit semantics by Apply. |
| `src/internal/application/projectapp` | **Replace** shared application composition and operation wrappers with `internal/bootstrap` and concrete application services. Preserve the single composition principle; remove old callback wiring and obsolete operations. |
| `src/internal/application/projectcli`, `mcp`, `src/cmd/markitect` | **Replace command/tool schemas and handlers.** CLI first exposes implemented model preparation, Apply and run inspection; retain suitable CLI/MCP transport mechanics. MCP calls the same services when wired. Neither surface retains old operations or implements its own review loop. |
| `src/internal/application/projectadoption`, `projectexplore`, `projectonboarding`, `projecthistory` | **Delete old feature implementations and consumers.** Basic run inspection is rebuilt as a journal projection. Guided adoption, inference, external delivery and historical compatibility are outside the first Apply. Unknown file intake required by reconciliation is implemented in the new path. |
| `src/internal/modules/githooks`, `pipelines`, application module/architecture CLI packages and their dedicated commands | **Keep useful repository maintenance checks as tooling**, with explicit owners. Replace old schema/module composition, inventory and pin integration where needed. These checks do not become built-in product vocabulary or reasons to retain the old compiler. |
| `src/internal/tooling/architecture` | **Replace old four-layer/import rules and generated package map** with the selected responsibility graph and current package inventory. Keep negative import/missing-owner/platform-file cases. Delete obsolete restrictions explicitly; add no blanket exemptions. |
| `src/internal/application/buildinfo`, release CLI/install, `src/internal/tooling/release`, `publish`, `licenses`, `src/cmd/markitect-release`, `integration` | **Retain independent versioning, packaging, licensing and guarded-install behavior.** Update source paths, package contents, bootstrap/command smoke and input contracts. These maintenance tools stay outside Manager semantics and do not impose old model compatibility. |
| `src/internal/testkit`, `src/harness/examples`, `tools/smoke` | **Replace old project/report fixtures and end-to-end command assumptions.** Keep useful process/Git helpers and packaging counterexamples. New positive fixtures use v1 and the real new entry point. No second Manager engine is built in a test harness. |
| `experiments/mcp-pilot` and active old example/Playground scripts | **Retire old executable product assumptions.** The library Playground calls the new product and independently assesses it. Reuse capture/container utilities only where independent. Preserve dated observations at their original revisions. |

## Canonical sources documentation and gates

The source replacement also owns these non-Go changes. Omitting them would leave two product contracts even after deleting the Go code.

| Area | Required change |
| --- | --- |
| `.markitect/project.yaml`, old schema inputs and `.markitect/model` | Author Markitect's own canonical v1 model with explicit ownership/dependencies, and an initial registry under `.markitect/registry/` that covers every repository file. Until Apply exists, registry changes are reviewed edits to these files, validated by the new `check`; the first real Apply then takes over through its ordinary intake. Delete the old manifest and fixed-schema authoring contract. Keep operational configuration and registry outside Definition roots. Do not automatically translate obsolete Manager Definitions into v1 language requirements. |
| `.markitect/runtime.yaml` and current ignore/record configuration | Replace their schemas with the new execution/registry/storage policy where needed. No legacy decoder, ambiguous fallback or automatic historical run migration. Preserve old run evidence as fixed historical data, outside the new active store. |
| Generated project views, schemas and code map | Regenerate from their new canonical owners. Delete outputs whose owner/feature disappears. Do not hand-edit generated pages to hide a mismatch. |
| `AGENTS.md`, `CONTRIBUTING.md`, current architecture/usage/provider/operation guides and maps | Update to the implemented cut in the same reviewed change. Keep completed v1, the new design and dated historical evidence distinct. Removed paths referenced by historical records can point to their immutable source revision. |
| `.github/workflows`, `.markitect/modules`, hooks and command scripts | Remove deleted wrapper/old command gates, update paths and regenerate pins. Add meaningful new compiler, port, ownership and architecture checks. A green gate must exercise the actual replacement rather than silently skip old failures. |
| Active examples and packaged smoke | Replace old-schema examples with supported v1 fixtures; use the library for the Apply proof. Source packaging must contain the new executable path and required runtime assets. Historical examples need no compatibility implementation. |

## Deletions by delivery package

The old product is removed in one coherent cut, so that Main never holds half of the old runtime without its compiler or a replacement that is only partly wired.

| Delivery package | Replaces or deletes |
| --- | --- |
| APPLY-MODEL, APPLY-SNAPSHOT, APPLY-ORG, APPLY-REALIZATION | Nothing; the new pure packages grow beside the current source. They take over the rules of `core`, `projectmodel`, `projectbriefing` and `projectcoverage` in new form. Python probes never stand in for production conformance. |
| APPLY-CUTOVER | Switches bootstrap, CLI preparation, the self-model, generated views and source gates to the new foundation in one change. Deletes the whole old product under `src/`: the old compiler and `projectmodel` family, every `application/project*` package, `mcp`, the old runtime (`projectrun`, `projectexecutor`, `projectworkspace`, `projectconfig`, `projectsetup`, `agentexec`, `codexappserver`, `exchangecli`), the old infrastructure (`source`, `guardedwrite`, `recordstore`), `markitect-exchange-executor`, `codexrunner`, `clauderunner`, and their tests and fixtures. It replaces the old `testkit`, `harness/examples` and `tools/smoke` assumptions, rewrites `tooling/architecture` for the target graph, retires `experiments/mcp-pilot` and the old Playground scripts, and keeps the independent release, licensing and packaging tools. Apply is explicitly unavailable until INTEGRATE. |
| APPLY-RUN, APPLY-JOURNAL, APPLY-RUNTIME, APPLY-AGENT, APPLY-REPOSITORY | Build `run`, `adapters/journal`, the container runtime (`adapters/container`, `markitect-agentd` and the base image), `adapters/codexappserver` and `adapters/claudecode`, and `adapters/gitrepo` with `adapters/checks`. They transplant the mechanisms the source map names, with their failure tests, from the pre-cutover source at a recorded commit. |
| APPLY-INTEGRATE | Declares the `apply` ports first, then wires `apply`, CLI and MCP to the new services and runs the library cycle. |

After CUTOVER, Main builds, has one active model and command path, no old DTO imports and current architecture, coverage and packaging checks. The cut is prepared on its feature branch and integrated only when the new foundation and its replacement gates are real and passing.

## Deletion and completion criteria

A removal is complete when production/test imports, bootstrap registrations, command/tool schemas, configuration fields, canonical owners, generated views, active documentation, fixtures and CI pins have all been accounted for. Use reference searches and the package inventory to identify residual consumers. Retain a historical observation's original identity; never reclassify an old native smoke as proof of the new adapter.

The result is one compiler, one Host organization protocol, one run journal, one container runtime with one session port and two adapters, and one guarded publication route.
