# Agent adapter selection and Apply rebuild plan

This plan owns the choice of the first Agent adapter, its session mapping and the map of what happens to the current source. The [Go architecture](manager-apply-go-architecture.md) and [port contracts](manager-apply-port-contracts.md) define the target; the [delivery plan](../../implementation-plan.md) owns the order of work. Architectural correctness and clear responsibilities take priority over saving implementation effort.

Source inspected: Main `bc4b49e76f9b9e4b3ec98033a0b8c73d5013b310`. Paths below name that source; target paths name the replacement.

## First adapter

Use **Codex App Server over local stdio**, called directly from Go. Implement `run.Sessions` in `internal/adapters/codexappserver`. Markitect owns the organization, task scheduler, context, authority, review policy and accepted results. Codex supplies the coding workflow inside an individual Agent session.

The official [App Server documentation](https://learn.chatgpt.com/docs/app-server) describes thread creation/resumption, turns, event streams, interruption, structured output and client-handled dynamic tools. Dynamic tools and their call protocol are experimental. The [integration guidance](https://developers.openai.com/blog/codex-as-a-platform) positions App Server for applications that need direct lifecycle and tool control. Those capabilities fit the proposed session port; choosing this boundary is a design judgment, not proof of compatibility with a particular installed build.

| Alternative | Treatment for the first Apply |
| --- | --- |
| Existing native App Server adapter | Select the transport and useful mechanisms; replace the adapter's old invocation, prompt and report design. |
| Existing `codex exec` / process proposal adapter | Remove this implementation path. Its current contract exchanges one invocation and final report rather than implementing the proposed Host session protocol. |
| Existing Claude wrapper | Remove the current proposal wrapper and its wiring. A future Claude session adapter must satisfy the same behavioral ports; this is not a conclusion about Claude's general capabilities. |
| Direct model API with a new coding loop | Leave as a possible later adapter. It would make Markitect responsible for another coding loop and its tools; that responsibility is not needed to define the organization. |
| External exchange packets | Remove their old implementation and CLI. Human/scripted packet exchange is outside the first real Apply. |

The choice rests on the session boundary and Host interaction, rather than the amount of existing Codex code. A new adapter must demonstrate the new contracts even where old tests pass.

## Session lifetime and protocol mapping

One Host session owns one App Server process, connection and provider thread. A process survives several known `Step` calls and can sit idle while its Manager waits. Fresh local reviews, integration reviews and final verification receive fresh sessions and threads. There is no shared process pool or provider-native Manager hierarchy in the first implementation. This spends more process resources in exchange for clear role configuration, lifetime and failure boundaries.

| Host port operation | Adapter mapping |
| --- | --- |
| `Open` | Start the selected executable, initialize the connection, create a fresh thread and durably bind provider handles to the preallocated Host session ID. |
| `Step` | Send one turn with the exact compiled input and output contract; route bounded tool calls/events to `SessionIO`; return the observed terminal turn and proposed result. |
| `Inspect` | Inspect only the journal-bound thread and exact dispatched turn, using the selected version's supported history/read methods. Inspection never starts replacement work. |
| `Stop` | Request interruption where applicable, terminate owned processes as needed and report observed termination or uncertainty. |

This mapping uses the documented [thread and turn lifecycle](https://learn.chatgpt.com/docs/app-server#lifecycle-overview); exact request shapes come from the chosen executable's schemas. Provider request IDs, thread IDs, turn IDs and provider session IDs are not interchangeable with Host identities.

Delegation is a short Host tool call returning a task ID. The Agent then ends its turn with a typed yield proposal. Host waits for the terminal observation and resolves active or uncertain tool/process activity before dispatching another writer. It later resumes the parent with a child result in a new step. A tool callback never executes a child session recursively or holds an RPC request open for the child's whole task.

Provider-native spawning, independent helpers and automatic review workflows are disabled in the first profile. Every organizational Agent is created through Host scheduling and counted in its budget/history. Local coding tools remain available within the assigned workspace. Their activity cannot become trusted project-check evidence without the `Checks` port. Markitect's reviewers use fresh ordinary role sessions and its own review contract; they do not inherit an implementer's conversation through a provider review shortcut.

Known Manager sessions may continue only with matching inputs, profile and an explicitly current candidate/workspace. A changed or unprovable binding requires a recorded new session, with supplied summaries rather than an invented continuation. Cancellation does not establish successful process termination. Unknown activity blocks affected handoff, even if a final message was received.

## Bind the executable before real work

Keep a single explicitly supported protocol profile for the first adapter. It binds the literal executable path and bytes, exact version, generated stable/experimental schema identities, selected model/effort, effective configuration and tool schemas. Model selection remains configuration; this design introduces no new default model. Both demonstration Applies use the same frozen product build and profile.

The current source accepts exactly `codex-cli 0.162.0` and embeds schema digests in [its protocol identity](../../../src/internal/runtime/codexappserver/process.go). A read-only `--version` probe of the locally found desktop executable returned `codex-cli 0.162.0-alpha.17.2`. These are different builds. Do not relax the old comparison, infer alpha compatibility or choose whichever executable PATH finds first.

Before the first real turn, select a build deliberately, generate its schemas, review the methods/fields actually used and freeze the resulting adapter profile. The old version is a candidate, not a compatibility obligation. Read/version/schema probes establish metadata only. A bounded real session must separately establish model access, tool exchange, input observation and lifecycle behavior. No real session was launched for this design.

The profile must make implicit context explicit: selected project instructions, provider configuration, environment policy and enabled tools/features. Use a dedicated operational location for provider state, avoid ambient user skills/plugins/memory changing a run, and record observed instruction sources. Reuse existing authenticated access without copying credentials into prompts, fixtures or public logs. Bind identity-relevant environment/configuration separately from secret values. If the selected build cannot control or observe a required input, reject that profile instead of claiming a complete transcript.

## Adapter implementation boundary

Start with a single cohesive package containing `adapter.go`, `process.go`, `rpc.go`, `session.go`, `tools.go`, `codec.go` and `recovery.go`, splitting files further only for a real responsibility. `rpc.go` owns framed bytes, correlation and bounded reads; `codec.go` owns provider wire shapes. It must not contain library rules, canonical Kind dispatch, review policy or old Project report branches.

Session protocols use typed Host tools for delegation, targeted reads, claims/registry proposals, questions, allowed checks and work/review submission. The adapter decodes provider JSON and calls `SessionIO`; `run` validates meaning, authority and state. Prompt/context construction belongs to `organization` and `run`, with the adapter adding only the provider envelope. There is no `agentexec.Invocation` compatibility envelope around the new records.

The current [client send path](../../../src/internal/runtime/codexappserver/client.go) observes successful outgoing requests after writing them. Replace that ordering: durably acknowledge the complete prepared bytes before writing, then record observed send outcome or uncertainty. Include tool responses and notifications, not only requests with IDs. A preparation record is not proof of delivery. The existing pre-turn recovery marker is useful material, but not a substitute for this contract.

The current [tool handler](../../../src/internal/runtime/codexappserver/tools.go) rejects duplicate call IDs. In the new design, Host owns durable deduplication and returns an existing reply for the same bound call/content. Different content under the same ID fails. The adapter must not consume an identity before the Host can reconcile a timed-out call.

Recovery retains the existing valuable rule: inspect the exact known turn and never automatically resend a lost dispatch. Replace old invocation-bound handles and whole-history assumptions with the selected protocol's supported reads. Missing handles or inconclusive history stay unknown. Neither a new process nor a new provider thread repairs uncertainty about an earlier writer.

## Source replacement map

“Retain mechanisms” means transplant a coherent implementation and its relevant failure tests when it fits the new contract. It does not require retaining package paths, exported APIs, wire formats or old storage layouts. Rewriting a mechanism is preferable when extraction would introduce old domain dependencies. The [current code map](../../../src/internal/tooling/architecture/codemap.yaml) bounds the package inventory used below.

| Current source family | Target and disposition |
| --- | --- |
| `src/internal/core`, excluding `snapshot` | **Replace.** `internal/model` implements complete canonical v1, its values, diagnostics and canonical bytes. Remove the old Schema/envelope identities and compiler contract; do not wrap them in a second v1 frontend. |
| `src/internal/core/projectmodel`, including `projectvalue`, `projectbriefing`, `projectcoverage` | **Delete the old family.** Move relevant rules into independently designed `organization` and `realization`. Remove the fixed Manager/Statement/Artifact catalogue and shared Project DTO. Explicit context and accounting stay; their old representations do not. |
| `src/internal/core/snapshot` | **Retain the small fixed-tree idea and suitable comparisons** in `internal/snapshot`. Review identity/mode rules against the new source and candidate bindings; keep I/O out. |
| `src/internal/infrastructure/source` | **Retain suitable hardened acquisition mechanisms** in `adapters/gitrepo`. New `apply.Source` returns fixed new-model inputs, explicit dependencies and a complete census instead of loading the old Project. |
| `src/internal/infrastructure/guardedwrite` | **Retain suitable checked-write mechanisms** behind `apply.Publisher` in `adapters/gitrepo`. Add exact registry/tree binding, final read-back and inspectable partial/unknown publication. Preserve stale and path-substitution counterexamples. |
| `src/internal/infrastructure/recordstore` | **Replace as the journal backend** with `adapters/journal`, a SQLite assignment-stream store. Retain suitable private-directory/identity/lease ideas and failure cases; verify the new driver's actual guarantees. Delete the closed four-record layout/codecs and old lifecycle coupling. |
| `src/internal/runtime/codexappserver` | **Rebuild** as `adapters/codexappserver`. Candidate reuse: bounded framing, event observation, literal process launch, configuration fingerprinting, approval refusal and exact-turn recovery. Replace session orchestration, prompt/report codecs, outgoing recording order, duplicate handling and old subject-alias machinery. |
| `src/internal/runtime/agentexec` | **Delete the old public execution protocol and runner.** Take only useful process/environment/fingerprinting mechanics and failure cases. A narrow concrete `adapters/process` package can serve the two actual consumers, Codex and project checks; it knows no tasks, models or reports and introduces no new generic port catalogue. |
| `src/internal/runtime/projectexecutor` | **Delete.** Its transport router, helper bridge and separate native journal are replaced by bootstrap wiring, `run.Sessions`, Host tools and one authoritative run journal. |
| `src/internal/runtime/projectrun` | **Replace the runtime, not its outer DTO.** `run` owns tasks, scheduling, budgets, session messages, review rounds and accepted deliveries; `apply` owns aggregate reconciliation/publication; `realization` owns registry, candidates and composition rules. Preserve meaningful behavioral counterexamples. No broad Host callback struct or Plan/Report compatibility facade remains. |
| `src/internal/runtime/projectworkspace` and `projectrun` workspace bridges | **Retain useful private materialization/harvest/cleanup mechanisms** in `adapters/gitrepo`; replace contracts with immutable candidates, Host claims and full protected-input observation. Remove old runtime-to-workspace bridges after the new consumer is wired. |
| `src/internal/runtime/projectconfig`, `projectsetup` | **Replace.** Organization owns effective Manager/team policy and profile references; `run` owns budgets and execution policy; adapter setup owns typed provider configuration. Source acquisition captures all affecting bytes. Remove old manifest/schema setup and transport-specific configuration duplicated across layers. |
| `src/internal/runtime/exchangecli`, `src/cmd/markitect-exchange-executor` | **Delete**, together with exchange protocol fixtures and active documentation. Future external work uses a new demonstrated need and the current ports. |
| `src/internal/tooling/codexrunner`, `clauderunner` | **Delete** wrappers, private fixtures and their old protocol checks. The first production Agent path is Go-to-App-Server; controlled fake sessions remain test fixtures. |
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
| `.markitect/project.yaml`, old schema inputs and `.markitect/model` | Author Markitect's own canonical v1 model with explicit ownership/dependencies. Delete the old manifest and fixed-schema authoring contract. Keep operational configuration and registry outside Definition roots. Do not automatically translate obsolete Manager Definitions into v1 language requirements. |
| `.markitect/runtime.yaml` and current ignore/record configuration | Replace their schemas with the new execution/registry/storage policy where needed. No legacy decoder, ambiguous fallback or automatic historical run migration. Preserve old run evidence as fixed historical data, outside the new active store. |
| Generated project views, schemas and code map | Regenerate from their new canonical owners. Delete outputs whose owner/feature disappears. Do not hand-edit generated pages to hide a mismatch. |
| `AGENTS.md`, `CONTRIBUTING.md`, current architecture/usage/provider/operation guides and maps | Update to the implemented cut in the same reviewed change. Keep completed v1, the new design and dated historical evidence distinct. Removed paths referenced by historical records can point to their immutable source revision. |
| `.github/workflows`, `.markitect/modules`, hooks and command scripts | Remove deleted wrapper/old command gates, update paths and regenerate pins. Add meaningful new compiler, port, ownership and architecture checks. A green gate must exercise the actual replacement rather than silently skip old failures. |
| Active examples and packaged smoke | Replace old-schema examples with supported v1 fixtures; use the library for the Apply proof. Source packaging must contain the new executable path and required runtime assets. Historical examples need no compatibility implementation. |

## Deletions by delivery package

| Delivery package | Replaces or deletes |
| --- | --- |
| APPLY-MODEL, APPLY-SNAPSHOT | Nothing yet; the new pure packages grow beside the current source. Python probes never stand in for production conformance. |
| APPLY-ORG, APPLY-REALIZATION | Nothing yet; they take over the rules of `projectmodel`, `projectbriefing` and `projectcoverage` in new form. |
| APPLY-CUTOVER | Switches bootstrap, CLI preparation, self-model, generated views and source gates together. Deletes `src/internal/core` apart from the reused snapshot ideas, the `projectmodel` family, `projectwork`, `projectcoverage`, `projectapp`, `projectadoption`, `projectexplore`, `projectonboarding`, `projecthistory`, `exchangecli`, `markitect-exchange-executor`, `codexrunner`, `clauderunner` and their tests and fixtures. Apply stays explicitly unavailable until INTEGRATE. |
| APPLY-RUN | Replaces `projectrun`, `projectexecutor`, `projectconfig` and `projectsetup` with `run` and its ports. |
| APPLY-JOURNAL | Replaces `recordstore` and the runtime store with the SQLite journal. |
| APPLY-AGENT | Rebuilds `codexappserver` as `adapters/codexappserver` and deletes `agentexec` after moving its process mechanics into `adapters/process`. |
| APPLY-REPOSITORY | Moves the useful parts of `infrastructure/source`, `guardedwrite` and `projectworkspace` into `adapters/gitrepo` and `adapters/checks`. |
| APPLY-INTEGRATE | Wires `apply`, CLI and MCP to the new services, removes the last old consumers and runs the library cycle. |

The cutover is prepared on its feature branch and integrated when the new foundation and replacement gates are real. Intermediate feature commits need not support the old product, but Main never holds deleted checks next to an unimplemented replacement. After CUTOVER, Main has one active model and command path, no old DTO imports and current architecture, coverage and packaging checks.

## Deletion and completion criteria

A removal is complete when production/test imports, bootstrap registrations, command/tool schemas, configuration fields, canonical owners, generated views, active documentation, fixtures and CI pins have all been accounted for. Use reference searches and the package inventory to identify residual consumers. Retain a historical observation's original identity; never reclassify an old native smoke as proof of the new adapter.

The result is one compiler, one Host organization protocol, one assignment journal, one session adapter and one guarded publication route.
