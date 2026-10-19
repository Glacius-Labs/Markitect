# Coordinator handoff: 17 October 2026

This is the owner's requested transition snapshot. All four delivery chats have acknowledged STOP and retain clean commits. Resume them only through an explicit takeover, or create fresh delivery chats and import the fixed commits below. Do not maintain this file as a backlog; subsequent state belongs in delivery chats and PRs.

## Start here

**Update after this snapshot (17 October 2026):** the real handshake finished and the owner adopted [DEC-037](../../concepts/register.md#dec-037-agent-sessions-and-checks-run-in-linux-containers). Agent sessions and checks run in Linux containers, and the Codex and Claude Code adapters receive the Host tools over MCP through `markitect-agentd`. The statements below about a pending `dynamicTools` result and a Process adapter with Windows Job Objects are superseded by the [session and process contracts](../../implementation-plan.md#session-and-process-contracts); the suggested first message at the end already reflects this. APPLY-RUNTIME is reserved for the owner's Claude Code session; do not dispatch another session to it.

Read [the current delivery rules](../../implementation-plan.md#integration), AGENTS.md, the concept register and relevant accepted Apply contracts on freshly fetched Main. The owner's 17 October rules replace older messages requiring a central merge, unconditional recomposition, Main-CI watches and artifact readbacks.

Create new scoped sessions with small task queues and one writer per shared file. Deliveries merge their own approved exact heads after required CI, using merge commits and `--match-head-commit`. Wait for CI in the background. Recompose only for conflicts or substantive overlap; react to a reported red Main. Use focused contract/error tests and existing evidence rather than repeating completed hardening.

The owner corrected the runtime handoff on 17 October: the handshake is complete. Codex worked through both experimental `dynamicTools` and MCP, and Claude Code worked through MCP. Clean isolation was observed only inside Linux containers; direct Windows runs inherited user configuration, including global MCP servers. These bounded probes establish the chosen direction, not product reliability or a completed adapter.

The accepted owner decision DEC-037 is being published in [PR319](https://github.com/Glacius-Labs/Markitect/pull/319). After it merges, read DEC-037 in the [concept register](../../concepts/register.md) and the section "Container runtime" in the [rebuild plan](../apply-design-before-dec-056/manager-apply-rebuild-plan.md). Each Agent session and each check runs in its own Linux container from a digest-pinned image. Mounts supply authority; workspaces live on Linux volumes. Host tools reach both providers through MCP, carried with provider/control traffic over the single stdio connection of `markitect-agentd`. The two Session adapters are Codex App Server and Claude Code.

This replaces the pending-handshake and Windows Process-adapter instructions in earlier handoffs and messages. Do not build `adapters/process`, Job Object handling or Windows short-name guards for the Session path. APPLY-RUNTIME is already assigned to the owner's Claude session; do not dispatch another writer. APPLY-AGENT waits for both RUN's Session port and RUNTIME. The RUN core continues unchanged. No duplicate provider experiment or additional provider/Apply budget is authorized.

## Integrated baseline

Last fetched Main: `20dc980f2aaafd99600bc4b78b75cc3420ba2bd1` (PR318). The source-delivery baseline below remains `e369e9eefd913658731d758bd184960bf9776c19`, tree `c6f47e8daa547c0351860d836d3386564bcecf42`.

- PR316 completed the coherent root-layout v1 cutover. Legacy runtime/compiler paths are removed. Current CLI supports actual preparation/context and guarded model docs; Apply remains unavailable.
- PR317 integrated the first `internal/run` admission/budget/journal-contract slice. Its review findings were closed and required exact-head CI passed. Its own Main CI also passed before the owner changed the observation rules; this is historical evidence, not an instruction to repeat those reads.
- PR318 integrated the simplified delivery rules and original handoff. Its pending-handshake/Windows Process instructions are superseded by the owner's DEC-037 correction above. PR319's owner was notified to remove the obsolete "Session and process contracts" section when composing with Main. PR319 was still open when this correction was prepared; its architectural decision is accepted, but its docs and runtime implementation are not claimed integrated.
- The canonical language, snapshot, organization, realization, repository capture and static profile metadata foundations are integrated. The Library M1/M2 oracle exists independently; neither real Apply has happened.

The primary checkout `C:\Users\Consiliari\Glacius Labs\Markitect` remains clean on the owner's `docs/manager-realization-north-star` at `3ebe3c6e6d283b998020350334e24a7303b089b7`. Preserve it. Use isolated branches from current Main. Unrelated draft PR264/265 and other worktrees remain outside this handoff.

## Delivery A: captured preparation to RUN basis

Status: ready for a PR; producer and complete integration branches are pushed at the exact heads below. No open independent review finding. No source PR or hosted CI yet.

| Item | Fixed value |
| --- | --- |
| Producer source | `2a0d37771194fa1f53bed451c2310855093871d2` |
| Producer branch/worktree | `dev/apply-captured-run-binding`; `C:\Users\Consiliari\.codex\worktrees\ac74\Markitect` |
| Complete source plus reviewed shared integration | `9be896e2cd9beeb807b52695d3f24deda725b6a6` |
| Complete tree | `999ac42aaa25a191d83fe21bc21b8d817eaee34f` |
| Complete branch/worktree | `dev/apply-captured-run-binding-publication`; `C:\Users\Consiliari\.codex\worktrees\apply-rebuild-baseline\Markitect` |

Three new `internal/apply/run_binding*.go` files provide `CapturedSettingsResult.BindRun(RunBindingFacts)`. They use only the final successful capture, preserve original inputs/provenance, require explicit owner payload/allocation/repository/build assertions and call `run.Bind` without acquisition, run creation or journal effects. Preview has no binding entrypoint. Host assertions do not establish physical Control or runtime readiness.

The complete commit adds exact registry entries, the actual Apply-to-Run import edge and matching guidance. All three producer files and five shared files received independent complete review. Root-local full gates and source smoke finished on exact `9be896e2`: 1,961 tests passed, 25 existing platform skips, no failures. No hosted CI has run for it.

Evidence: `%TEMP%\markitect-captured-binding-9be896e2\` (complete gates and smoke), and `%TEMP%\markitect-apply-run-binding-e369-gates\final-manifest.json` (producer evidence including the earlier honest missing-registration/import failures).

Queue for its next owner:

1. Fetch the complete `9be896e2` delivery from `origin/dev/apply-captured-run-binding-publication` into an owned branch; preserve the old worktrees. Open one focused PR, wait for exact-head required CI in the background, then self-merge.
2. Establish the smallest actual Apply Control consumer and source/store identity seam, grounded in captured inputs and the reviewed RUN reconstruction. Control is not currently implemented.
3. Connect it to the Journal adapter after that contract exists; retain the unavailable public Apply entrypoint until execution/publication is real.

## Delivery B: journal reconstruction and recovery barrier

Status: source and correction reviewed; shared registration prepared; not yet composed into one final candidate. Both branches below are pushed at the exact listed heads. No source PR or hosted CI yet.

| Item | Fixed value |
| --- | --- |
| Final corrected producer source | `a914323a9c527b64bb12fd0596629b61d8b19bc8` |
| Producer tree | `660ae656ac9275187bba2ecabf486fbbb3ba41b6` |
| Producer branch/worktree | `dev/apply-run-admission`; `C:\Users\Consiliari\.codex\worktrees\1f8c\Markitect` |
| Reviewed shared registration/docs | `6f61eb971c5f20380932113bbc3adaf550b74ee0` |
| Shared branch/worktree | `dev/apply-run-rehydration-publication`; `C:\Users\Consiliari\.codex\worktrees\apply-foundation-packaging\Markitect` |

Important: `6f61eb97` contains the earlier uncorrected source `ee366ba491cb5d3b6c78d32948f0f34a5cc6b602`. Combine it with `a914323a`; do not publish `6f61eb97` alone.

`Restore` reconstructs from bounded, revision-pinned whole batches and verified basis/owner blobs, through the same reducer, without effects or current-compiler reinterpretation. Open holds require explicit reconciliation behind `RecoveryRequired`. Pure `BindingRefs`/`BindingOwnerRef` APIs let the store verify required blobs. The revision range is explicitly 0 through 2^63-1.

A targeted review found that newly hashed malformed raw-model closures could pass the historical decoder. The correction shares intrinsic Origin/manifest/count/Local/original-digest checks with Bind. Fourteen distinguishing cases failed before the fix and pass after it. The full original nine-file review, three-file correction review and three-file shared review have no open findings. Final producer test/vet/local Linux race passed; all 11 correction proof hashes were independently checked. These are local proofs, not hosted CI or physical durability.

Exactly three new files need registration: `internal/run/restore.go`, `restore_binding.go`, `restore_test.go`; the prepared shared commit includes them. All worktrees are clean. Evidence: `%TEMP%\markitect-run-1r-raw-closure-1f8c\handoff-proof.json`; earlier proofs and failures remain under `%TEMP%\markitect-run-1r-1f8c-df58afd8\`.

Queue for its next owner:

1. Combine corrected source and reviewed shared commit. Check actual overlap with Delivery A/current Main, run relevant composition checks, obtain required exact-head CI and self-merge.
2. Implement a small assignment lifecycle slice: explicit bounded cap extensions without resetting consumption/holds, closure only after quiescence, immutable closed assignments and another ordinary assignment in the same repository stream. A new assignment gets its own budget; previous ledgers remain history.
3. Establish the provider-neutral Session port in RUN using DEC-037, then continue Manager/reviewer/rework orchestration. The handshake is complete. APPLY-AGENT implements its two adapters only after that port and the assigned RUNTIME foundation exist.

## Other stopped chats and queued work

Titles below are the actual app titles; their latest roles differ from their original names.

| Chat | ID | Retained state / next queue |
| --- | --- | --- |
| APPLY-MODEL: Compiler v1 | `01a145d2-9ea4-7ce3-9e18-90c2ec41ab2d` | Delivery B; stopped clean at `a914323a`. |
| APPLY-SNAPSHOT: Dateibäume | `01a145d2-bceb-7603-9612-c2b51e01c127` | Delivery A; stopped clean at `2a0d3777`. |
| APPLY-BOOT: Architekturregeln | `01a145d2-818f-77c1-ba63-8cdae4c0af6a` | Cutover complete, clean `70c777e39f2301afb09de56bfcce0067c9e56c17` in `4346\Markitect`. Read-only Journal preparation complete. Next: actual RUN/Control contracts, dependency pin, narrow SQL adapter with real failure tests. |
| APPLY-PLAY: Bibliotheksprüfung | `01a145d2-e012-72d0-b8eb-d3b8985de753` | All watches closed, clean `5aaf236ba9d7d7cad4faaa7c003170c1daa57fc3` in `4b18\Markitect`. Next: preserve independent oracle; independently review container/adapter evidence when delivered; independently inspect actual A1/A2 when authorized. No Windows Process-adapter queue and no CI-watch turns. |

All code/document handoff heads are pushed and their remote refs were verified. The two stopped chats above retain remote branches `dev/apply-product-cutover` and `fix/playground-candidate-git-lifecycle`. Detailed test logs and proof manifests remain local at the paths listed here; they are not Git contents. The [remote securing update](https://github.com/Glacius-Labs/Markitect/pull/318#issuecomment-6231511146) records all seven fixed refs.

None of the four stopped delivery chats owns an active process or CI job. This does not describe the separately assigned APPLY-RUNTIME session. The completed Root-local assembler gate also stopped. The previous Coordinator heartbeat `markitect-apply-weiter-orchestrieren`, targeting `01a14472-9221-7a93-a474-38604c860ad9`, is PAUSED for takeover. Do not reactivate its obsolete central-merge prompt.

## Reserved runtime work and dependencies

| Zone | State and scope | Dependency |
| --- | --- | --- |
| APPLY-RUNTIME | Already occupied by the owner's Claude session: `internal/adapters/container/`, `internal/agentd/`, `cmd/markitect-agentd/` and the base runtime image. Do not start another session or write these paths. | May proceed alongside RUN below the Session port; no task/model/review semantics. |
| APPLY-AGENT | Not ready for dispatch: `internal/adapters/codexappserver/` and `internal/adapters/claudecode/`. No `internal/adapters/process/`. | Both RUN's Session port and RUNTIME must exist first. |

These are reservations and prerequisites, not evidence that either adapter or the container runtime is implemented. Existing RUN admission, reconstruction, budget and assignment work continues. Do not turn the runtime decision into a RUN rewrite.

## Journal and container prerequisites

The real `apply.Control`, durable repository/store identity, epoch/lock lifecycle and physical Journal are absent. SQL must not introduce a second reducer or scheduler. Read-only driver research proposed `modernc.org/sqlite v1.60.1` with matching `libc v1.77.1`; nothing was adopted or installed. Verify the chosen pin/selected graph when adopting it, then test atomic append/CAS/exact retry, bounded pinned reads, blob integrity and actual control failures. Keep process-kill evidence distinct from power-loss durability.

The reserved RUNTIME owner supplies the pinned container image, Linux volumes and the `markitect-agentd` MCP/stdio bridge. Session/check authority comes from mounts, and ending a container ends its process tree. Image-recorded CLI/toolchain metadata replaces selection of a Windows desktop executable. Keep failure tests at these actual boundaries; do not transplant the obsolete Windows Process adapter or its Job Object/short-name queue. Real adapter smoke and provider execution still need their separate finite authority; the completed throwaway handshake does not grant another experiment.

## Remaining product path

Complete assignment lifecycle and actual Control/Journal; establish RUN's Session port while the already assigned RUNTIME proceeds. Start APPLY-AGENT only when both dependencies exist. Implement one Package through independent review/rework, then recursion/integration/checks and guarded publication. Finally realize Library M1, inspect A1, change only the loan limit five to three, Apply M2 against A1 and inspect A2. No manual artifact repair between Applies. Real provider/Apply experiments, releases and deployments still require their separate finite authorization.

Suggested first message for the new Coordinator:

> Take over Markitect using this handoff and the current delivery plan. Fetch Main and read DEC-037 and "Container runtime" in the rebuild plan once PR319 merges. Preserve stopped sources and start small scoped delivery sessions with direct self-merge ownership. First finish the two reviewed deliveries. The handshake is complete; RUN continues unchanged. APPLY-RUNTIME is already occupied by the owner's Claude session: do not dispatch a duplicate. APPLY-AGENT waits for RUN's Session port and RUNTIME. Do not build a Windows Process adapter or restart the old Coordinator heartbeat.
