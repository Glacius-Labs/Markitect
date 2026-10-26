# V1 comparison Phase C, arm M, on F_final 4d23f7ae — 26 October 2026

This record covers the counted native runs of arm M in Phase C of the [comparison protocol](v1-comparison-protocol-20261024.md#addendum-phase-c-makerspace) ([DEC-072](../concepts/register.md#dec-072-stable-v1-justified-by-data)), N = 3 chains of ten steps each. The coordinator dispatched them under the [standing evidence authorization](../concepts/sources/v1-source-implementation-approval-20261023.md#subsequent-standing-evidence-authorization). The study ordered each step and recorded every repeat or no-candidate decision ([C4](v1-comparison-protocol-20261024.md#c4-steps-order-and-attempts)). The record holds arm M's raw outcomes only. The study's assessor judges the candidates, and the comparison with arm B belongs to the study's own results document.

## Result

- **Chain 1:** all ten steps published a candidate (Apply status `passed`). C9's strict no-change condition was not met: Implement changed three test files.
- **Chain 2:** nine of ten steps published a candidate.
  - C5 has no candidate: all three attempts ended on provider capacity during Assessment.
  - C9's strict no-change condition was not met: one test file changed.
- **Chain 3: no step published a candidate.** Every step therefore started fresh from the neutral input (C4). Of its ten counted Applies:
  - seven ended at Root's 600 s per-message limit during Implement;
  - one exhausted the message allowance after a Verify reject (C1);
  - one exhausted it on question traffic between Managers (C4);
  - one was stopped by the no-progress rule (C8).
  An earlier W0 attempt failed in preflight on a Docker Desktop idle shutdown.
- **Totals:** 33 attempts, 32 of them with an Apply: 37,167 s of Apply time, 3,532 messages and 1,627,798,064 provider-reported tokens.
- **Hygiene:** no residue and no credential hit in any attempt. The login file was unchanged across every Apply.
- **P2:** the Verify-reject path was exercised once, partially, in chain 3's C1. Its observation is `technical` because the final Verify could not start ([P2](#p2-observation)).
- **The limit that dominates chain 3 is a busy Root's total.** In chains 1 and 2, which continued from candidates, Root's Implement turn used at most 414 s of provider time. Chain 3's fresh starts put 18–62 questions on Root and reached 527–599 s ([Root split](#root-turn-split)).

## Binding

| Item | Value |
| --- | --- |
| Source F_final | `4d23f7aea0576ff5c477fd301b456a24bbb6dcb3`, tree `30fb0eca4316928b8f534267e581df7f7fe0c8ba`: Main after [#725](https://github.com/Glacius-Labs/Markitect/pull/725) (the per-message clock pauses while the host serves a tool call). Since Phase B's F′ it adds #705, #706, #722, #723, #724, #726, #725 and the Phase C materials |
| Product binary | Built from F_final with `-trimpath` and go1.27.1; module version `v0.0.0-20261025222723-4d23f7aea057`. SHA-256 `54cff11ffad14d88d98658745fb39b6a4b0dbdccd18dfa228a8d721c5f11057e` |
| Images | Built from F_final's `packaging/runtime`. Base `sha256:c2920c613186459de91c78b416fe4d144d22dd97337a25ad86239a4d8310f915`; Go `sha256:3d75d81f438e71254d01f013cd35e0c61f9eecfb8abba723b34e12158c33e943` with `codex-cli 0.162.0` |
| Provider | Codex App Server 0.162.0, `gpt-6-sol`, effort `high`, normal account login. The login file was only hashed; its hash was equal before and after every Apply |
| Configuration and limits | The example runtime profile and run configuration (format 3): Apply deadline 5,400 s, 256 shared messages, 32 Managers, `messageTimeoutSeconds` 600. One attempt per step unless the study recorded a repeat. Shared `NATIVE-RUN.lock` |
| Materials | Phase C materials freeze v2 (`4840fb20`). Every step's snapshot manifest and both execution files were pinned and re-verified before each input was committed. `w0.yaml` is `2741e461…` and `c7.yaml` is `749d2c81…`. Model digests: W0 `cfb7a9fe`, C1 `f7a8bd29`, C2 `daf68db9`, C3 `d9d631c7`, C4 `e1f3acb6`, C5 `5e61dc52`, C6 `3a081c99`, C7 `4fb0c37a`, C8 and C9 `2505dc4b` |
| Inputs (C4) | A step continues from the chain's last published candidate plus one neutral commit that replaces `.markitect/model/` with the step's snapshot. At C7 that commit also replaces `execution.yaml` with `c7.yaml` and records owner attention. After a published C8, C9 is that candidate unchanged, with a strict no-change expectation. With no published candidate yet, a step starts as W0 does: a fresh repository from the step's snapshot and its pinned execution file. State is carried from the previous step's latest attempt |
| Managers | 21 for W0–C6, 20 for C7–C9 (C7 folds `billing/payments` into `billing/invoices`) |
| Controller | `controller-phase-c`: `requalify.py` SHA-256 `c21fd27569fe8d088c3de4670949fffa137fe220e7146c34e99d3c09787aac25`, 55 offline self-tests, independently reviewed READY before the first counted run. It enforces the C4 chain rule (input from the last deliverable, state from the previous step's latest attempt), the snapshot and execution pins, the exact Manager set per step and the P2 observation |
| Quietness | Before each go, the study checked that the lock was free and that no host test suite or other Apply was running. From M-c3-W0 attempt 02 on, it also checked that Docker answered. The step script refused to launch while the lock was held or a host test or container was running |

## Before the counted series

Uncounted smokes on W0 found three defects, all fixed before the first counted run:

- **Per-message limit.** A re-smoke on the frozen materials hit the 600 s per-message limit in Implement while Root was waiting on its children. That led to [#725](https://github.com/Glacius-Labs/Markitect/pull/725): the clock measures provider work only and pauses while the host serves a tool call.
- **Check-profile mapping.** The final smoke on F_final failed on a controller defect: it renamed the example's `go` check profile as for the Library. The controller was fixed.
- **Controller verdict.** The replacement smoke `ee91c7e6` passed and published, but the controller reported a false negative: it expected exactly one `project-tests` check, a Library assumption. The controller was fixed and re-reviewed.

One earlier re-smoke failed before any Run on a container-engine timeout under host load. The smoke records are in the step-4 notes ([Retained evidence](#retained-evidence)).

## Runs

The study ran the steps in pairs under Phase A's parity rule: M first in chains 1 and 3, B first in chain 2. Seconds are Apply process time. Messages are the total and the split per Run. A passed Apply ran Assessment, Implement and Verify with the full Manager set in fresh native sessions, and its final bundle reopened exactly. P2 is the observation of [the Verify-reject path](#p2-observation).

| Packet | Apply | C1 / scenario | s | Messages | Paths | Final commit / tree | Tokens | P2 | Cause class | `package-result.json` |
| --- | --- | --- | ---: | --- | ---: | --- | ---: | --- | --- | --- |
| m-w0-c1 | `060f5ef1dc138dcae9fe55b42d606524` | pass / met | 1,361 | 170 (21/128/21) | 59 | `450580ff` / `bcca9da2` | 67,665,567 | not exercised | — | `c2d6b2f10461` |
| m-c1-c1 | `9cc08bcd836a780a1ecb2081dc0faa75` | pass / met | 1,101 | 76 (21/34/21) | 14 | `71a43419` / `93362e87` | 45,039,348 | not exercised | — | `0a8202511095` |
| m-c2-c1 | `5a90a28b6c42e66127fb538eedd2f2ff` | pass / met | 1,160 | 78 (21/36/21) | 21 | `f6e25868` / `e98d3ee6` | 50,441,335 | not exercised | — | `d1c70a0a8e27` |
| m-c3-c1 | `1a31c74e3fca9e94799917f8df32d249` | pass / met | 1,159 | 63 (21/21/21) | 11 | `116d7b8d` / `917cb86b` | 45,265,961 | not exercised | — | `ce4b21a07837` |
| m-c4-c1 | `3fcaac686b5ef0f2c9f603f5bc485c7e` | pass / met | 1,158 | 75 (21/33/21) | 33 | `4727f7d1` / `ac263ff7` | 54,047,493 | not exercised | — | `b6be04a8eebd` |
| m-c5-c1 | `aeab25c00436df9bbded81b5ca1f9e57` | pass / met | 1,150 | 63 (21/21/21) | 15 | `ab9e10d9` / `9a67ca1d` | 44,565,696 | not exercised | — | `0d368b59b510` |
| m-c6-c1 | `f5877b0d6d8d1c68aa399717ddb7f503` | pass / met | 1,220 | 71 (21/29/21) | 9 | `48837206` / `4bc779a5` | 46,436,808 | not exercised | — | `25c167286db6` |
| m-c7-c1 | `535615a93b72d098a8f9c203e731a813` | pass / met | 1,162 | 74 (20/34/20) | 19 | `baf9c521` / `e798d3e1` | 47,018,450 | not exercised | — | `4670f8b880c8` |
| m-c8-c1 | `0bbd30702fd38a02561b0adea5e5a438` | pass / met | 1,166 | 73 (20/33/20) | 13 | `685aa5f5` / `e91630a8` | 46,699,700 | not exercised | — | `fd60747a346e` |
| m-c9-c1 | `8e8f4f06e89dca2edd99dfb075142e05` | pass / strict not met | 1,105 | 62 (22/20/20) | 3 | `fb347890` / `f92b2a66` | 44,348,853 | not exercised | scenario: test-only change | `b03e3f27242e` |
| m-w0-c2 | `bbdfdadd73e821289c4f1d67fb2bfb99` | pass / met | 1,477 | 160 (21/117/22) | 67 | `592be54d` / `b4597dbc` | 63,196,626 | not exercised | — | `a4aee966dd0b` |
| m-c1-c2 | `b233816ab1cdd04146eb3c1ef64baaaa` | pass / met | 1,408 | 109 (24/64/21) | 33 | `daff9efd` / `be118c65` | 54,599,704 | not exercised | — | `ec2ab028c6f2` |
| m-c2-c2 | `415b6132dd14e020303ad03077c2b698` | pass / met | 1,262 | 72 (21/30/21) | 18 | `563fa21d` / `e6e81f52` | 43,141,212 | not exercised | — | `b0bd7d7581e0` |
| m-c3-c2 | `dd667fe12d6c41355cef265e0c2a1a1c` | pass / met | 1,276 | 66 (21/24/21) | 9 | `7f986c21` / `367b0527` | 41,734,352 | not exercised | — | `9737559ddfbd` |
| m-c4-c2 | `cc3e8a1aad9b4657bcfff164efd68b69` | pass / met | 1,414 | 79 (21/37/21) | 25 | `f0ae7e58` / `4a5dd20d` | 49,179,424 | not exercised | — | `0afe653da208` |
| m-c5-c2 | `0ec33adbedb83b36a2f9715aab8d941a` | fail / not assessed | 258 | 21 (21) | 0 | — | 12,074,402 | not exercised | infrastructure: provider capacity | `953e6fe57cc8` |
| m-c5-c2-02 | `27bcdd9b6b68def8cdda7110bd381019` | fail / not assessed | 199 | 21 (21) | 0 | — | 5,220,403 | not exercised | infrastructure: provider capacity | `b7dc45935c7b` |
| m-c5-c2-03 | `f9a51bd316322fae4b5065cda6e666b2` | fail / not assessed | 325 | 21 (21) | 0 | — | 14,137,535 | not exercised | infrastructure: provider capacity | `39f5ad0ace89` |
| m-c6-c2 | `fb988416a9e39ecb13b6692211dd3d9f` | pass / met | 1,436 | 76 (21/34/21) | 18 | `ff9a8c7f` / `0f0b9a7c` | 48,712,184 | not exercised | — | `60d85938577f` |
| m-c7-c2 | `e9ba246956decdd8df4afdbc2a6d5099` | pass / met | 1,210 | 60 (20/20/20) | 12 | `391330c6` / `21de344c` | 41,394,943 | not exercised | — | `5f25d3b6e797` |
| m-c8-c2 | `d08a1acf289a3e973ac65a6d45b5f076` | pass / met | 1,294 | 64 (20/24/20) | 10 | `ac35b224` / `0444c80a` | 44,235,638 | not exercised | — | `17e9c9678adf` |
| m-c9-c2 | `96bda99fe45efa29b4d80b7d809ed742` | pass / strict not met | 1,266 | 60 (20/20/20) | 1 | `fd4a8d72` / `564d8d6e` | 44,837,593 | not exercised | scenario: test-only change | `770d27c7c123` |
| m-w0-c3 | — | incomplete / not assessed | 0 | 0 | 0 | — | 0 | — | infrastructure: Docker idle shutdown in preflight | `50479204e41e` |
| m-w0-c3-02 | `83c5229fdcc8127b9ff1ff45c6b1f748` | fail / not assessed | 1,112 | 152 (21/131) | 0 | — | 56,544,323 | technical | product: Root message limit | `f35f097c3c43` |
| m-c1-c3 | `59a55e9f12536e1f1ede7ce68ad1b600` | fail / not assessed | 2,670 | 241 (21/175/21/24/–) | 0 | — | 96,526,601 | technical | product: allowance short for the correction pair | `e6d1cc8ac0e2` |
| m-c2-c3 | `1c39337b3f14511b13b32cd4def100e8` | fail / not assessed | 1,266 | 186 (21/165) | 0 | — | 65,032,467 | technical | product: Root message limit | `97bfe764e6c8` |
| m-c3-c3 | `f8f7984b7c4d22507f8b579910eec74b` | fail / not assessed | 1,284 | 122 (21/101) | 0 | — | 52,606,248 | technical | product: Root message limit | `5753f897bb7f` |
| m-c4-c3 | `5d1adf42933f97035b34a6cbd65db316` | fail / not assessed | 897 | 256 (21/235) | 0 | — | 74,536,138 | technical | product: allowance spent on question traffic | `0ce503753b0a` |
| m-c5-c3 | `4c0d6147eb3c3f7e6890fc5144617854` | fail / not assessed | 1,120 | 176 (21/155) | 0 | — | 64,232,502 | technical | product: Root message limit | `795c22f68237` |
| m-c6-c3 | `1dbfa41d8922d3ef83b3ea8321c82149` | fail / not assessed | 1,074 | 185 (21/164) | 0 | — | 66,338,617 | technical | product: Root message limit | `b7eaa828916d` |
| m-c7-c3 | `59169b034763ed4a0579568016610135` | fail / not assessed | 1,045 | 219 (20/199) | 0 | — | 66,482,893 | technical | product: Root message limit | `515421ffc340` |
| m-c8-c3 | `8b5a7ab2487a29eaf91cb35a67f6d4e2` | fail / not assessed | 1,018 | 187 (20/167) | 0 | — | 63,712,141 | technical | product: no-progress stop | `8ac1d789a836` |
| m-c9-c3 | `3107060271a6040cb2490d52d6fa0781` | fail / not assessed | 1,115 | 194 (20/174) | 0 | — | 67,792,907 | technical | product: Root message limit | `8a4396c624b5` |

| Chain | Attempts | Published | Apply s | Messages | Tokens |
| --- | ---: | ---: | ---: | ---: | ---: |
| 1 | 10 | 10 | 11,741 | 805 | 491,529,211 |
| 2 | 12 | 9 | 12,825 | 809 | 462,464,016 |
| 3 | 11 | 0 | 12,600 | 1,918 | 673,804,837 |
| All | 33 | 19 | 37,167 | 3,532 | 1,627,798,064 |

Every attempt had no residue and a clean worktree count. Every Apply left an empty temporary area and kept the login file unchanged, and the credential scan found nothing. No step raised a model blocker.

## What the runs show

- **Continuation (chains 1 and 2).**
  - Every Apply that continued from a candidate passed with Assessment, Implement and Verify, apart from the three capacity failures at chain 2's C5, and none took the correction branch.
  - At C7, the neutral commit replaced `execution.yaml` with `c7.yaml`, recorded as owner attention, and the Apply ran with 20 Managers. The fresh C7–C9 Applies of chain 3 were built with `c7.yaml` from the start, so their bindings record no replacement.
- **Strict no-change (C9) was not met in either chain that reached it.** Assessment approved, and Implement then published test-only changes:
  - chain 1: three paths, a reworked `billing_test.go` case and new tests in `maintenance_integration_test.go` and `lifecycle_test.go`;
  - chain 2: one path, `realization/bookings/request_test.go`, with 14 lines of opening-hours and four-hour-maximum boundary cases.
  Neither changed production code.
- **Fresh starts (chain 3).** Without a published candidate, every chain-3 step rebuilt the whole Makerspace from its snapshot. The children's shared needs, chiefly the not-yet-built `workshop` API, turned into questions to Root and relays between siblings. In the seven Applies that ended at the message limit, 16 to 20 children had handed off (all of them in four), and Root's own turn was the one that hit the limit.
- **Three further limit modes.**
  - **C1, allowance short for the correction pair:**
    - Verify rejected with 43 Findings, and the correction Implement used 24 messages.
    - The second Verify was then refused: 15 of 256 messages remained, but a Run needs at least one per Manager (`Allowance.minimum`).
    - The correction pair had started with 39 messages left, fewer than the 42 that Implement plus Verify need.
    - Apply checks the remaining allowance only at its start, and the end reports as "invalid Run request" rather than as a budget stop.
  - **C4, question traffic:** the Implement sessions made 231 `ask` and 203 `reply` calls in about eight minutes. Root alone made 60 and 47, relaying the `workshop` API child by child. The allowance ran out with 4 of 20 handoffs in.
  - **C8, no-progress stop:**
    - `safety` handed off at 16:24:08Z.
    - Root's earlier questions to it stayed open, because `safety` had answered them with new `ask` calls rather than `reply` with the question's ID.
    - The host re-delivered them. After two responses without a tool call, the no-progress rule stopped the Run with 18 of 19 handoffs in.
    - The 20 copy closures then reported "exec container is closed". No container or volume remained.

## Root turn split

For every Root Implement turn, the split comes from the retained transcripts. Wall time runs from `turn/started` to `turn/completed`, or to the turn's last event if it was still open. Host-served time is the union of the transcript's `tool-call` → `tool-reply` intervals; the per-message clock pauses during them. Provider time is wall minus host-served time, which is what the clock counts. Shell commands run in the Agent's container and count as provider time. Asks and replies are Root's calls in that session. Closing turns shorter than 30 s are left out.

| Packet | Root Implement turn | Wall s | Host-served s | Provider s | Asks | Replies |
| --- | --- | ---: | ---: | ---: | ---: | ---: |
| m-w0-c1 | completed | 674.9 | 309.5 | 365.4 | 17 | 29 |
| m-c1-c1 | completed | 414.4 | 278.4 | 136.0 | 0 | 3 |
| m-c2-c1 | completed | 479.8 | 285.0 | 194.8 | 2 | 8 |
| m-c3-c1 | completed | 472.5 | 319.4 | 153.2 | 0 | 0 |
| m-c4-c1 | completed | 456.0 | 257.7 | 198.3 | 0 | 4 |
| m-c5-c1 | completed | 449.9 | 303.1 | 146.7 | 0 | 0 |
| m-c6-c1 | completed | 477.1 | 269.8 | 207.3 | 1 | 2 |
| m-c7-c1 | completed | 438.3 | 299.6 | 138.7 | 1 | 6 |
| m-c8-c1 | completed | 464.2 | 273.6 | 190.6 | 2 | 6 |
| m-c9-c1 | completed | 423.0 | 298.8 | 124.1 | 0 | 0 |
| m-w0-c2 | completed | 734.9 | 321.1 | 413.8 | 10 | 28 |
| m-c1-c2 | completed | 688.7 | 386.2 | 302.5 | 3 | 13 |
| m-c2-c2 | completed | 496.6 | 314.2 | 182.3 | 0 | 4 |
| m-c3-c2 | completed | 497.8 | 358.0 | 139.9 | 0 | 2 |
| m-c4-c2 | completed | 529.9 | 369.3 | 160.5 | 1 | 6 |
| m-c6-c2 | completed | 643.4 | 418.4 | 225.0 | 2 | 8 |
| m-c7-c2 | completed | 472.1 | 359.5 | 112.6 | 0 | 0 |
| m-c8-c2 | completed | 517.4 | 360.3 | 157.0 | 0 | 3 |
| m-c9-c2 | completed | 497.7 | 361.5 | 136.3 | 0 | 0 |
| m-w0-c3-02 | open when the Run ended | 767.2 | 172.8 | 594.4 | 21 | 28 |
| m-c1-c3 | completed (first) | 1,029.1 | 502.2 | 527.0 | 25 | 42 |
| m-c1-c3 | completed (correction) | 606.3 | 433.4 | 172.9 | 1 | 2 |
| m-c2-c3 | open when the Run ended | 924.5 | 326.3 | 598.2 | 32 | 35 |
| m-c3-c3 | open when the Run ended | 885.8 | 291.1 | 594.6 | 18 | 26 |
| m-c4-c3 | open when the Run ended | 482.6 | 92.7 | 389.8 | 60 | 47 |
| m-c5-c3 | open when the Run ended | 738.9 | 141.9 | 596.9 | 35 | 35 |
| m-c6-c3 | open when the Run ended | 719.8 | 120.8 | 599.0 | 51 | 26 |
| m-c7-c3 | open when the Run ended | 677.7 | 79.3 | 598.4 | 62 | 38 |
| m-c8-c3 | open when the Run ended | 652.2 | 104.9 | 547.3 | 30 | 46 |
| m-c9-c3 | open when the Run ended | 699.3 | 101.1 | 598.2 | 38 | 46 |

- **#725 holds natively.**
  - The seven Root turns that ended at the limit ran 653–925 s wall, well past 600 s, because their host-served time was excluded.
  - Each stopped when provider time reached the limit: its last recorded event was at 594–599 s.
  - C1's first Implement turn ran 1,029 s wall but used only 527 s of provider time, so it completed.
  - In C4 and C8, Root's turn was still under the limit when the Run ended for the reasons above.
- **The limit bounds a busy turn's total, not inactivity.** No single model step was slow: the turns that hit the limit spent their provider time answering children, delegating, reviewing and running commands. Chains 1 and 2 stayed at or below 414 s because a continuation leaves Root few questions (0–17 asks).

## P2 observation

[Addendum C](v1-comparison-protocol-20261024.md#c4-steps-order-and-attempts) records a P2 observation, of the public Verify-reject path, on every M Apply. It never changes a study outcome. The verdict comes from the stored Apply report:

- **exercised:** the sequence is Assessment, Implement, Verify, Implement, Verify; the first Verify rejects with Findings; the last approves; and the published commit and tree equal the last Verify candidate;
- **not exercised:** Assessment blocks, or the first Verify approves;
- **technical:** anything else.

Results:
- All 22 Applies of chains 1 and 2 are **not exercised**: the first Verify approved, or the Apply ended in Assessment.
- All ten Applies of chain 3 are **technical**: nine never reached a Verify, and C1's final Verify never started.
- **C1 of chain 3 is the only partial exercise.** The first Verify rejected with 43 Findings. The correction Implement received them (`rejectsWithFindings` and `inputsChain` true) and submitted. The final Verify was refused for lack of messages, so `publishedIsLastVerify`, `closures`, `readback` and `freshSessions` are false: there is no fifth Run record.

No dedicated P2 attempt was run. The design reserves those for the case that Phase C shows no exercised path; that decision belongs to the coordinator.

## Infrastructure faults, repeats and probes

**M-c2-C5, provider capacity.** All three attempts ended in Assessment with "Selected model is at capacity" (`serverOverloaded`, `willRetry: false`). None reached Implement.

| Attempt | Apply | Fault at | Handoffs before | Closures | Decision |
| --- | --- | --- | ---: | --- | --- |
| 01 `m-c5-c2` | `0ec33adb` | 07:08:54Z | 9/21 | 23/44 (the copy closures report "exec container is closed") | repeat 1 (study, 07:11Z) |
| 02 `m-c5-c2-02` | `27bcdd9b` | 07:14:39Z | 0/21 | 44/44 | wait; capacity probe first |
| 03 `m-c5-c2-03` | `f9a51bd3` | 08:21:50Z | 20/21 | 44/44 | no-candidate (coordinator) |

- **Probes.** Each uncounted probe was one short `codex exec` request with the counted model, effort and login, in a throwaway directory under the lock. It used no Markitect and no study input.
  - Probe 01 succeeded at 08:15:47–08:15:57Z, before attempt 03.
  - The chain resumed at C6 only after two more successful probes at least 30 minutes apart: 02 at 09:51:42Z and 03 at 10:22:21Z. Each returned exit 0, reply "ok" and no capacity message.
- **C6** continued from C4's candidate, with state from C5's latest attempt.

**M-c3-W0, Docker Desktop idle shutdown.**
- Attempt 01 (12:14:56Z) failed in preflight: `docker image inspect` hit its 20 s timeout and the container census its 8 s timeout.
- Docker Desktop's backend log shows its idle module scanning volumes from 12:15:13Z and stopping the VM at 12:15:54Z ("idle: shutdown"). It restarted on demand at 12:16:08–12:16:24Z.
- No Apply, Session or provider call started. The study recorded repeat 1, the infrastructure rule.
- From then on the study checked that the pinned image answered `docker image inspect` before each go.

Capacity failures and the preflight failure were the only repeats; every product failure was recorded as no-candidate.

## Frozen-tooling verification

- `controller-phase-c` passes `sha256sum -c SHA256SUMS` for `launch.py`, `rehearse-phase-c.sh`, `requalify.py` and `test_requalify.py`. No file in it is newer than its `SHA256SUMS`.
- **Step script.** `launch\phase-c-step.sh` lies outside the controller. It is SHA-256 `3416123a37cf421bd3502508b1d4e3e66e3eb9286b7cc88ffab4f7ebf65cf1e9`, the version written before the first counted run.
  - After M-c3-W0's preflight failure, a Docker-readiness wait and a fail-closed quiet check were added to it, from 12:18:06Z to 12:18:54Z, at the operator's own initiative.
  - The study declined changes to the launch helpers during the series, and the edit was reverted byte for byte, which the hash confirms. No launch ran in that window: attempt 01 started at 12:14:56Z and attempt 02 at 12:19:13Z.
  - Both improvements are recorded as findings for after Phase C.
- **Read-only analysis.** The scripts in `analysis\` (`root-split.py`, `fail-check.py`, `record-table.py`, `record-split.py`) read retained evidence only and were never part of a launch.

## Deviations and corrections

- **Root split, first reports.** For M-c3-W0 attempt 02 and M-c3-C2, the first reports said Root "never blocked". They had measured tool calls by Codex's `mcpToolCall` items, which Codex emits only after the host replies. Measured from the `tool-call` → `tool-reply` pairs, Root's awaits did block; the table above uses those numbers. The mechanism stayed as reported, and both the study and the coordinator received the correction.
- **Launch helper.** See [the frozen-tooling verification](#frozen-tooling-verification).
- **C7 and C9 in chain 3** were fresh starts. Their bindings record `replaced: null` and no owner attention, although the configuration equals chain 2's C7 and C8 (`c7.yaml`), and C9 expected a fresh build instead of a strict no-change.

## Limits

- **Chain 3 measures fresh builds only.** After W0 failed, C4 made every later step a fresh start, so chain 3 says nothing about continuing a candidate, and it is not comparable step by step with chains 1 and 2.
- **Root split method.** The host's own clock is not logged. Host-served time comes from the transcript's call and reply timestamps, which put the observed expiry at 594–599 s against the 600 s limit.
- **No retained-result reviews.** Unlike Phase B, the counted runs had no per-chain independent review of the retained results. This record was checked against the evidence by its own review.
- **P2** was exercised only partially, once.
- **Time of day.** Chain 2's capacity failures came in one morning window; the probes and the later steps succeeded.
- **Scope.** One fixture, one provider and model, one machine. Not covered: release, deployment, other providers, account refresh and owner acceptance.

## Retained evidence

Execution host root: `Documents\Codex\Markitect-V1-Requalification\4d23f7ae\`.
- `controller-phase-c\` (with `SHA256SUMS`), `build\`, `launch\` (step, wait and probe scripts, launch records).
- `inputs\<name>\` with `input-binding.json`.
- `packets\m-<step>-c<chain>[-NN]\`, each with `attempt-01\` holding every step record, `state\` (Run records, candidate bundles, transcripts, Notes, Apply report), `package-result.json`, `run-summary.json` and `files.sha256`.
- `probes\capacity-probe-0N\result.json`.
- `analysis\`: the read-only scripts and the per-Apply `<packet>.root-split.json`.

Run log, Root split and smoke records: the Phase C arm M notes of the coordinating session.
