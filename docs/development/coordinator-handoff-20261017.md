# Coordinator handoff: 17 October 2026

This is the owner's requested transition snapshot. All four delivery chats have acknowledged STOP and retain clean commits. Resume them only through an explicit takeover, or create fresh delivery chats and import the fixed commits below. Do not maintain this file as a backlog; subsequent state belongs in delivery chats and PRs.

## Start here

Read [the current delivery rules](../implementation-plan.md#integration), AGENTS.md, the concept register and relevant accepted Apply contracts on freshly fetched Main. The owner's 17 October rules replace older messages requiring a central merge, unconditional recomposition, Main-CI watches and artifact readbacks.

Create new scoped sessions with small task queues and one writer per shared file. Deliveries merge their own approved exact heads after required CI, using merge commits and `--match-head-commit`. Wait for CI in the background. Recompose only for conflicts or substantive overlap; react to a reported red Main. Use focused contract/error tests and existing evidence rather than repeating completed hardening.

The real Codex handshake with experimental `dynamicTools` is running in a separate owner-commissioned session. Its result is pending and will arrive at the previous Coordinator chat, `01a14472-9221-7a93-a474-38604c860ad9` (title: Architect). Read that result before fixing the RUN Session protocol. Do not start a duplicate experiment. This handoff grants no additional provider/Apply budget.

## Integrated baseline

Last fetched Main: `e369e9eefd913658731d758bd184960bf9776c19`, tree `c6f47e8daa547c0351860d836d3386564bcecf42`.

- PR316 completed the coherent root-layout v1 cutover. Legacy runtime/compiler paths are removed. Current CLI supports actual preparation/context and guarded model docs; Apply remains unavailable.
- PR317 integrated the first `internal/run` admission/budget/journal-contract slice. Its review findings were closed and required exact-head CI passed. Its own Main CI also passed before the owner changed the observation rules; this is historical evidence, not an instruction to repeat those reads.
- The canonical language, snapshot, organization, realization, repository capture and static profile metadata foundations are integrated. The Library M1/M2 oracle exists independently; neither real Apply has happened.

The primary checkout `C:\Users\Consiliari\Glacius Labs\Markitect` remains clean on the owner's `docs/manager-realization-north-star` at `3ebe3c6e6d283b998020350334e24a7303b089b7`. Preserve it. Use isolated branches from current Main. Unrelated draft PR264/265 and other worktrees remain outside this handoff.

## Delivery A: captured preparation to RUN basis

Status: ready for publication, no PR/push yet. No open independent review finding.

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

1. Import the complete `9be896e2` delivery into an owned branch; preserve the old worktrees. Open one focused PR, wait for exact-head required CI in the background, then self-merge.
2. Establish the smallest actual Apply Control consumer and source/store identity seam, grounded in captured inputs and the reviewed RUN reconstruction. Control is not currently implemented.
3. Connect it to the Journal adapter after that contract exists; retain the unavailable public Apply entrypoint until execution/publication is real.

## Delivery B: journal reconstruction and recovery barrier

Status: source and correction reviewed; shared registration prepared; not yet composed into one final candidate. No push/PR.

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
3. Continue Manager/reviewer/rework orchestration after the real dynamicTools result. Keep Session wire choices provisional until then.

## Other stopped chats and queued work

Titles below are the actual app titles; their latest roles differ from their original names.

| Chat | ID | Retained state / next queue |
| --- | --- | --- |
| APPLY-MODEL: Compiler v1 | `01a145d2-9ea4-7ce3-9e18-90c2ec41ab2d` | Delivery B; stopped clean at `a914323a`. |
| APPLY-SNAPSHOT: Dateibäume | `01a145d2-bceb-7603-9612-c2b51e01c127` | Delivery A; stopped clean at `2a0d3777`. |
| APPLY-BOOT: Architekturregeln | `01a145d2-818f-77c1-ba63-8cdae4c0af6a` | Cutover complete, clean `70c777e39f2301afb09de56bfcce0067c9e56c17` in `4346\Markitect`. Read-only Journal preparation complete. Next: actual RUN/Control contracts, dependency pin, narrow SQL adapter with real failure tests. |
| APPLY-PLAY: Bibliotheksprüfung | `01a145d2-e012-72d0-b8eb-d3b8985de753` | All watches closed, clean `5aaf236ba9d7d7cad4faaa7c003170c1daa57fc3` in `4b18\Markitect`. Next: preserve independent oracle; apply relevant process/path failure tests when those adapters exist; independently inspect actual A1/A2 when authorized. Do not restore CI-watch turns. |

No delivery chat owns an active process or CI job. The completed Root-local assembler gate also stopped. The previous Coordinator heartbeat `markitect-apply-weiter-orchestrieren`, targeting `01a14472-9221-7a93-a474-38604c860ad9`, is PAUSED for takeover. Do not reactivate its obsolete central-merge prompt.

## Journal and Process prerequisites

The real `apply.Control`, durable repository/store identity, epoch/lock lifecycle and physical Journal are absent. SQL must not introduce a second reducer or scheduler. Read-only driver research proposed `modernc.org/sqlite v1.60.1` with matching `libc v1.77.1`; nothing was adopted or installed. Verify the chosen pin/selected graph when adopting it, then test atomic append/CAS/exact retry, bounded pinned reads, blob integrity and actual control failures. Keep process-kill evidence distinct from power-loss durability.

Create a Process/adapter queue from [the five verified legacy commits](../implementation-plan.md#session-and-process-contracts): whole-tree termination with Job Objects and a child/grandchild timeout test; exact CLI-version mismatch; visible CLI rejection with a large prompt; Windows alias/short-name cases at the owning process/file boundaries. Use owned helper processes first. Do not revive old runtime packages or freeze the RUN Session protocol while the separate dynamicTools experiment is pending.

## Remaining product path

Complete assignment lifecycle and actual Control/Journal; use the handshake result to establish Session/process ports; implement one Package through independent review/rework, then recursion/integration/checks and guarded publication. Finally realize Library M1, inspect A1, change only the loan limit five to three, Apply M2 against A1 and inspect A2. No manual artifact repair between Applies. Real provider/Apply experiments, releases and deployments still require their separate finite authorization.

Suggested first message for the new Coordinator:

> Take over Markitect using this handoff and the current delivery plan. Fetch Main, preserve stopped sources, start small scoped delivery sessions with the queued tasks and direct self-merge ownership. First finish the two reviewed deliveries; assess the separately arriving real Codex dynamicTools handshake before fixing the RUN Session protocol. Follow the 17 October integration rules and do not restart the old Coordinator heartbeat.
