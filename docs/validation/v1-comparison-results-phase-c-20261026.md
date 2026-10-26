# V1 comparison results, Phase C: the larger Makerspace project

Date: 26 October 2026. This record reports Phase C of the [V1 comparison protocol](v1-comparison-protocol-20261024.md#addendum-phase-c-makerspace), approved with N = 3 and frozen before the first counted run. Phase C is read by its own rule (C6) and is never pooled with [Phase A](v1-comparison-results-20261025.md) or [Phase B](v1-comparison-results-phase-b-20261025.md).

## Result

**Counts against Markitect** (C6, computed by `analyze.py --phase C` at R_C from the official ledger). B's per-chain totals of missed probe families are lower.

| | M (Markitect Apply on F_final) | B (one Codex agent) |
| --- | --- | --- |
| Missed probe families, chains 1 / 2 / 3 | 0 / 31 / 308 | 10 / 0 / 0 |
| Missed probe families, total | 339 | 10 |
| Conforming steps | 17 of 30 (95 % CI 37–75 %) | 20 of 30 (47–83 %) |
| Steps with a deliverable candidate | 19 of 30 | 30 of 30 |
| Time, all 30 steps | 37,167 s | 3,995 s |
| Tokens, all 30 steps | 1,627.8 M | 38.6 M |
| Messages | 3,532 | 30 |

- The one-sided rank-sum p is 0.95. M missed more probe families than B at every step, summed over chains. M also shows error types that B does not, starting with `no-candidate`. None of the three "supports" conditions holds.
- Neither arm raised a model blocker, and neither published an unsupported success.
- After chain 2 the two chains had gone opposite ways (M 0 and 31, B 10 and 0), so "supports" could no longer be reached. The coordinator ran chain 3 exactly as predeclared, and the reading above is the rule's output after chain 3.

**Descriptive sensitivity line, not the reading.** All 339 of M's missed families come from the 11 steps where M's Apply published nothing (C6 counts a step without a candidate as missing every family). On its 19 deliverable candidates M missed none. B missed 10, the same `P-AUDIT-ACCEPTED` defect at every step of chain 1, on 30 candidates. Without chain 2's C5 the chain-2 total would be 0 rather than 31. Chain 3's 308 has no such single cause: every M step in it failed on a product limit.

**Conclusion.** On the larger project, one Codex agent completed all 30 steps, and in two of three chains it met every hidden obligation. Markitect V1 met every hidden obligation whenever it delivered, but it delivered in only 19 of 30 steps:
- once the provider was at capacity (chain 2, C5);
- ten times its own run limits stopped it (all of chain 3).

It also failed the C9 no-change gate in both chains that reached C9, with test-only edits. It cost about nine times the time and forty times the tokens. Under the predeclared rule this counts against Markitect. The reliability of the Apply at this scale is the main finding.

## Execution

- **Sources.** Both arms used the F_final image `sha256:3d75d81f…` (F_final = `4d23f7ae`). The other bindings:
  - the assessor and oracle at R_C `4840fb20`;
  - the review tools at RR `4ea5b964`;
  - B's runner and prompt at R `74b7d7df`;
  - step 4's frozen Phase C controller (`SHA256SUMS` `9ce5372a…`) for every M Apply, all as bound in [C8](v1-comparison-protocol-20261024.md#c8-frozen-study-infrastructure).
  - The study infrastructure did not change in content during the counted runs. The one repair, A1, is a separate amendment to the ledger builder (incident 2).
- **Order** (26 October 2026, UTC; parity as predeclared):
  - chain 1, M first, 00:17–04:33;
  - chain 2, B first, 04:34–12:10, with waits after the capacity failures at M-c2-C5 (07:15–08:16 and 08:22–10:22);
  - chain 3, M first, 12:14–16:55.
- **Attempts and decisions.** Under C4 the operator recorded every decision in the step's run directory, and the coordinator confirmed each:
  - **M-c2-C5:** two `repeat`s (provider capacity), then `no-candidate` after the third capacity failure.
  - **M-c3-W0:** one `repeat` (Docker Desktop idle shutdown before any Run), then `no-candidate`.
  - **Other M steps without a candidate:** a single attempt each, recorded `no-candidate`.
  - **Arm B:** no repeat.
- **Assessment.** Every candidate was assessed at R_C under its step's profile: hidden cases, change census at C3, C7 and C9, own checks, path census and the C7 billing-area rule.
- **Reviews.** All 49 candidates (30 B, 19 M) got one blinded review by a fresh Sonnet reviewer at high effort with the frozen prompt and the Phase C rubric. None reported a critical finding, so no confirmation review was needed.
- **Contamination audit.** It covered 63 attempts (30 B, 33 M) with 9,481 commands and found 0 web items. Its 4 hits are all `git remote -v` / `git branch -a` listings inside M's own chain-3 working copies, which are read-only and touch no network. A supplementary Makerspace-specific scan of the same items (`makerspace-oracle`, `tools/comparison`, `phase-c/`, the evidence root, `reference/`, `mutant`) found nothing. The reading needs no recomputation.
- **Quietness.**
  - 57 of the 60 rows are quiet.
  - B-c3-C8 and M-c3-C9 were not quiet: targeted Go tests of three parallel product-fix sessions ran on the host (incident 7).
  - B-c3-C9 is unrecorded, because its 33 s run fell between the monitor's one-minute samples.
  - These three keep their quality result. Their cost is left out of the quiet-only cost comparison below.
- **Quota.** The weekly quota went from 20 % to 48 % in chain 1. The account's weekly window was reset early at about 06:08 on 26 October. The old window stood at about 57 %; the new one reached 18 % at the end of chain 2 and 51 % at the end of chain 3. The 90 % guard never applied.

## Every run

Time: M Apply seconds summed over the step's attempts, B session seconds. Tokens: M provider usage over all attempts, B's own tokens (the resumed thread's total minus the previous step's). Files: comparable files changed against the start (M registry files in brackets). Probe: failed probe families. NQ not quiet, U unrecorded.

| Step | M candidate | M s | M tokens | M msgs | M files | M probe | M conforming | B candidate | B s | B tokens | B files | B probe | B conforming |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| c1-W0 | `450580ff` | 1,361 | 67,665,567 | 170 | 41 (+18) | 0 | yes | `4c09dba0` | 295 | 1,245,352 | 6 | 1 | no (C2, C3) |
| c1-C1 | `71a43419` | 1,101 | 45,039,348 | 76 | 13 (+1) | 0 | yes | `200e6950` | 77 | 1,017,977 | 2 | 1 | no (C2, C3) |
| c1-C2 | `f6e25868` | 1,160 | 50,441,335 | 78 | 17 (+4) | 0 | yes | `4d33b7d7` | 177 | 1,710,797 | 5 | 1 | no (C2, C3) |
| c1-C3 | `116d7b8d` | 1,159 | 45,265,961 | 63 | 8 (+3) | 0 | yes | `8c3cae74` | 75 | 1,052,323 | 3 | 1 | no (C2, C3) |
| c1-C4 | `4727f7d1` | 1,158 | 54,047,493 | 75 | 26 (+7) | 0 | yes | `565fbe88` | 105 | 1,705,154 | 4 | 1 | no (C2, C3) |
| c1-C5 | `ab9e10d9` | 1,150 | 44,565,696 | 63 | 12 (+3) | 0 | yes | `5c7adf0d` | 118 | 1,538,010 | 4 | 1 | no (C2, C3) |
| c1-C6 | `48837206` | 1,220 | 46,436,808 | 71 | 8 (+1) | 0 | yes | `bb8f026b` | 90 | 1,835,748 | 5 | 1 | no (C2, C3) |
| c1-C7 | `baf9c521` | 1,162 | 47,018,450 | 74 | 15 (+4) | 0 | yes | `9195a123` | 28 | 715,787 | 0 | 1 | no (C2, C3) |
| c1-C8 | `685aa5f5` | 1,166 | 46,699,700 | 73 | 11 (+2) | 0 | yes | `10401c1e` | 152 | 841,960 | 6 | 1 | no (C2, C3) |
| c1-C9 | `fb347890` | 1,105 | 44,348,853 | 62 | 3 (+0) | 0 | no (C6) | `ce11faaf` | 28 | 275,303 | 0 | 1 | no (C2, C3) |
| c2-W0 | `592be54d` | 1,477 | 63,196,626 | 160 | 46 (+21) | 0 | yes | `a9d640a2` | 324 | 1,690,871 | 4 | 0 | yes |
| c2-C1 | `daff9efd` | 1,408 | 54,599,704 | 109 | 19 (+14) | 0 | yes | `2e2c3c23` | 66 | 891,779 | 2 | 0 | yes |
| c2-C2 | `563fa21d` | 1,262 | 43,141,212 | 72 | 16 (+2) | 0 | yes | `6c7b6c58` | 194 | 2,129,416 | 2 | 0 | yes |
| c2-C3 | `7f986c21` | 1,276 | 41,734,352 | 66 | 8 (+1) | 0 | yes | `e732ccbb` | 87 | 1,383,225 | 2 | 0 | yes |
| c2-C4 | `f0ae7e58` | 1,414 | 49,179,424 | 79 | 25 (+0) | 0 | yes | `d2c6ebc4` | 171 | 2,212,522 | 2 | 0 | yes |
| c2-C5 | none (3 attempts) | 782 | 31,432,340 | 63 | – | 31 | no-candidate | `f688c2ff` | 203 | 2,849,987 | 2 | 0 | yes |
| c2-C6 | `ff9a8c7f` | 1,436 | 48,712,184 | 76 | 15 (+3) | 0 | yes | `b28ab96a` | 90 | 573,327 | 2 | 0 | yes |
| c2-C7 | `391330c6` | 1,210 | 41,394,943 | 60 | 8 (+4) | 0 | yes | `e6bba949` | 38 | 510,869 | 0 | 0 | yes |
| c2-C8 | `ac35b224` | 1,294 | 44,235,638 | 64 | 10 (+0) | 0 | yes | `1727e116` | 132 | 1,211,813 | 2 | 0 | yes |
| c2-C9 | `fd4a8d72` | 1,266 | 44,837,593 | 60 | 1 (+0) | 0 | no (C6) | `c72489be` | 27 | 531,101 | 0 | 0 | yes |
| c3-W0 | none (2 attempts) | 1,112 | 56,544,323 | 152 | – | 30 | no-candidate | `9f422ac9` | 522 | 1,707,450 | 10 | 0 | yes |
| c3-C1 | none | 2,670 | 96,526,601 | 241 | – | 30 | no-candidate | `c0bbe606` | 109 | 955,066 | 3 | 0 | yes |
| c3-C2 | none | 1,266 | 65,032,467 | 186 | – | 31 | no-candidate | `0f9de466` | 254 | 2,461,937 | 7 | 0 | yes |
| c3-C3 | none | 1,284 | 52,606,248 | 122 | – | 31 | no-candidate | `37115c2a` | 80 | 932,536 | 3 | 0 | yes |
| c3-C4 | none | 897 | 74,536,138 | 256 | – | 31 | no-candidate | `e9f541f3` | 107 | 1,288,184 | 3 | 0 | yes |
| c3-C5 | none | 1,120 | 64,232,502 | 176 | – | 31 | no-candidate | `e332fb3c` | 162 | 2,072,227 | 4 | 0 | yes |
| c3-C6 | none | 1,074 | 66,338,617 | 185 | – | 31 | no-candidate | `f6084729` | 94 | 1,902,441 | 3 | 0 | yes |
| c3-C7 | none | 1,045 | 66,482,893 | 219 | – | 31 | no-candidate | `f262935c` | 52 | 323,926 | 0 | 0 | yes |
| c3-C8 | none | 1,018 | 63,712,141 | 187 | – | 31 | no-candidate | `2f37e8f5` | 104 | 667,109 | 2 | 0 | yes NQ |
| c3-C9 | none | 1,115 | 67,792,907 | 194 | – | 31 | no-candidate NQ | `73f3c3cc` | 33 | 378,620 | 0 | 0 | yes U |

## Descriptive sections (not part of the reading)

These sections separate "M missed obligations" from "M's run did not complete", as the coordinator asked after an Architect review requested by the owner. They do not change the reading above.

### Domain misses on deliverable candidates only

| | Candidates | Failed probe families | Failed behavior families | Steps with an invariant failure |
| --- | --- | --- | --- | --- |
| M chain 1 | 10 of 10 | 0 | 0 | 0 |
| M chain 2 | 9 of 10 | 0 | 0 | 0 |
| M chain 3 | 0 of 10 | – | – | – |
| B chain 1 | 10 of 10 | 10 (`P-AUDIT-ACCEPTED` at every step) | B-BILLING, B-BOOKINGS, B-EQUIPMENT, B-SAFETY, B-TRAINING at every step; D-C2 to D-C6 and D-C8 at 2–8 steps | 0 |
| B chain 2 | 10 of 10 | 0 | 0 | 0 |
| B chain 3 | 10 of 10 | 0 | 0 | 0 |

- **B's chain-1 defect.** `addMachine` recorded the machine's family as the audit target instead of the machine. B never noticed or fixed it, so it failed the same probe family and the dependent behavior families at all ten steps. Chains 2 and 3 started from fresh threads and did not have it.
- **Neither arm regressed** a family it had passed at its previous deliverable step.
- **Change discipline on candidates.** Every C3 census and every C7 census was empty in both arms, and no C7 file lay outside the billing area. Both of M's C9 results failed the strict gate on test-only edits: 3 test files in chain 1, one test file with 14 added lines in chain 2. All three of B's C9 results changed nothing.
- **Review measures.** R5 classified M's changes as 209 required, 92 optional and 1 repair. B's were 71 required, 15 optional and 2 repair. Neither arm had an unjustified change. Comparable lines changed: M 17,701 over 19 candidates (302 files plus 88 registry files), B 6,639 over 30 (88 files). R3b test topics were complete in all 19 M candidates and in 24 of 30 B candidates. No review reported an invented requirement (R7).

### Technical outcomes by cause

| Cause | Arm | Steps (attempts) | Class |
| --- | --- | --- | --- |
| Provider capacity: "Selected model is at capacity" (`serverOverloaded`, `willRetry` false) during Assessment, before any candidate | M | c2-C5 (3) | infrastructure; two repeats, then `no-candidate` by the coordinator's decision |
| Preflight engine timeout: Docker Desktop's idle shutdown | M | c3-W0 (1) | infrastructure; repeat |
| Product limit: the per-message clock (600 s of provider time) on an actively coordinating Root during Implement | M | c3-W0, C2, C3, C5, C6, C7, C9 (7) | product |
| Product limit: message budget (256) used up in a correction pair; the Workflow starts a pair it cannot finish and ends with "invalid Run request" instead of a budget stop | M | c3-C1 (1) | product; first native observation of Verify reject → fresh Implement |
| Product limit: message allowance used up by hierarchical question relay while siblings waited on one unbuilt shared Package | M | c3-C4 (1) | product |
| Product defect: questions to a handed-off Manager stay open, and its non-action triggers the Run-wide no-progress stop (the child had answered with `ask` instead of `reply`) | M | c3-C8 (1) | product |
| Model blockers | M, B | none | – |
| Deadline hits (5,400 s) | M, B | none | – |

- **M's Apply completion rate:** 19 of 33 attempts published (chain 1 10 of 10, chain 2 9 of 12, chain 3 0 of 11). Only 4 attempts failed on infrastructure; the other 10 failures are the product's own.
- **B:** 30 of 30 sessions completed, with no repeat and no deadline hit.
- **Root's time.** Step 4 split each Root turn into host-served and provider time from its tool-call pairs. In the seven per-message-clock failures, Root's open Implement turn reached 594–599 s of provider time within 678–924 s of wall time. In M-c3-C1 Root finished its Implement turn at 527 s provider over 1,029 s wall.

### The coordinator's C5 decision

M-c2-C5 failed three times on the same provider overload. Step 4's capacity probe succeeded just before the third attempt, which failed 6 minutes later. C4 allows two repeats and stops the series after a third infrastructure failure. The coordinator then decided the continuation: `no-candidate` for the step, with all 31 probe families missed and the step not conforming, and no fourth attempt. The coordinator chose the continuation least favorable to M on purpose, so that this decision could not bias the result toward Markitect.

The series resumed after at least two hours and two successful uncounted probes 30 minutes apart (09:51 and 10:22). Each probe was one short request with the same model and login, outside the study. The probe does not reproduce M's load of about 21 parallel sessions, so the overload may depend on that load; B's single session and the probes got through. One transient overload of one Manager ends M's whole Apply, with no retry.

### The fresh-start effect

After a step without a candidate, the input rule starts every following M step as W0 does: the neutral input plus the step's snapshot, so the hierarchy rebuilds the whole system. B instead continues its own last result.

In chain 3, M-c3-W0 failed, so all ten M steps were fresh builds of about 20 Packages. Each hit a product limit; seven of them hit the per-message clock in Root's coordinating turn. A step's failure therefore repeats down the chain for M, while it does not for B.

The coordinator decided before chain 3 continued that it would run exactly as predeclared through C9. There was to be no stop, no product change and no new limit mid-series, because changing course would have changed the evaluation after seeing data.

### Cost analysis

| Comparison | Pairs | M / B time | M / B tokens | M / B messages |
| --- | --- | --- | --- | --- |
| Chain 1 | 10 | 10.2× | 41.2× | 805 / 10 |
| Chain 2 | 10 | 9.6× | 33.1× | 809 / 10 |
| Chain 3 | 10 | 8.3× | 53.1× | 1,918 / 10 |
| All, quiet pairs only | 28 | 9.1× | 39.8× | 3,151 / 28 |

- **By Run type** (all 33 M attempts, 1,627.8 M tokens): Assessment 22 %, Implement 59 %, Verify 19 %.
- **By Manager level:** Root 14 %, level 1 50 %, level 2 36 %.
- **Dominant tool calls:** 9,020 shell commands, 4,012 awaits, 1,840 checks, 1,685 judgments, 1,673 delegations, 1,561 reviews, 1,528 asks and 1,471 replies.
- **Fixed versus scaling cost.** M's cost per published change step is mostly fixed:
  - Assessment (9.4–14.5 M tokens) and Verify (13.9–16.5 M) run the whole hierarchy at every step.
  - Implement of a change step took 13.0–31.0 M.
  - A change step therefore cost 41–55 M tokens. C9, with nothing to change, still cost 44 M in both chains, against B's 0.3–0.5 M.
  - B's cost follows the change: 0.3–2.9 M tokens per change step.
  - The failed fresh builds of chain 3 cost 52.6–96.5 M tokens each, with nothing published.

## Deviations and incidents, in sequence

1. **Blinding observation (chain 2).** M's README names `.markitect/model/`, so a reviewer can tell the arm from the text, beyond the layout limit already named in C9. No score depends on it.
2. **Ledger builder amendment A1, after chain 2's data.** The frozen `build-ledger-pc.py` required an M step's state packet (`previousPacket`) to be the last deliverable step's packet. Step 4's controller takes state from the previous step's latest attempt after a no-candidate step, and refuses any other. So the frozen check stopped the build at M-c2-C6, whose candidate input `f0ae7e58` was correct.
   - The coordinator approved a separately reported bookkeeping amendment, `build-ledger-pc-a1.py` (sha256 `0890ece1…`). It accepts either packet and still requires `inputFrom` to be the chain's last deliverable candidate. The frozen file stays unchanged (`74045912…`).
   - The chains 1–2 build with A1 gave rows identical to a provisional in-memory build that only relaxed that check.
   - The official chains 1–3 ledger, `ledger-c3-a1.json`, was built with A1.
3. **Quota misreading.** During chain 2 the operator reported 57 % from the maximum over transcripts spanning two weekly windows; the window had been reset early. The correct reading was given to the coordinator after chain 2 and changed no decision.
4. **Capacity failures and the stop at M-c2-C5.** See the C5 decision above. Attempt 02 started 2.5 minutes after the first fault, before the coordinator's waiting advice arrived. All later repeats waited as required.
5. **Docker Desktop idle shutdown (M-c3-W0 attempt 01).** The preflight `docker image inspect` of the pinned image timed out after the engine had idled since the previous Apply. Nothing ran, so this was a repeat.
   - Step 4 offered launch-helper changes. The operator declined them during the counted runs (C8) and recorded them as findings.
   - Instead, the operator's own pre-go check, not study infrastructure, required the pinned image to answer `docker image inspect` within 20 s.
6. **Product failures in chain 3** are classed in the table above. The coordinator confirmed each `no-candidate` with its cause wording.
7. **Parallel product-fix sessions during chain 3.**
   - The coordinator started three fix sessions (provider retry, idle-clock accounting, change minimality), whose targeted Go tests ran on the host. They made B-c3-C8 (16:28:52, two `go.exe`) and M-c3-C9 (16:34–16:36, `bootstrap.test.exe`, `go.exe` and a go-build-cache `markitect.exe`) not quiet.
   - After B-c3-C8 the operator held M-c3-C9 until the coordinator stopped all host Go work. Two of the stop messages were still queued when M-c3-C9 started on a fresh quiet check.
   - Both runs keep their quality result. Their cost is reported but left out of the quiet-only comparison.
8. **Constant background load.** Two orphaned `python3.exe -` processes from stale Claude shell commands (PIDs 2372 and 21276, started 25 October 06:49 and 09:05) each used about 0.75 of a CPU core throughout Phase C, for both arms alike. The monitor does not count them, because they are neither test suites nor Markitect processes. They affect run times, not quality. The coordinator ended them after chain 3.
9. **Review record.** One R5 path in M-c2-C8's review came back malformed. It was saved verbatim; R5 is not scored, and the citation check covers criticals only.
10. **Contamination audit script written after the series.** Addendum C froze no Phase C audit script. `contamination-audit-pc.py` reuses Phase A's audit and patterns unchanged and only selects the Phase C transcripts.

## Limits

- C9's limits apply.
- N = 3, and chain 3 is a single failure regime: once M-c3-W0 failed, the input rule made every later M step a fresh build.
- The C5 continuation and the `repeat` and `no-candidate` decisions were the operator's, under C4. The coordinator confirmed each, and every one is recorded with its evidence.
- Provider capacity is outside both arms' control. Only M's load of about 21 parallel sessions met it.
- The machine carried a constant background load from two orphaned processes, and three runs are not quiet or unrecorded.
- The blind review cannot hide M's layout, and M's README names its model directory.
- The analyst was not blinded; the reading is computed by [`analyze.py`](../../tools/comparison/analyze.py) `--phase C` from the ledger.

## Retained evidence

Root `C:\Users\Consiliari\Documents\Codex\Markitect-V1-Comparison\e11e6cff883b\phase-c\`:

- `ledger-c3-a1.json` (sha256 `015ffa50…`), built by `build-ledger-pc-a1.py`. The chains 1–2 build is `ledger-c2-a1.json`.
- `analysis-phase-c.json` (sha256 `972dd45f…`).
- `contamination-audit-pc.json` with `contamination-audit-pc.py`.
- `runs\B-c*` for arm B, `m-runs\M-c*` for M's bound attempts with every `decision` file, and `costs\M-c*` for the per-attempt cost breakdowns.
- `assess\`, `reviews\` and `quiet\`: the assessments, the reviews with citation checks, and the quietness samples.
- `FREEZE.sha256`, `study.env` and the frozen scripts.
- The operator log `..\LEDGER.md`.

M evidence stays under step 4's `Markitect-V1-Requalification\4d23f7ae\packets\m-*\attempt-01\`. Step 4's per-Apply records are `PHASEC-armM-counted-run-log.md` and `PHASEC-armM-root-split.md` in the coordinator's notes folder.
