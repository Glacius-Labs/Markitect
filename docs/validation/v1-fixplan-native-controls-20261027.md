# Fix plan v2 native controls on F_fix 714ae136 and F_fix2 0075171c — 26 and 27 October 2026

This record covers the bounded native controls of the [fix plan v2](../implementation-plan.md#product-fixes-before-the-next-study-owner-approval-26-october), approved by the [owner](../concepts/sources/v1-fix-plan-approval-20261026.md). They ran outside any study, after Phase C:
- the four planned controls A, B1, B2 and C on F_fix, the merge of S1–S3;
- one added control, B2-fix, on F_fix2. That is F_fix plus only the fix for the await deadlock that B2 found.

The coordinator dispatched each under the [standing evidence authorization](../concepts/sources/v1-source-implementation-approval-20261023.md#subsequent-standing-evidence-authorization), with one attempt each. The record is descriptive: what each control asked, what it showed, what it cost, the census and the limits.

## Result

| Control | Tests | Question | Result |
| --- | --- | --- | --- |
| A, thread probe | WP1 (S1's open question) | Does a Codex thread accept a new turn after a turn a provider error ended? | **Yes.** The same thread accepted and completed a new turn, with its earlier context, after a turn that a non-retrying provider error ended. Documented proxy: not `serverOverloaded` |
| B1, fresh W0 | WP2, WP3a, WP3c | Does a fresh Makerspace build complete under the idle limit and the new message accounting? | **Yes.** Passed and published, with 92 of 256 messages. Root's longest provider-idle stretch was 41.5 s. WP3c's check was not reached (no correction pair) |
| B2, fresh C4 | WP3a, the WP3b deferral; WP2 | Does question traffic still exhaust the allowance in the fresh C4 situation? | **Not in this run.** 84 asks in Implement cost only 40 messages, and 195 of 256 remained. It still ended without a candidate: an undetected **mutual await deadlock** between Root and `policies` held until the Apply deadline ([B2](#b2-fresh-c4)) |
| C, unchanged re-application | WP4 (S3, DEC-073) | Does the host's "already verified" fact reach the Runs, with no change or only rejected ones? | **The fact held** in every Run. The predeclared condition was **not met**: Implement made a Definition-bound correction of a gap in B1's candidate, and Verify approved it with no Findings. Under concept §11.3 such a change is a Reject unless a defect is found, so this is a contract gap ([C](#c-unchanged-re-application)) |
| B2-fix, fresh C4 on F_fix2 | the await fix (#759); WP3a, WP3c, WP2 | Does the fresh C4 now complete without the mutual-await stall, and how does the question traffic look? | **Yes.** Passed and published after one correction pair: Verify rejected with three Definition-bound Findings, Implement corrected, Verify approved. 124 of 256 messages. The Verify-reject path (P2) was **exercised**, with every check true ([B2-fix](#b2-fix-fresh-c4-on-f_fix2)) |

No control left residue or a credential hit, and the runtime login file was unchanged. All Apply attempts are listed under [Controls](#controls).

**Per WP:**
- **WP1:** a thread accepts a new turn after an error-terminated one (A, proxy).
- **WP2:** no turn came near the idle limit. The longest provider-idle stretch was 51 s (B2-fix), and busy Root turns ran up to 514 s of provider time (B1).
- **WP3a:** asks no longer exhaust the allowance (B1, B2, B2-fix).
- **WP3c:** a correction pair started with enough messages and completed (B2-fix).
- **WP4:** the fact holds, but the approval of a change under it is not checked by the host (C).
- **WP5:** closed answers for questions to Managers that had handed off (B1).
- **The await fix:** a fresh C4 completes (B2-fix).

## Binding

| Item | Value |
| --- | --- |
| Source F_fix | `714ae136c738dedb7bf4b881d1ade689c001913f`, tree `9e578297120311d4c27c36fd4ff31921a23475b2`: the merge of [#745](https://github.com/Glacius-Labs/Markitect/pull/745) (S3), with parents `0fd813cc` and `25c47342`. It contains [#733](https://github.com/Glacius-Labs/Markitect/pull/733) (S1) and [#734](https://github.com/Glacius-Labs/Markitect/pull/734) (S2). Fresh clean clone; head, tree and clean state checked before the build |
| Product binary | Built from F_fix with `-trimpath` and go1.27.1 at 21:31:25–21:32:15Z (50 s). SHA-256 `b4f8e16f9d714923875ff28c0938967b8a671f1f8a07f9160512f48af0fa13e1`, productBuild `0.14.1+714ae136…`, `vcs.modified=false` |
| Images | Built from F_fix's `packaging/runtime`. Base `sha256:5a036e1d427476e21648bbd67171318f2735ddd30f9e4abb81d0cbc3c0a18968`; Go `sha256:d8d6f826a024d5c4afa5cb7624648191597ccd1cc4087eb0c17bad77e07cf1a3`. Both `runtime.env` records hold `codex-cli 0.162.0`, `codexSchema` `816cef0e…` and `agentd` F_fix |
| Source F_fix2 (B2-fix only) | `0075171c4d1a1e2e9985ac152ee843b4eef158eb`, tree `382c939767df13f6b533a356fc5a4d2dd1e4b92b`, on branch `fix/ffix2-await-on-714ae136`. That is F_fix plus #759's two commits (`71fab36f`, `0075171c`), which touch `internal/run` only; Linux CI 40423066407 was green. Main was not used, because after G1 (Core 1.2.0) it can no longer check the Phase C material. The Phase C material and `testdata/core` trees equal F_fix's |
| Binary and images F_fix2 | Built at 00:20:01–00:20:47Z (46 s). Binary SHA-256 `35e64dc1adc0d0d223a5de18ea63cf160e10c6312ac8a6d9b6ef61c0544feaf7`, productBuild `0.14.1+0075171c…`, `vcs.modified=false`. Base `sha256:a3242173557eb7440d3f204ba8e3b2d7e2521446f9cff164932b84fe2062aced`; Go `sha256:0aab3b63300219d94555755d91747a2c9739423fc452311813d181af88973e40`, with `codex-cli 0.162.0` and `agentd` F_fix2 |
| Provider | Codex 0.162.0, `gpt-6-sol`, effort `high`, normal account login. The Applies used the dedicated runtime login, which was only hashed, before and after each Apply. Control A used the host Codex's default login, also only hashed |
| Run configuration | F_fix's example, format 4: `providerIdleTimeoutSeconds` 600, `maxMessages` raised to 256, Apply deadline 5,400 s, 32 Managers |
| Material | From the F_fix clone: the Phase C snapshots and execution files (model tree `7dee554d…`, execution tree `0307c790…`, identical to Phase C's freeze) and `testdata/core`. The pinned manifests and `w0.yaml` were re-verified for every input |
| Controller | `controller-fixplan`, a copy of the frozen Phase C controller. It adds control scenarios `XC4` (a fresh C4 with fresh state) and `XNC` (B1's published candidate re-applied unchanged on B1's state), keeps `providerRetries` and each Run's `basis` and `origin`, and records control C's verified fact with the trees it compares. 60 offline self-tests pass. One independent review and its delta reviews returned READY before any run |
| Gates before each launch | `NATIVE-RUN.lock`, Docker readiness on the pinned image, and a provider quota gate (closed at or above 85 %; [quota](#quota)). The host was quiet: a blocking check up to B2, then record-only ([machine rule](#deviations-and-rule-changes)) |

## Controls

Seconds are Apply process time. Messages are the total and the split per Run.

| Control | Apply | Status | s | Messages | Paths | Final commit / tree | Tokens | `providerRetries` | P2 | `package-result.json` |
| --- | --- | --- | ---: | --- | ---: | --- | ---: | ---: | --- | --- |
| A | — (probe) | accepted-and-completed-with-context | 16 (wall) | — | — | — | 94,059 | — | — | `result.json` `e8317bdc9357` |
| B1 (`m-b1-w0`) | `86f3f73b17500a7b88cd0427b266e3a6` | technical pass, passed, scenario met | 1,750 | 92 (21/50/21) | 66 | `754e8eef` / `30942ae5` | 82,482,115 | 0 | not exercised | `87245b8ba329` |
| B2 (`m-b2-c4`) | `f6e0731d4f3af37dd42765754348ed56` | technical fail (product), not completed | 5,414 | 61 (21/40) | 0 | — | 48,188,260 | 0 | technical | `d82597dbae1b` |
| C (`m-c-xnc`) | `f232238f94bc82977088bf04aba51cd2` | technical pass, passed; scenario not met | 1,353 | 89 (21/21/47) | 3 | `aa12b745` / `890c4c1d` | 42,473,683 | 0 | not exercised | `7929885d5e55` |
| B2-fix (`m-b2fix-c4`, F_fix2) | `3f3559cd5883170a4e90267b57f9bd4e` | technical pass, passed, scenario met | 2,830 | 124 (21/38/21/23/21) | 82 | `d4bd27cb` / `6ad45861` | 110,665,471 | 0 | **exercised** | `455d032b0da6` |

The four Applies used 11,346 s and 283,809,529 tokens. No transient provider error occurred, so S1's repeat did not run.

### A: thread probe

- **Setup:** a small client of `codex app-server` 0.162.0 with no Markitect. It would have removed any Codex or OpenAI environment variable, but none was set. It ran with the host Codex's home and configuration: its stderr shows OAuth errors of a configured MCP server and one failed `exec_command`, neither in a turn the probe judged.
- **Turn 1:** with the normal model, it planted a token.
- **Turn 2:** a per-turn override to a model the account cannot use. The turn **started** and then ended `failed` with the provider error `400 model_not_found` ("not supported when using Codex with a ChatGPT account"), `codexErrorInfo: "other"`, `willRetry: false`. The error arrived about 0.6 s after `turn/start`, before any model work, so the proxy is far from an overload in the middle of a turn.
- **Turn 3, same thread, normal model:** `turn/start` was accepted, the turn completed, and it returned the planted token.
- **Contrast, a new thread:** completed, and did not know the token.
- **Reading for S1's open question** (may a response continue in its thread after a provider error ended a turn that had already served actions; S1 now ends the response then): after a turn that a non-retrying provider error ended, a Codex 0.162.0 thread accepts and completes a new turn with its context. The probe does not show that continuing after served actions is safe; that stays an open follow-up.

### B1: fresh W0

Assessment approved, Implement submitted, Verify approved, with no correction branch.

| Run | Messages charged | Left | Asks / replies (all Managers) | Refused calls | Root: wall = host-served + provider; longest idle |
| --- | ---: | ---: | --- | --- | --- |
| Assessment | 21 | 235 | 0 / 0 | 1 `await` | 105.7 = 52.4 + 53.3; 7.2 s |
| Implement | 50 | 185 | 101 / 91 | 11 `ask`, 3 `reply`, 5 `await`, 1 `delegate` | 929.3 = 415.2 + 514.0; 41.5 s |
| Verify | 21 | 164 | 0 / 0 | — | 157.6 = 91.0 + 66.5; 32.7 s |

- **WP2:** Root's Implement turn worked 514 s of provider time. Under Phase C's per-message clock that was close to the 600 s limit. Under the idle limit, the longest stretch without a served call was 41.5 s, and the turn completed.
- **WP3a:** 101 asks cost no allowance of their own; Implement was charged 50 messages. The refused asks are consistent with the cap of 8 open questions. In Phase C the fresh W0s used 152–170 messages.
- **WP3c:** not exercised, since Verify approved the first candidate.
- **WP5:** exercised. Questions to Managers that had already handed off were closed, and the askers received the closed answer ("… has handed off without answering"), for example q8 and q35. No Run stopped on a re-delivered question.
- Root's Implement had a second, closing turn of 5.1 s (provider), which the table leaves out.

### B2: fresh C4

| Run | Messages charged | Left | Asks / replies | Refused calls | Root: wall = host-served + provider; longest idle |
| --- | ---: | ---: | --- | --- | --- |
| Assessment | 21 | 235 | 0 / 0 | — | 95.8 = 57.9 + 38.0; 7.3 s |
| Implement | 40 | 195 | 84 / 75 | 5 `ask`, 10 `await` (8 refusals, and the 2 blocked calls the deadline cancelled) | 5,081.7 = 4,944.0 + 137.6; 39.6 s (open when the Apply ended) |

- **WP3a and the WP3b deferral:** in the same fresh C4 situation, M-c3-C4 exhausted all 256 messages on 231 asks and 203 replies. Here, 84 asks and 75 replies cost 40 messages, with 195 left.
  - The Run never finished, so 40 is a partial count.
  - Asks also fell from 231 to 84, and there was one attempt each, so the accounting change is not isolated as the only cause.
  - Whether the existing sequencing suffices once a fresh C4 completes is not shown here; see [B2-fix](#b2-fix-fresh-c4-on-f_fix2).
- **Verdict:** not completed; product defect, an undetected mutual `await`. WP3a and WP2 behaved as observed above.
- **The deadlock, a timeline from the transcripts:**
  - `policies` asked Root three questions that Root never answered: q74 at 22:31:08Z, q76 at 22:32:43Z and q77, about the money interface, at 22:33:01Z. They were delivered to Root at 22:33:14–22:33:17Z. Root kept collecting other children's handoffs.
  - From 22:34:13Z Root waited in `await` for `policies`' handoff.
  - From 22:34:33Z `policies` waited in `await` for Root's answers ("Root answer pending").
  - Every other Implement session had finished by 22:34:22Z.
- **The defect:** the host's "nothing can arrive: every other Manager is waiting" refusal (`internal/run/inbox.go`, `othersRunningLocked`) counted a Manager as running for its whole response, including while it was blocked inside `await` (`running` is set when a response starts and cleared when it returns). So each of the two saw the other as running.
- **How it ended:** since S2, serving `await` is host time and does not run the idle clock. Only the Apply deadline (23:50:39Z) ended the Apply, about 76 minutes after the deadlock began, at no provider cost.
- **Cleanup and census after the host deadlock:**
  - The controller's cleanup closed all owned sessions and copies.
  - Afterwards there were 0 store containers and 0 store volumes, the temporary area was empty, there was one worktree, the credential scan found nothing, and the login was unchanged.
- **The fix:** [#759](https://github.com/Glacius-Labs/Markitect/pull/759) merged to Main at 23:55:04Z, during control C, so B2 and C ran on F_fix without it. In it:
  - a Manager blocked inside `await` counts as waiting;
  - a Manager that starts blocking wakes the other blocked Managers;
  - when every other Manager waits, the refusal goes to the Manager that holds an unanswered question;
  - a Manager has at most one `await` at a time.

### C: unchanged re-application

- **Input:** B1's published candidate unchanged (commit `754e8eef`, tree `30942ae5`), with B1's state carried. The controller's gate required B1 to have passed cleanly: technical pass, readback, no residue or credential hit, protected inputs unchanged.
- **The fact:**
  - Every Run's `origin.verified` names B1's Verify Run `86f3f73b17500a7b88cd0427b266e3a6-3-verify`.
  - The compared trees are equal: C's bound input `30942ae5`, and the candidate B1's Verify approved, `30942ae5`.
  - The basis `57f5ff694ea3…` and the object format are equal too.
  - The input tree is the binding's. `holds` is true.

| Run | Messages charged | Left | Asks / replies | Refused calls | Root: wall = host-served + provider; longest idle |
| --- | ---: | ---: | --- | --- | --- |
| Assessment | 21 | 235 | 0 / 0 | — | 118.1 = 51.4 + 66.7; 7.4 s |
| Implement | 21 | 214 | 0 / 0 | — | 536.1 = 474.1 + 61.9; 17.9 s |
| Verify | 47 | 167 | 17 / 21 | 4 `reply`, 5 `judge` | 240.3 = 78.6 + 161.7; 11.7 s |

- **What changed:**
  - Assessment approved with no Findings, so Implement received no Initial Findings.
  - In Implement, the `workshop` Manager replaced the program's `bufio.Scanner` loop with line-bounded reading and added a regression test. It bound the change to `workshop/program`, and Root integrated it.
  - The changed paths are `realization/cmd/makerspace/main.go`, the new `main_test.go` and one registry entry.
  - The W0 model's `workshop/program` says the program writes and flushes each response before it reads the next request. A `Scanner` can read bytes of the next request into its buffer before flushing. So the change corrects a gap against that wording, which B1's own Verify had approved. On the wire, the responses and their order are the same either way.
- **How Verify judged it (47 messages, against 21 for the other Runs):**
  - 13 Managers first judged **reject**, bound to `workshop/program`. Some had read the diff in the reverse direction (`git diff aa12b745 754e8eef`), taking the input's `Scanner` loop for the new code and the repair for a regression.
  - Root first agreed in replies to their questions (00:11:22–00:11:37Z, "tracking this workshop/program regression").
  - At 00:11:45Z Root posted "Correction to my earlier reply: I reversed the direction". Those replies were refused, because the questions were already closed.
  - Five later `judge` calls were refused by the fixed-handoff rule ("your handoff … is fixed; no further judgment is needed"). These were Managers trying to change a judgment already handed off.
  - The Run ended with Root's **approve** and no Findings.
- **Verdict against the predeclared condition** ("no published change, fact held, any changed candidate rejected by its Verify, target census unchanged"): **not met**. The fact held, but a changed candidate was approved and published.
- **Against the contract:** [concept §11.3](../concepts/v1-concept.md#113-verify) and [DEC-073](../concepts/register.md#dec-073-change-minimality-on-an-unchanged-model) say that when the host states the input was already verified and Verify finds no defect, any change is a Reject. C's Verify approved the changed candidate with **no Findings**, so its approval names no defect that would justify the change.
- **Two separate readings:**
  1. **Quality:** the change itself is a Definition-bound correction of a gap against `workshop/program`'s flush-before-read wording, which B1's own Verify approved. C's Verify reached its approve only after a reversed reading of the diff. Whether the Verify judgments are consistent across B1 and C is a quality question this control raises but does not settle.
  2. **Contract gap:** when the fact holds and the candidate differs from the input, the host accepts a Verify approval without a Definition-bound Finding that names the defect. Requiring one would make the §11.3 rule checkable by the host.

### B2-fix: fresh C4 on F_fix2

- **Setup:** the same fresh C4 input as B2 (tree `0e8583a4`, digest `e1f3acb6`), with fresh state, limits and gates; only the source differed (F_fix2).
- **Sequence:** Assessment approve, Implement submitted, Verify **reject**, Implement submitted, Verify approve. Launched at 00:21:14Z, ended at 01:08:33Z.

| Run | Messages charged | Left | Asks / replies | Refused calls | Root: wall = host-served + provider; longest idle |
| --- | ---: | ---: | --- | --- | --- |
| Assessment | 21 | 235 | 0 / 0 | — | 102.9 = 50.4 + 52.5; 7.2 s |
| Implement | 38 | 197 | 73 / 72 | 2 `ask`, 1 `reply`, 1 `judge`, 4 `await` | 905.4 = 567.9 + 337.6; 51.0 s |
| Verify (reject) | 21 | 176 | 0 / 0 | — | 221.8 = 78.0 + 143.8; 44.4 s |
| Implement (correction) | 23 | 153 | 2 / 2 | — | 581.4 = 452.2 + 129.2; 21.8 s |
| Verify (approve) | 21 | 132 | 0 / 0 | — | 183.9 = 78.5 + 105.5; 28.0 s |

- **The await fix:** the fresh C4 completed. No Manager stalled in `await`, and every Run closed.
- **Question traffic (WP3a):** 73 asks and 72 replies in the first Implement were charged 38 messages; 2 asks were refused by the open-question cap. In the same situation M-c3-C4 used 231 asks and all 256 messages; B2 used 84 asks and 40 messages before its stall.
- **WP3c and P2:**
  - The first Verify rejected with three Findings, each bound to a current Definition. Two named `workshop/delivery` (request-level tests do not cover a successful request and every refusal code for every operation), and one named `policies/authorization/permission-matrix` (the four-role matrix is checked as metadata, not through requests).
  - With 176 messages left against the 42 a pair needs, the correction pair started. The Findings reached the correction Implement, and the second Verify approved the published candidate.
  - P2 verdict **exercised**, with every check true: `rejectsWithFindings`, `inputsChain`, `publishedIsLastVerify`, `freshSessions`, `closures`, `readback`, `processSucceeded`, `withinMessages` and `endedByDeadline`.
- **WP2:** the longest provider-idle stretch of any Root turn was 51.0 s.

## Quota

Before each launch the gate read the host login's usage live (`account/rateLimits/read`, no model turn), and took the runtime login's latest reading from the newest Apply transcripts. It records both.

| Point | Host | Runtime | Gate |
| --- | ---: | ---: | --- |
| Before A | 61 % | 51 % (Phase C, 16:51Z) | open |
| After A / before B1 | 61 % | 51 % | open |
| After B1 / before B2 | 65 % | 65 % | open |
| After B2 / before C | 67 % | 67 % | open |
| After C | 69 % | 69 % | open |
| Before B2-fix | 69 % | 51 % (Phase C: the F_fix root was not among B2-fix's evidence roots) | open |
| After B2-fix | 75 % | 75 % | open |

From B1 on, the host and runtime readings were equal whenever both came from current runs, which indicates that the two logins belong to one account. All readings are of the weekly window.

## Deviations and rule changes

- **Lock as reservation.** The coordinator asked for `NATIVE-RUN.lock` to be held from P0 through all controls.
  - The controller's own lock is created exclusively and released after each command.
  - So the lock was held as one reservation, from 21:31:16Z to 00:16:06Z (holder `step4-armM`, run `fixplan-controls-714ae136`), and released by its nonce after C.
  - B2-fix was dispatched later. It got its own reservation, from 00:20:01Z to 01:09:06Z (run `fixplan-b2fix-0075171c`), also released by its nonce.
  - The controller used a private lock in each source root, one command at a time, and the launchers required the reservation's nonce.
- **Machine rule.** After an owner direction, the coordinator made quietness a requirement only for counted study runs. For these controls the quiet check became record-only "from the next launch on".
  - A, B1 and B2 had launched under the blocking check and passed it.
  - C and B2-fix launched under the record-only rule, and both were quiet at launch.
  - Other sessions could run targeted builds and package tests during the controls, but not full suites or image builds. The controls' samples record no such load at launch, but they were not sampled during the runs.
- **C's first launch call stopped before launching.** The new record-only quiet block ended the step script under strict shell mode on a `grep` with no match. That happened after the packet was prepared and before any launch or provider call. The block was fixed and tested, and the same prepared packet was then launched once. There was no second attempt.
- **B2 was not stopped early.** The deadlock was diagnosed at 23:19Z and reported to the coordinator. The Apply was left to its deadline, so the one-attempt record has no operator intervention.
- **Probe login.** Control A used the host Codex's default login, not the Applies' runtime login, because the runtime login file may only be hashed. The quota readings indicate one account.

## Limits

- **One attempt each,** on one fixture, one provider and model, one machine. The controls bound what the fixes do in these situations; they are not study outcomes.
- **A is a proxy:** a provider error from a per-turn model override, not a `serverOverloaded` error. It shows thread acceptance only.
- **B2 alone shows the accounting change,** but not that a fresh C4 completes. B2-fix, on F_fix2, shows the completion with #759's fix. #759 merged to Main at 23:55:04Z, during C; F_fix2 carries only its two commits.
- **WP3c:** only B2-fix had a Verify reject, and its pair had ample messages (176 against 42). The refusal path of the check, too few messages for a pair, was not exercised.
- **C is one case,** on W0's scale, with one gap against the model's wording in its input. A no-change re-application of a gap-free candidate was not run.
- **B2-fix is one fresh C4.** That its correction pair completed shows the mechanism works once, not how often Verify rejects.
- **Root split:** host-served time comes from the transcripts' call and reply timestamps; the idle clock's own state is not logged.
- **Not covered:** release, deployment, other providers, account refresh and owner acceptance.

## Retained evidence

Execution host root: `Documents\Codex\Markitect-V1-Requalification\fixplan\`.
- `714ae136\`: `source\`, `build\`, `launch\` (launch records), `packets\m-b1-w0\`, `m-b2-c4\`, `m-c-xnc\`, each with `attempt-01\` (`run-summary.json`, `package-result.json`, `files.sha256`, `state\` with Run records, candidate bundles and transcripts), `inputs\`, `quota\`, `quiet\`, `analysis-*.json` (per-Run reports), `pin.env`, `lock.nonce`.
- `0075171c\` (B2-fix): `source\`, `build\`, `launch\`, `packets\m-b2fix-c4\attempt-01\`, `inputs\`, `quota\`, `quiet\`, `analysis-b2fix-c4.json`, `pin.env`, `lock.nonce`.
- `probe-wp1\runs\a01\` (`result.json`, `transcript.jsonl`) and `probe-wp1\quota\`.
- `controller-fixplan\`, `launch\`, `quota\`, `analysis\` and `probe-wp1\`: the kit and its offline tests. `rehearsal-xnc\` holds the offline rehearsal of control C's input and packet.

Design, decisions and run notes: the fix-plan control notes of the coordinating session.
