# V1 final review on F_final 4d23f7ae — 25 October 2026

Independent complete-diff and architecture review of the frozen final source for the technical V1 acceptance that the [owner directive](../concepts/sources/v1-acceptance-directive-20261025.md) asks for ([delivery plan](../implementation-plan.md#acceptance-and-evaluation-owner-directive-25-october), item "Review"). The reviewer wrote none of the reviewed code. This record is static review plus local source gates. It is not native provider, Apply, release or owner acceptance evidence.

## Verdict

**Acceptable as the V1 technical source state, with limits.** No blocking finding. Every concept guarantee checked below has one clear holder in the host, and no prompt text stands in for one. Four should-fix findings concern the canonical self-model, the delivery plan, a Cockpit leftover and the missing provider matrix; none changes runtime behavior. The native evidence on F_final is still open (see [Limits](#limits)).

## Binding

| Item | Value |
| --- | --- |
| F_final | `4d23f7aea0576ff5c477fd301b456a24bbb6dcb3` (Main after #722, #723, #724, #725, #726) |
| Diff base | `c9f7cb4560042e6755e3d94e292eba4939d29dd8`, the commit that recorded concept 1.0 |
| Reviewed scope | `internal/`, `cmd/`, `packaging/`, `examples/`, `.github/`, `.markitect/model/`, the product registry shards; 134 files, +8,237/−5,061 lines without tests. `tools/` (comparison study, oracles, Makerspace references) is outside the product and was not reviewed |
| Docs checked against F_final | [architecture](../architecture.md), [project workflow](../guides/project-workflow.md), [execution architecture](https://github.com/Glacius-Labs/Markitect/blob/e6d9b94a64c2083ee1487414a728a520a87def03/docs/design/v1-execution-architecture.md), [code map](../development/code-map.md), [delivery plan](../implementation-plan.md), package READMEs of Run, Apply, gitrepo and Codex |
| Toolchain | go1.27.1 windows/amd64 |

## Source gates on F_final

| Gate | Result |
| --- | --- |
| Hosted CI on exactly `4d23f7ae` ([run 40240295977](https://github.com/Glacius-Labs/Markitect/actions/runs/40240295977)) | Pass: Quality (ubuntu-24.04), Race detector (ubuntu-24.04), Quality (windows-latest) and Publish main source artifact all succeeded |
| `go build ./...`, `go vet ./...` (local, Windows) | Pass |
| `markitect check --repo . --revision 4d23f7ae` (local) | Pass; no stale link, no orphaned shard. A draft check of the reviewer's worktree flags only the reviewer's own untracked `.claude/settings.local.json`, as designed |
| `go test ./... -count=1` (local, Windows) | 31 packages pass, 11 have no tests. `internal/adapters/gitrepo` did not complete. It needs more than Go's default 10-minute timeout on Windows, and CONTRIBUTING prescribes `-timeout=60m`; with the default, its last running test hit the deadline it derives from the test binary, and the binary timed out. A rerun with `-timeout=60m` was stopped at the coordinator's request to free the machine for a native run, so the hosted CI is the evidence for this package |

## Guarantees and their holders

Each row names the one place that holds the guarantee on F_final and the controlled tests that exercise it.

| Guarantee | Concept | Holder | Controlled tests |
| --- | --- | --- | --- |
| No Manager on invalid or incomplete input | §5.3, V01 | `project.Prepare` (`internal/project/prepared.go:40`) in the Apply preflight; `apply.Command.Execute` checks the target and captures before any Run | gitrepo preflight and `TestCustomManifestPublication…StartsNoRun` |
| Model authority: model, execution, ignore and instruction inputs unchanged; registry is realization | §5.2, §6.3 | `Prepared.ValidateCandidateFor` (`prepared.go:120`), the one candidate check used at every checkpoint | `TestApplyRealizesADeletedDefinitionThroughTheShardOwner`, `…PackageMerge…` |
| Complete accounting; stale links and orphaned shards are work for Implement, never published | §6.1–6.3 | `realization/accounting.go`, `project.StaleLinks` (`prepared.go:203`) | `TestPublicationRefusesACandidateWithAStaleRegistryLink` |
| Complete hierarchy in every mandate | §7, V02 | `run.Execute` opens a copy and Session per Manager before work; `completeRoot` requires every other Manager reviewed; `ValidateClosed` requires one handoff per Manager | `TestCompleteHierarchyUsesRealGitAndClosedDurableStateForEveryMandate` |
| Questions only to parent or children; crossed waits | §8.2, D10, V03 | `execution.ask` and `await` (`internal/run/inbox.go`) | `TestQuestionsOutsideTheHierarchyAreRefused`, `TestAskFollowsTheHierarchyAndAwaitNeverWaitsForNothing` |
| Review of the actual base and fixed result, full diff, by commit | §6.4, §8.3, V04, V14 | `deliver` → `Copy.Review(base, result)`; `review` refuses a stale commit (`handoff.go`) | `TestLinuxChildReviewPreservesCompleteContextWithoutBinaryPatch` |
| Invalid Implement child reaches the parent and cannot be integrated | §8.4, D11, V05 | `fix` checks every submission; `review` refuses integration of an invalid one | `TestInvalidChildCandidateReachesParentAndIsCorrected` |
| Root completion checkpoint | §12.2, V05 | `judge` checks Root's state; `completeRoot` re-validates it on the quiet hierarchy | `TestRootCompletionRefusesAnEarlierJudgmentAfterItsStateChanged` |
| Judgment bound to assignment and assessed state; explicit state | §13, D12, V17 | `internal/run/assignment.go`, one struct with named transitions | the §13.3 counterexample tests in `run_collaboration_test.go` and `inbox_test.go` |
| Response end is no judgment; bounded idle stop | §13.2, §20 | `drive`, `next`, `acted` (`inbox.go`) | `TestRepeatedNoProgressEndsTheRunIncomplete`, `TestReadOnlyReviewIsNotStopped` |
| Fixed tool set, one implementation each, refusals with reason and next step | §14, D13 | `execution.tools` (`execute.go`); the Codex adapter only validates arguments and maps | `TestRefusalsNameReasonAndNextStep` |
| One closed-result validation | §10.3, D15 | `run.ValidateClosed` (`internal/run/validate.go`), used by Workflow and publication | `TestValidateClosed…` |
| Notes pages within the transport limit | §19.3, D17, V11 | `runstore.Store.Notes` and `Append` | ten runstore tests |
| No Claude path | §2.3, D18 | adapter, wiring, image recipe and metadata removed; sweep of the reviewed scope clean | – |
| Candidate retained and independently reopenable before closure | §17.4–17.5, V08 | `LocalGit.storeBundle`, `verifyBundle`, `Reopen` (`internal/adapters/gitrepo/local_run.go`) | `TestDurableCandidateRetentionAndReopenUnderLongPaths` |
| Lifecycle: bounded closure of every resource, terminal record saved and read back | §9.3, §18.2, §20.4, V09, V16 | `run.Execute` (`execute.go:69-276`) | `TestRunFailuresPersistHonestTerminalStateAndCloseRealGit` |
| Failure classes | §20.1 | `execute.go:154-162`: port and provider errors fail; deadline, budget, observation and no-progress stops are incomplete | – |
| Shared budget and deadline, no reset; fresh correction Runs | §20.2, §11.4, V06 | one `run.Allowance` pointer through `workflow.Apply` | `TestApplyVerifyRejectionStartsFreshCorrectionWithOriginalAllowanceAndFindings` |
| Late model blocker ends without publication | §11.4, V07 | `assignment.blocked`, `requiredDecision`; Workflow stops on any non-approval | `TestApplyStopsForModelBlockerCleanupUncertaintyAndMandatoryStorageFailure` |
| Publication of exactly the stored Verify approval, drift, partial transfer, readback, no commit | §21, V15 | `apply.Command.Execute` rereads and compares the stored record; `gitrepo.Publisher.Publish` | `TestApplyReopensStoredFinalVerifyAndRefusesSubstitutedAcceptance`, drift and cancellation tests |
| Deadlines reach Git and publication subprocesses | §20.3, §26.7 | context through `localCommand`, the Linux service and every publication guard | `TestPublicationGitQueriesCancelBeforeMutationAndAfterPartialTransfer` |

Changes since the step 4 source `e11e6cff` were reviewed with particular care:

- **Per-message clock (#725).** `MessageTimeout` now bounds provider work only; the clock pauses while Run serves a tool call (`inbox.go`, `messageClock`). Pausing, parallel tool calls, expiry racing a tool call and stop are handled under one mutex; expiry cancels with `DeadlineExceeded`, which stays incomplete. The Codex adapter passes the Exchange context into every tool call, so the pause takes effect. A parent's `await` is now bounded by the Run deadline and the allowance, not by the per-message limit; the architecture and project workflow say so.
- **ELF build outputs (#706, #724).** Only untracked ELF files that no registry shard in the copy's index names exactly stay out of a fixed state. Each Implement check lists them, the Run record keeps the final ones, and Verify receives them with its assignment. A binary registered by exact path is fixed whether or not it is staged; a staged file is captured as usual. The owner's P1 question, whether a required artifact can disappear silently, is answered no: an unregistered binary leaves the candidate but is always reported (see note 6 for the directory-entry boundary).
- **Await race (#726).** `nextStepLocked` now runs under the lock; no other `…Locked` helper is called outside it.

## Findings

Ranked blocking, should-fix, note. Line numbers are on F_final. A separate, independent agent re-checked should-fix 1–3 against the code (all confirmed) and searched Run, Workflow, Apply, publication and the snapshot code for a blocking defect. It found none and raised the boundary recorded as note 6.

Disposition by the coordinator: should-fix 1 is fixed by [PR #728](https://github.com/Glacius-Labs/Markitect/pull/728) (`3627c55f`), should-fix 2 by the coordinator's plan update after this record, should-fix 3 goes to the step 5 cleanup, and should-fix 4 to the technical acceptance report.

### Blocking

None.

### Should-fix

1. **The canonical self-model requires write enforcement that V1 deliberately does not have.** `.markitect/model/product/realization/artifact-accounting.yaml:14-15` requires mount-enforced write ownership in every Agent container ("refuse an ownership shape that mounts cannot enforce") and acceptance "after the File check of the submitter's stable area composed with the accepted stand"; line 13 names "the Orchestrator". These describe the retired assignment runtime. They contradict concept §6.4 and §15.6, which rule out per-write ACLs, and the self-model's own `product/runtime/container-runtime.yaml:8`. No production mount is read-only per area. *Failure scenario:* an Assessment or Verify of Markitect's own model, or a reader taking the canonical model as the architecture (concept §5.5), finds a binding Rule that the host does not and should not hold. The delivery plan's §5.5 deviation row names only the Package cut, not this.
2. **The delivery plan does not describe F_final.** `docs/implementation-plan.md:9` still names Main `334e7c31` as the current state, and `:13` says the evidence does not qualify the changed contract, with no link to the step 4 [requalification](v1-requalification-20261025.md) on `e11e6cff`. The technical-decisions list (`:30-40`) lacks the two contract-level decisions made since step 4: the per-message clock (#725) and the ELF build-output rule (#706, #724). The plan does not list which F_final changes have no native evidence yet: #696, #701, #705, #706 and #724, #725. *Failure scenario:* the owner reads the plan, which owns the current state and known deviations, and gets a wrong picture of what F_final holds and of what is natively proven.
3. **Cockpit leftovers in the state lock.** `internal/bootstrap/lock.go:38-41` justifies the 500 ms start retry by "a concurrent cockpit"; `stateHeld` (`:62`) is used only by a test; `stateLock.Check` (`:93`), documented as fencing a publication, has no caller. Cockpit is retired (concept §2.2). *Failure scenario:* a maintainer keeps code for a consumer that no longer exists and reads `Check` as an existing publication fence when publication is not wired to it.
4. **No provider matrix yet.** Concept §28.2 requires provider scope, runtime limits, repetition evidence and known limitations in a provider matrix before owner acceptance; step 5 of the plan names it. No document on F_final holds it, and the status of the generic AGENTS.md harness (§15.5) is recorded nowhere. *Failure scenario:* owner acceptance is asked for without the bounded support statement the concept makes its precondition.

### Notes

1. **Mandate text explains the host workflow.** `internal/harness/harness.go:144` ("An approved Assessment always proceeds through Implement"), `:149` ("Run aggregates the reviewed hierarchy's outcome and does not grant publication authority") and `:151` ("Realization Findings are returned to the host for a fresh Implement Run followed by a fresh Verify") are process explanation rather than role, rules, task and answer format (D13). The host enforces all of it independently; A1 reduced `Rules` but not `mandateDuties`. *Scenario:* a reader cannot tell which sentences are binding rules and which describe the host.
2. **Budget exhaustion before a correction Run is reported generically.** `run.validateRequest` (`internal/run/contracts.go:430`) refuses a Run whose allowance cannot cover one message per Manager with "invalid Run request, checked binding or finite limits", stores no record, and `workflow.Apply` (`internal/workflow/apply.go:37`) still appends the never-started Run to the report. The outcome is honestly not completed. *Scenario:* after several Verify rejections an operator sees a Run entry without a stored record or error and a message that does not say the allowance ran out.
3. **Test-only production symbols.** `run.UnavailableSessions` (`contracts.go:390`) is used only by a Codex adapter test, so the `ErrUnavailable` branch of the failure classification (`execute.go:157`) is unreachable in production. *Scenario:* dead surface in the Run contract that a reader takes for a supported mode.
4. **Self-model Package cut still open.** Project, Workflow and Harness are owned by `product/apply` and `product/adapters/agent`; the plan schedules the cut in step 5 (§28.1 leaves it to implementation). Known and recorded; listed for completeness.
5. **Workflow and Apply have no package tests of their own.** Their branches are exercised through the real-Git verticals in `internal/adapters/gitrepo`; acceptable, but a change to either package needs those suites.
6. **A directory registry entry claims no build output.** `unregistered` (`internal/adapters/gitrepo/local_run.go:487-541`) keeps an untracked ELF file out of the fixed state unless an exact-file entry names it, also below a directory the Manager registered; `RegisteredFiles` (`internal/project/registry_decode.go`) skips directory entries on purpose, so ordinary build outputs inside registered source directories do not become artifacts. The file is listed in `Check.BuildOutputs` and reaches Verify, but no check fails on it. *Scenario:* a model requires a binary fixture in a directory-registered area; Implement builds it without an exact entry, the candidate lacks it, and only Verify's reading of the reported build outputs can catch the omission. The boundary is documented in [WORKSPACES](https://github.com/Glacius-Labs/Markitect/blob/e6d9b94a64c2083ee1487414a728a520a87def03/internal/adapters/gitrepo/WORKSPACES.md).

## Owner standard

- **The canonical model determines the architecture.** Topology, contexts, ownership, accounting and the one candidate check all derive from the checked model; nothing in Run, Workflow or Apply maintains a second organization. The exception is the stale self-model Rule of should-fix 1.
- **Managers carry the domain responsibility.** The host plans, judges and repairs nothing: delegation, review, integration order, corrections and model blockers are Manager decisions; Run refuses only technically impossible or unbound actions, each with its reason and next step.
- **The host guarantees technical correctness without micromanaging.** Every V01–V17 guarantee above sits in one place in the host. No check receipts, evidence matrices or per-write approvals are demanded of agents. Complexity added since concept 1.0 (assignment state, message clock, build-output reporting) each answers an observed failure or an owner question, and none introduces a framework, DSL or extra agent role.

## Limits

- Static review and local source tests only. The concept §25.1 kinds "native component test", "public native Apply" and "independent oracle" are not covered by this record.
- No native evidence exists yet on F_final. Step 4 qualified `e11e6cff`; the refined idle stop (#696), stale links and orphaned shards (#701, #705), the ELF rule (#706, #724) and the per-message clock (#725) changed the contract afterwards. P2, the public Verify Reject → fresh Implement → fresh Verify → publication branch on F_final, is still to run.
- Static review cannot establish freedom from races; the race detector ran in Linux CI only.
