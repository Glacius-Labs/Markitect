# Old-versus-new Makerspace gate on be7813a6 and F_cmp2 14310a70 — 27 October 2026

This record covers the old-versus-new comparison gate of the [canonical model track](../implementation-plan.md#canonical-model-track-owner-approval-26-october), step W6. Markitect built the Makerspace project at W0, C1 and C2 from two models:
- **old**, the frozen Phase C model;
- **new**, the remodeled `tools/comparison/makerspace-v2` ([PR #764](https://github.com/Glacius-Labs/Markitect/pull/764), [PR #768](https://github.com/Glacius-Labs/Markitect/pull/768)), with the [owner-approved architecture](../concepts/sources/model-track-w4-w6-decisions-20261027.md).

The coordinator dispatched it under the [standing evidence authorization](../concepts/sources/v1-source-implementation-approval-20261023.md#subsequent-standing-evidence-authorization). It is a pilot: uncounted, descriptive, one run per cell. Following the [owner's decision of 27 October](../concepts/sources/owner-night-decisions-20261027.md#3-when-the-comparison-runs), it ran at once, without the 85 % quota launch gate.

## Result

| Step | old (Phase C model) | new (makerspace-v2) |
| --- | --- | --- |
| W0 | **Published** after one correction pair. Oracle 54/54, deliverable | **Published** in attempt 02 on F_cmp2. Oracle 54/54, deliverable; import graph conforming. Attempt 01 hit a publication defect ([W0, new](#w0-new-the-publication-defect)) |
| C1 | **Published**. Oracle 55/55, deliverable | **Published**. Oracle 55/55, deliverable; import graph conforming |
| C2 | **Published**. Oracle 57/57, deliverable | **Not published:** Root ended Assessment with a **model blocker** in the makerspace-v2 model ([C2, new](#c2-new-model-blocker)). Attempt 02 on makerspace-v2.1, which addresses it, **published**: oracle 57/57, deliverable, import graph conforming ([addendum](#addendum-new-c2-on-makerspace-v21)) |

**Reading.**
- Both models produced deliverable programs at every step they published. The hidden Makerspace oracle passed every case.
- The new model ran with 12 Managers instead of 21. With one run per cell, the cost figures below are observations, not effects.
  - **W0 and C1, published attempts only:** new used 80 messages against old's 201, 107.5 M tokens against 189.8 M, and 3,578 s against 4,689 s. Old W0's extra correction pair accounts for part of the difference.
  - **With new W0 attempt 01 counted:** new used 125 messages, 168.0 M tokens and 5,396 s, which is more time than old.
  - **C1, the like-for-like pair** (no correction in either arm): 39 against 68 messages, 32.1 M against 47.1 M tokens, and 1,359 s against 1,433 s.
  - **Run-to-run variation:** new W0's two attempts had the same input and model. Both closed three Runs, at 60.4 M and 75.4 M tokens and 1,818 s and 2,219 s.
- The new arm's measured candidates follow its own DependencyRules. Only `protocol` imports other Packages, and no domain Package imports another.
- At C2, the Managers reported a contradiction in the new model's equipment Interface. The same definitions are in all three steps, but only C2's Assessment flagged them.
- The gate found one product defect, a publication guard that depended on read order. It is fixed.

No Apply left residue or a credential hit, and the runtime login was unchanged in all seven of the original series. Usage rose from 75 % to 94 % before the addendum, and no quota stop occurred. What this pilot cannot show is under [Limits](#limits).

## Binding

| Item | Value |
| --- | --- |
| Source, old arm and new W0 attempt 01 | `be7813a67e683f0a9f8afbfb981fd0ec4b25a684`, tree `adf2d2eb89e9d5f6e702ea4284993b1cc292be58`: Main after [PR #767](https://github.com/Glacius-Labs/Markitect/pull/767) (the DEC-073 gap fix). Fresh clean clone |
| Binary and images, be7813a6 | Built from 02:35:41Z with `-trimpath` and go1.27.1. SHA-256 `e67b5d0c4306015464ecd9ff6bd77934ca6b4e3f794c079e3d425727f4b8e875`, productBuild `0.14.1+be7813a6…`, `vcs.modified=false`. Base `sha256:7daf960fb644931acd53a6c91c13511364a3edecd50b361a8de4c3f7b2508e2a`; Go `sha256:7bceea4480149cf43e53ede88dfced82a35f07bfd20e1538d4f82b54249c700e` |
| Source F_cmp2, new arm from W0 attempt 02 | `14310a7092f9fe546ea279d527380d19517fd12a`, tree `063d85b16b33ec15eced5a01faa14b432c72b0fb`, branch `fix/fcmp2-guard-on-be7813a6`. That is be7813a6 plus only #776's fix, cherry-picked. It touches `internal/adapters/gitrepo` only; Linux CI 40451668337 was green. Every pinned material tree and `packaging/` equal be7813a6's |
| Binary and images, F_cmp2 | Built from 05:04:36Z. SHA-256 `ebff3a908cd42ac1dfcd85ca19fc4f8e6f833e872d674fdece31a2216ede059e`, productBuild `0.14.1+14310a70…`, `vcs.modified=false`. Base `sha256:d626ac7aea642600e203dbcc483b3da03da5268450abd51cd3f47c69296529f6`; Go `sha256:3e334a4b1a22545a61129fbd0605d7c4c88f712757daab8a7c725683b3369cc1`. In both builds, `runtime-base.env` and `runtime-go.env` hold `codex-cli 0.162.0`, `codexSchema` `816cef0e…` and the build's own `agentd` |
| Provider | Codex 0.162.0, `gpt-6-sol`, effort `high`, normal account login. The dedicated runtime login was only hashed, and it was unchanged before and after every Apply |
| Run configuration | The source's example, format 4: `providerIdleTimeoutSeconds` 600, `maxMessages` 256, Apply deadline 5,400 s, 32 Managers |
| Old arm material | `tools/comparison/phase-c/model/<step>/model`, unchanged, with execution file `phase-c/execution/w0.yaml`. Dependencies `testdata/core` and `testdata/library/vocabulary`. The manifest and lock were rebound in the run copy only: `markitect-core` 1.1.0 → 1.2.0 (`sha256:b338a1d6…`, the same Library as in the new arm's lock) and `software-basics` 0.2.0 → 0.3.0 (`sha256:85a70962…`). 21 Managers |
| New arm material | `tools/comparison/makerspace-v2/model` (W0), `c1/model` and `c2/model`, with their own `model.yaml`, `model-lock.yaml` and `execution.yaml`. Dependencies: the six vendored Libraries under `makerspace-v2/libraries/`. 12 Managers |
| Pins | Model digests per arm and step: old `1a0f730c…`, `cad8b7c3…` and `2db0e9e9…`; new `ffa2f01b…`, `5054b463…` and `ed20df5b…`. For every input the controller re-checked: the git tree ids of all material, that the working tree under it was clean, the digest and the Manager count. Every input matched |
| Chain rule | Phase C's C4, per arm. A step continues the arm's last published candidate, and only `.markitect/model/` is replaced; the manifest, lock and execution file must equal the arm's. The state comes from the arm's previous step. Old ran in chain 1 and new in chain 2 |
| Controller | `controller-gate`, a copy of the fix-plan controller. It adds per-arm material, pins, dependency arguments and Manager sets, and a chain guard per arm. It has 66 offline self-tests. A rehearsal against a stand-in engine covered both arms' inputs and packets, and an independent review returned ready with notes, all applied, before any run |
| Gates before each launch | The `NATIVE-RUN.lock` reservation (one for the series) and a free controller lock. Usage read and recorded, refusing a launch only at a reported limit or when unreadable; the owner waived the 85 % gate. Docker readiness on the pinned image. Quietness recorded only |
| Oracle | `tools/comparison/assess_candidate.py` with profile `w0`, `c1` or `c2`: the hidden Makerspace oracle from the source, on its Go image, without network, against the step's start bundle, plus the candidate's own build and tests. The Evaluation session confirmed that the oracle fits makerspace-v2 at W0, C1 and C2, from a mechanical comparison of the requirement and rendered specification texts. It does not test the new architecture (requirements §16). A smoke before W0, on the Phase C chain-1 W0 candidate, passed 54/54 |
| Import graph | For the new arm's candidates: every module-internal Go import is an edge between the Packages owning the two folders under `execution.yaml`. It is checked against the model's `Architecture.DependencyRule`s; test-only edges are reported apart. It checks import edges, not which exposed types are used |

## Applies

Seconds are Apply process time. Messages are the total and the split per Run.

The source is be7813a6 for the old arm and new W0 attempt 01, and F_cmp2 for the other new-arm Applies.

| Apply (packet) | Apply ID | Status | s | Messages | Paths | Final commit / tree | Tokens | Retries | P2 |
| --- | --- | --- | ---: | --- | ---: | --- | ---: | ---: | --- |
| old W0 (`m-old-w0`) | `3e99b9e8cecb257dc817c10aa2be9181` | technical pass, passed | 3,256 | 133 (21/49/21/21/21) | 64 | `ae1f98ed` / `bba41ceb` | 142,674,029 | 0 | exercised |
| new W0, attempt 01 (`m-new-w0`, be7813a6) | `bd36a0d9c319e19bb979c0199fe4af74` | technical fail, not completed | 1,818 | 45 (12/21/12) | 0 | verified `537a1571` / `32093b73`, not published | 60,446,081 | 0 | not exercised |
| old C1 (`m-old-c1`) | `6a4da422afbcb6f4e83d034cf44c4eb9` | technical pass, passed | 1,433 | 68 (21/26/21) | 19 | `6177e7db` / `660a04e8` | 47,094,033 | 0 | not exercised |
| old C2 (`m-old-c2`) | `ae5c27a9cbc84aff2634ff3f3c6e1c60` | technical pass, passed | 1,427 | 66 (21/24/21) | 14 | `d7f0d738` / `0ee74cf7` | 43,796,617 | 0 | not exercised |
| new W0, attempt 02 (`m-new-w0-02`, F_cmp2) | `245863d72a4f63812870428ba5e8d470` | technical pass, passed | 2,219 | 41 (12/17/12) | 41 | `f7180418` / `a5beb61b` | 75,419,099 | 0 | not exercised |
| new C1 (`m-new-c1`, F_cmp2) | `0963aeb9b6cdc15946e3c2fa8a0c3df1` | technical pass, passed | 1,359 | 39 (12/15/12) | 11 | `c19a5c04` / `49873842` | 32,083,025 | 0 | not exercised |
| new C2 (`m-new-c2`, F_cmp2) | `8a1aafa228b0c0f51f6729a24c343fd3` | failed: model blocker in Assessment | 279 | 12 (12) | 0 | — | 9,578,873 | 0 | not exercised |

The seven Applies of the original series used 11,791 s and 411,091,757 tokens. The [addendum](#addendum-new-c2-on-makerspace-v21)'s eighth used 1,647 s and 45,970,909 tokens more.

| Apply | Root turns | Root's peak provider turn | Root's longest turn | Root's `ask` / `reply` calls | Oracle | Import graph | Usage before → after |
| --- | ---: | ---: | ---: | --- | --- | --- | --- |
| old W0 | 5 | 39.3 s | 1,343 s | 22 / 50 | 54/54, deliverable | — | 75 → 81 % |
| new W0, attempt 01 | 3 | 18.8 s | 1,181 s | 0 / 0 | 54/54, deliverable (verified, not published) | conforming, 26 files | 81 → 84 % |
| old C1 | 3 | 32.1 s | 600 s | 2 / 5 | 55/55, deliverable | — | 84 → 86 % |
| old C2 | 3 | 21.1 s | 616 s | 1 / 4 | 57/57, deliverable | — | 86 → 89 % |
| new W0, attempt 02 | 3 | 47.3 s | 1,464 s | 1 / 0 | 54/54, deliverable | conforming, 28 files | 89 → 92 % |
| new C1 | 3 | 12.3 s | 717 s | 1 / 0 | 55/55, deliverable | conforming, 28 files | 92 → 93 % |
| new C2 | 1 | 16.5 s | 165 s | 0 / 0 | — | — | 93 → 94 % |

Root's peak provider turn is the longest stretch of provider work without a served tool call, the quantity the idle limit bounds. The `ask` and `reply` columns count only Root's own tool calls, not the questions of the whole hierarchy. The per-Run totals are in each step's `control-report.json`. Every oracle run had 0 failed families and 0 errors, and its own build and tests passed. The container and volume census was unchanged around every oracle run.

### W0, new: the publication defect

Attempt 01 closed all three Runs: Assessment approved, Implement submitted, Verify approved. Publication then refused with `guarded write precondition failed: source changed during acquisition: publication source binding drift`. The input repository was unchanged and clean afterwards. The candidate changed only `realization/` and added 11 registry shards.

The coordinator traced the cause:
- The publication guard hashed model entries in read order.
- Git tree order and the disk walk differ when a file like `billing-dependencies.yaml` sits beside a folder `billing/`, which happens only in makerspace-v2.
- The Phase C model has no such pair, so the old arm hashed identically in both orders.

#776 binds the entries in canonical order. Attempt 02 ran on F_cmp2 under the coordinator's dispatch and published.

Attempt 01's verified candidate `537a1571` was assessed as well, descriptively: verified but not published (publication defect). It passed the oracle 54/54, and its import graph conforms.

### C2, new: model blocker

Assessment ended at Root with a model blocker after 279 s, and nothing was published ([DEC-063](../concepts/register.md#dec-063-the-model-is-authoritative-defects-escalate)). The blocker is on `makerspace/equipment/equipment-interface`:
- The Interface exposes the types `family`, `machine` and `maintenance-window`, and Packages outside equipment may use only exposed types.
- But `makerspace/bookings/reservation` declares `rateCentsPerMinute` with field type `Equipment.rate-cents-per-minute`.
- In the Managers' reading, the model therefore both requires and forbids this cross-Package type dependency, and choosing either in code would invent a model decision.

Root, `makerspace` and `equipment` each reported it, and `makerspace`'s review confirmed it. The reading is arguable: the Interface's body also says that other modules ask for a machine's family and current rate.

The equipment Interface, the reservation and the `rate-cents-per-minute` definition have identical git blob ids in the makerspace-v2 W0, C1 and C2 models, and the W0 and C1 Assessments approved. So C2 did not introduce the issue, and only C2's Assessment flagged it. With one run per step, that may be detection variation as much as a property of C2. It is a finding for the model's owners: make the Interface's exposed types and the reservation's field type agree.

The controller labels the stop failure class "product", its default for a failed Apply. This record classes it a model blocker. It is neither a product failure nor a quota stop, and it was not repeated.

### Structure of the new arm's code

The three new-arm candidates measured show the same import shape: W0 attempt 01 (verified, not published), W0 attempt 02 and C1.
- `protocol` (with `cmd/makerspace`) imports the nine domain Packages;
- no domain Package imports another.

So no DependencyRule edge was exercised beyond protocol's. The check sees import edges only: shared types and duplicated structures within one Package are not visible to it. The old model declares no DependencyRules, so the old arm has no import-graph counterpart.

## Order and deviations

- **Order.** Planned: W0 old → new, C1 new → old, C2 old → new. Actual: old W0, new W0 (attempt 01), old C1, old C2, new W0 (attempt 02), new C1, new C2.
  - After attempt 01 the coordinator held the new arm until the defect was diagnosed, and ran the old arm's C1 and C2 first.
- **Sources differ between the arms** by exactly #776's publication-guard fix. The old arm ran on be7813a6, and new from W0 attempt 02 on F_cmp2.
  - The fix makes the guard bind model entries in canonical order for every project, and changes its label from `gitrepo-capture/2` to `/3`.
  - According to the coordinator's diagnosis, it does not change the outcome for models without a file beside a folder of the same name. The Phase C models have no such pair.
- **Repeat of new W0.** The coordinator dispatched attempt 02 because attempt 01 failed on a product defect that was then fixed. Both attempts are recorded.
- **Quota.**
  - The 85 % launch gate was waived for this series by the owner ([owner decisions, §3](../concepts/sources/owner-night-decisions-20261027.md#3-when-the-comparison-runs)), and usage reached 94 %.
  - The agreed rule went unused: a quota stop would have been classed "quota" and repeated once after the 2 November reset.
- **Quietness**, recorded only and sampled at launch. At the old C2 launch, `gitrepo.test.exe` and two `go.exe` processes ran, probably the coordinator's diagnosis of the defect, which may also have run during that Apply. Every other launch was quiet.
- **Added before W0:** the oracle smoke, at the review's request.
- **Failure class of new C2.** The controller's "product" is corrected to "model blocker" in this record, as the coordinator agreed.

## Limits

- **A pilot, not a study.** One run per cell, uncounted, with no predeclared reading. The differences in cost and behavior are observations. New W0's two attempts show run-to-run variation of about a quarter in tokens and time.
- **One source per arm, one provider.** Old ran on be7813a6 and new on F_cmp2, which differ only by #776. Both used Codex `gpt-6-sol`, effort `high`.
- **Old model against new model, both through Markitect.** There is no single-agent baseline, so the record says nothing about Markitect against one agent.
- **The oracle's limits, as confirmed by the Evaluation session:**
  - a mechanical comparison of the requirement and rendered specification texts, not a semantic review;
  - no test of the new architecture (requirements §16);
  - valid for W0, C1 and C2 only;
  - confirmed for the v2.0 texts; the addendum's v2.1 changes only the section-16 Interface line, and the confirmation was not repeated.
- **Quality** is measured by the hidden oracle and the candidates' own build and tests only. There was no rubric review of the code. The import graph covers module-internal import edges of the new arm only.

## Addendum: new C2 on makerspace-v2.1

After the [owner's instruction of 27 October](../concepts/sources/owner-morning-decisions-20261027.md#1-the-last-pilot-cell), the coordinator dispatched one more cell: new C2, attempt 02, on makerspace-v2.1. It ran on Codex, because no Claude adapter exists yet, with the remaining quota and the gate's limits, gates and abort rules. A quota stop would have been a failed attempt classed "quota", not repeated.

- **Model.** [PR #779](https://github.com/Glacius-Labs/Markitect/pull/779) (makerspace-v2.1) changes only `equipment/equipment-interface.yaml`, in W0 and every change step. It adds `rate-cents-per-minute` to the Interface's types and names "the machine rate type" in its body. The rendered requirement texts change only in that Interface line of section 16, which the oracle does not test. Manifest, lock, execution file, Libraries, oracle and assessor are unchanged.
- **Binding.**
  - F_cmp2's binary and images, as for the rest of the new arm.
  - The C2 v2.1 model from a clean clone at Main `00c687b6`: tree `5efef6d9`, digest `sha256:b61d791a5fcb5c6fb78fed0f43cc9d9e4549dac929f9b49e4c45d38dc9863e84`, 12 Managers.
  - The digest was measured with F_cmp2's `markitect check` on a scratch commit, and the same method reproduced the v2.0 pin `ed20df5b`. The controller then confirmed the input's digest equals the pin.
  - The controller gained an arm entry `new21`, with chain 2, its own material clone and pins, and passes its 66 self-tests.
- **Chain.**
  - The step continued new C1's published candidate (`c19a5c04`), with new C1's state, under rule C4.
  - The swap from C1's v2.0 model to C2's v2.1 model therefore carries the Interface fix along with C2's own change.
  - Old C2, and new W0 and C1, stay as recorded on v2.0.

| Item | Value |
| --- | --- |
| Apply (`m-new-c2-02`) | `07dd328612c62bce56ee081dc41fe660`. Technical pass, passed. Runs: assessment approve, implement submitted, verify approve, with no correction |
| Cost | 1,647 s, 41 messages (12/17/12), 16 paths, 45,970,909 tokens, 0 retries, no residue or credential hit, login unchanged |
| Final commit / tree | `5a0ea178` / `4b472975` |
| Root | 3 turns, peak provider turn 25.7 s, longest turn 889 s, `ask` / `reply` 0 / 0 |
| Oracle at c2 | 57/57, deliverable, mechanical pass, 0 failed families, census unchanged |
| Import graph | Conforming: 28 Go files, 10 internal imports, all from `protocol`. C2's permission `bookings → safety` went unused |
| Usage | 95 % → 97 %. The launch was quiet |

**Reading.**
- On v2.1 the new model's C2 published, and the candidate passed every hidden case. Its Assessment did not raise the contradiction again. With one run per attempt, that does not show the change caused the pass; the v2.0 stop may have been detection variation as much as a property of v2.0.
- **Against new C1:** 41 against 39 messages, but 46.0 M against 32.1 M tokens (+43 %) and 1,647 s against 1,359 s (+21 %), for a larger change (16 paths against 11).
- **Against old C2:** fewer messages (41 against 66), but more tokens (46.0 M against 43.8 M, about +5 %) and more time (1,647 s against 1,427 s, about +15 %). The W0 and C1 cost figures in the [Result](#result) do not carry over to C2.
- The approval does not show that v2.1 holds no other latent contradiction. It shows only that this Assessment raised none.
- **The oracle's fit** was confirmed by the Evaluation session for the v2.0 texts and was not repeated for v2.1. v2.1 changes only the section-16 Interface line, which the oracle does not test.

## Evidence

The packets, measures, quota and quiet records live on the host:
- `Documents\Codex\Markitect-V1-Requalification\gate\be7813a6\` (the old arm and new W0 attempt 01);
- `Documents\Codex\Markitect-V1-Requalification\gate\14310a70\` (the new arm on F_cmp2, including the addendum's `m-new-c2-02` and its material clone `material-00c687b6`).

Each step has `measures\<name>\summary.json`, `control-report.json`, `oracle\` and, for new, `import-graph.json`. The kit lives under `gate\controller-gate`, `gate\launch` and `gate\measures`.
