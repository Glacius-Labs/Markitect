# V1 comparison Phase B, arm M, on F′ 528ef346 — 25 October 2026

This record covers the native runs of arm M in Phase B of the [comparison protocol](v1-comparison-protocol-20261024.md) ([DEC-072](../concepts/register.md#dec-072-stable-v1-justified-by-data)), as amended by its section B8. The coordinator dispatched them under the [standing evidence authorization](../concepts/sources/v1-source-implementation-approval-20261023.md#subsequent-standing-evidence-authorization). Every run had exactly one attempt. The record holds arm M's raw outcomes only. The study's assessor judges the candidates, and the comparison with arm B belongs to the study's own results document.

## Result

- **Finding of F.** On F (`e11e6cff`), V1 could not process a Definition deletion. The structural preflight refused the T3 input before any Manager started ([F's refusal](#finding-of-f-a-definition-deletion-was-refused)). That moved arm M's Phase B to F′, as B8 records.
- **On F′:**
  - the same T3 input passes structural preparation, with the two stale links reported for realization;
  - all 26 counted Applies passed protocol C1, the technical pass, each with its scenario condition met;
  - chains 1–3 ran T1–T4, and chains 4 and 5 ran S1–S3 and then T1–T4;
  - strict no-change held in all 7 S3 and T4 runs;
  - every T3 run removed the stale registry links natively, and no published candidate keeps one.
- **Reviews:** each chain segment had an independent retained-result review. None found a problem.

## Binding

| Item | Value |
| --- | --- |
| Source F′ | `528ef346d3149002f53bf8eca1c54ea5613453ed`, tree `d813e5e537cbd71feaaa52ec6c5e8a4fd51af09b`: Main after [#701](https://github.com/Glacius-Labs/Markitect/pull/701) (stale registry links) with [#696](https://github.com/Glacius-Labs/Markitect/pull/696) (no-progress stop). Fresh clean clone; HEAD, tree and status were checked before and after every run |
| Product binary | Built from F′ with `go build -trimpath -buildvcs=true` and the pinned go1.27.1 toolchain. SHA-256 `818b0fa8e7c744fe15d9abd5fe3a0bc86520a1ba01401c7e497933bf28d0d1bd`, productBuild `0.14.1+528ef346…`. P0 built on the first attempt in 78.6 s |
| Images | Built from F′'s `packaging/runtime`. Base `sha256:5ed9b04f917e078970a5beb0a5e056e4116a1b8512c1344fa127f9d50d84a5a4`; Go `sha256:dd0ddd7089b52024e96cc35655c4d8590caabaa6b6327923717584408403736f`. `runtime.env` holds `codex=codex-cli 0.162.0`, `codexSchema=sha256:816cef0e…` and `agentd=` F′ |
| Provider | Codex App Server 0.162.0, `gpt-6-sol`, effort `high`, normal account login. The login file was only hashed, and the hash was equal before and after every run |
| Configuration and limits | As in [step 4](v1-requalification-20261025.md#binding): the exact example runtime profile and run configuration, an Apply deadline of 2,700 s, 128 shared messages, one attempt per run and the shared `NATIVE-RUN.lock` |
| Fixtures | T1–T3: the chain's previous published candidate plus one neutral commit that replaces `.markitect/model/` wholesale with the study's snapshot. M5 therefore deletes `members/set-member-status.yaml`, and `model.yaml` and `model-lock.yaml` stay unchanged. T4 and S3: the previous candidate unchanged. S1 and S2 are as in Phase A. State was carried from the previous step, except for S1 |
| Snapshots | Each manifest's SHA-256 was pinned (m3 `5f23c592…`, m4 `ba1b6ac0…`, m5 `7b93c978…`). Every copied model was re-verified against its manifest before the commit. The snapshots were confirmed independently: `tools/comparison/requirements.py materialize` from Main reproduces them exactly |
| Controller | The step-4 controller extended for Phase B: `requalify.py` SHA-256 `a826e56a37e2d28990ff78785e43c5b22f639f2abf53998950911674fefd0817`, 35 offline self-tests. It had one independent review and a delta review (READY) before any run. Each step needs its own snapshot profile and its exact predecessor, and every run gets post-run checks for product scratch, fixture worktrees and credentials |

## Finding of F: a Definition deletion was refused

Step 4's offline rehearsal built the T3 input from chain 1 (`4136b583` + M3 + M4 + M5). The result was commit `cdc879cb4f69eb07782d56cb3ce681874798ac55`, tree `b8934fede81e1faf84d6613cc77aa8d141da349f`.

- **On F:** `markitect check --revision HEAD` with the Library vocabulary and Core exited 1. It reported `registry.definition_missing` for `members/set-member-status`, from two entries in `.markitect/registry/members/registry.yaml`. Apply's structural preflight requires a passed registry, so the Apply would have stopped before any Run.
- **Why it is a product gap:** cleaning up those links is realization work for the Managers ([concept §6.3](../concepts/v1-concept.md#63-registry-maintenance-is-realization-project-intent-is-not)), and the owner change stays model-only. Every chain's S3 candidate links the Definition twice, so every M unit would have failed.
- **On F′:** the same tree passes. `check` exits 0, and the two links appear as `registry.stale_link` under `stages.structural.files.accounting.stale`.

## Runs

Order followed B8: chains 1–3 with T1–T4, then chains 4 and 5 with S1–S3 and T1–T4. Seconds are Apply process time. Messages are the sum and the Assessment/Implement/Verify split. Every Apply ran assessment → implement → verify with all six Managers in fresh native Sessions, and no run took the public correction branch (Verify Reject → fresh Implement; `correctionBranch` is false in all 26 reports). Every final candidate bundle reopened exactly. There was no residue, the product's temporary area was empty after every run, each fixture had one worktree, and the credential scan found nothing.

| Packet | Apply | C1 / scenario | s | Messages | Paths | Final commit / tree | Tokens | `package-result.json` |
| --- | --- | --- | ---: | --- | ---: | --- | ---: | --- |
| m-t1-c1 | `fe3eaffbcaf03551375e430c39fa315b` | pass / met | 718 | 20 (6/8/6) | 8 | `aa2280db` / `205c6cef` | 7,549,219 | `296e4ad6436e` |
| m-t2-c1 | `b4e44ab9827c91bc1397d04949a5aed1` | pass / met | 810 | 22 (6/10/6) | 9 | `0c81ea71` / `ddc6addb` | 9,157,134 | `eb57f8c4c7bd` |
| m-t3-c1 | `0267623e44d327ce59feb65fecf94b8d` | pass / met | 681 | 18 (6/6/6) | 10 | `551743f9` / `71c6adbe` | 7,293,294 | `4a6f555aef65` |
| m-t4-c1 | `28c906b4090ad2f2e03066338e138660` | pass / strict met | 634 | 18 (6/6/6) | 0 | `d439291c` / `71c6adbe` | 7,108,813 | `8c424fad87a3` |
| m-t1-c2 | `1875f660134b5bc1a050957abdf14a8d` | pass / met | 687 | 25 (8/11/6) | 10 | `1788e344` / `cd5ed418` | 8,704,941 | `6e0d8f9968e4` |
| m-t2-c2 | `45ff195a47894e6a13bc38ebb02aa506` | pass / met | 694 | 18 (6/6/6) | 13 | `dc601f9e` / `ea68ae63` | 8,352,941 | `71a3fccf16fa` |
| m-t3-c2 | `517858465e576cda4c43ffa24c6b21f4` | pass / met | 788 | 28 (8/11/9) | 11 | `5e21594d` / `f1882b5e` | 10,957,076 | `2be04de10700` |
| m-t4-c2 | `92461f0405925ed8ab0c3835cf280795` | pass / strict met | 671 | 18 (6/6/6) | 0 | `1e30decf` / `f1882b5e` | 7,196,705 | `0ceee646a708` |
| m-t1-c3 | `3369cee2a248721d9142e024ef811d34` | pass / met | 711 | 22 (6/10/6) | 7 | `3944c5dd` / `ecb01e72` | 7,719,409 | `fe88ddce0d2b` |
| m-t2-c3 | `ae7564120e4ea2f2ab22f3441db173e1` | pass / met | 600 | 24 (6/12/6) | 6 | `80d961bf` / `e5868dee` | 7,576,204 | `e7e9f7861605` |
| m-t3-c3 | `3c6fc37aec10220a3472440849016309` | pass / met | 625 | 18 (6/6/6) | 9 | `deca0d03` / `f48dbd60` | 7,698,348 | `c2605f0dc8ea` |
| m-t4-c3 | `32bffc74bcb69437e3cfc025a8f60680` | pass / strict met | 592 | 18 (6/6/6) | 0 | `deca0d03` / `f48dbd60` | 6,520,878 | `589e993f67f7` |
| m-s1-c4 | `df7f14c8d4fa257b69cd80e1a228d5bb` | pass / met | 999 | 22 (8/6/8) | 19 | `ef69dc6c` / `4f6eb071` | 8,705,447 | `f95487646bab` |
| m-s2-c4 | `d02a41da44e3255068fc52c383606e60` | pass / met | 590 | 19 (6/7/6) | 4 | `a7cc22c2` / `d115b7c1` | 6,921,935 | `f9dab52ca262` |
| m-s3-c4 | `7310aa0381651705fbd83d8a05fb3e27` | pass / strict met | 599 | 18 (6/6/6) | 0 | `a784444e` / `d115b7c1` | 6,468,316 | `0c3d5927d979` |
| m-t1-c4 | `70aa89772b3d6840548b6cc198fe5cd0` | pass / met | 647 | 19 (6/7/6) | 9 | `961924ed` / `5e851115` | 7,874,789 | `9edd08456619` |
| m-t2-c4 | `7b519cf3b4b6c6a65822cf11ad879e62` | pass / met | 636 | 24 (6/11/7) | 7 | `05d62743` / `f18271a4` | 9,645,018 | `30b16a141a51` |
| m-t3-c4 | `f24ae00b262a243e170b0c22d86b1ef1` | pass / met | 725 | 22 (8/7/7) | 10 | `12e497f2` / `b301f1cd` | 9,816,754 | `f92706fbccf9` |
| m-t4-c4 | `bda684cd412db836edd56db137c00e50` | pass / strict met | 587 | 18 (6/6/6) | 0 | `12e497f2` / `b301f1cd` | 7,259,353 | `2b7bfcd5f27a` |
| m-s1-c5 | `df1ec733e326c8dbb990629e97caf4fb` | pass / met | 694 | 21 (6/9/6) | 20 | `d8fbef6e` / `d493e22e` | 7,247,725 | `244cbb507997` |
| m-s2-c5 | `d8c49e72f85af52051c6892c4151015e` | pass / met | 605 | 19 (6/7/6) | 4 | `e14b5843` / `9e62a421` | 7,453,427 | `9cc02370f7c6` |
| m-s3-c5 | `9d96303efe47e538b8a53dc14e785b93` | pass / strict met | 570 | 18 (6/6/6) | 0 | `5e0c201b` / `9e62a421` | 6,275,587 | `b96a9950328e` |
| m-t1-c5 | `1650db123429bff6bc9d098d40e3d8e6` | pass / met | 638 | 22 (6/10/6) | 10 | `805df4da` / `cc90cd2f` | 7,367,997 | `c8647e3cc3e5` |
| m-t2-c5 | `b9aed1d4417bbe3717078985e239112e` | pass / met | 645 | 18 (6/6/6) | 8 | `b069ebab` / `e4a09f9b` | 7,855,181 | `438e7af4f50f` |
| m-t3-c5 | `533de3ea05cfec8dd6c5dd392dffc250` | pass / met | 628 | 18 (6/6/6) | 13 | `2e732a9d` / `61d85a0e` | 7,700,869 | `b2ed08167645` |
| m-t4-c5 | `49b69ac8c50250e0b36ff13704823e87` | pass / strict met | 587 | 18 (6/6/6) | 0 | `ba82345a` / `61d85a0e` | 7,145,575 | `2ca8bd288ebd` |

Totals: 17,362 s of Apply time, 525 messages and 203,572,935 provider-reported tokens. The longest Apply took 999 s of its 2,700 s deadline (m-s1-c4); the most messages in one Apply were 28 of 128 (m-t3-c2).

## What the runs show

- **Deletion through the hierarchy (T3, scenario B2).** In every T3 run, no stale link remains in the published candidate. For chains 1–3 the reviews traced the steps: Assessment named the links, then the owning Managers removed or realigned them in Implement (`members`, and in chain 2 also `lending` and `lending/reminders` in their own shards), and Root verified. No published candidate keeps a registry link to `members/set-member-status` or a `realizes` target the model lacks. The remaining mentions of `setMemberStatus` are tests expecting `invalidRequest`.
- **Model changes on carried realizations (T1, T2).** Chains 1–3 continued from Phase A results built on F, and chains 4 and 5 ran every step on F′. All 10 T1 and T2 runs passed C1.
- **Strict no-change (S3, T4).** In all seven runs every candidate tree equalled the input tree, publication reported `noChange=true` with no paths, and the target census was unchanged. The published commit is sometimes the input commit itself (c3-T4, c4-T4) and otherwise a new commit with the identical tree.
- **Continuity.** Each input was built from the previous step's final bundle. Each carried state equalled the previous state file for file and was unchanged afterwards.
- **Questions and corrections.** Every `ask` went to the direct parent or a direct child, and in c3-T2 and c4-T2 Root relayed a question between siblings, as those reviews record. There were no model blockers and no no-progress stops. In c4-S1 one delegate call was refused because a correction needs concrete Findings, and it was retried. In c3-T1 Root ran one correction round inside Implement, for a borrow-limit test.
- **Shared Notes across steps (V11).** Notes from earlier steps stayed readable through the carried state. Managers treated Notes written under an older model as outdated, for example the five-loan limit after M2, as the chain 5 review records.

## Reviews

One independent retained-result review covered each chain segment. Each checked integrity, continuity, limits, T3 link handling, hygiene including the full history of every candidate, and suspicious behaviour.

- Chain 1: four CLEAR WITH NOTES.
- Chain 2: T1 and T2 CLEAR; T3 and T4 CLEAR WITH NOTES.
- Chain 3: T1–T3 CLEAR; T4 CLEAR WITH NOTES.
- Chain 4: T1 and T2 CLEAR; the other five CLEAR WITH NOTES.
- Chain 5: all seven CLEAR. It also checked every final candidate's registry: no dangling `realizes` target, no registered path missing and no unregistered `realization/` file.

The notes changed no outcome:
- A 4 MB `go build` binary (`realization/library`) reached intermediate handoff commits in some runs (c2-T3, c2-T4, c4-S3, c4-T3). Later commits removed it, and no published tree contains it. In other runs the checks caught it as `file.unexplained` or Root declined the handoff.
- Managers refer to the steps as "M3", "M4" and "M5", most likely from the neutral commit subject ("Library M3 model") or the branch name. Phase A used the same convention, and arm B's documents carry the same names, so it reveals no expected case.
- New parameter Definitions such as `shortLoanPeriodDays` and `maxRenewals` are not always linked in the registry. Judging that is a quality matter for the assessor.

## Deviations and corrections

- **Preparation failure (offline).** The first `t1-c1` input failed before any engine call: the step script passed the commit with a trailing carriage return from Windows Python output. It is kept in `failed-preparation\t1-c1-crlf-commit`, and the script was fixed.
- **Lock refusals.** The launches of c2-T3 and c3-T2 first found the shared lock held by a Phase C calibration job. The step script stopped before launching, so no attempt or launch record existed. Both were launched once the lock was free.
- **Handover.** After 10 automatic cross-session messages the app paused further messages to the study, starting with the c3-T3 handover. Later runs were handed over through a shared notes file instead, and the final "done" message went through afterwards.
- **Timing of the F′ record.** B8 records that M-c1-T1 started on F′ at 09:08 UTC, before the protocol had F′ written into it, so the "before the first M run" timing was missed. F′ was the source the coordinator dispatched once #701 merged.

## Limits

- **Image.** M's runtime image Id (`sha256:dd0ddd70…`) differs from arm B's `sha256:1271a363…0ba9`, so section 2's shared-image condition does not hold for Phase B (B8).
- **Source mix.** Chains 1–3 continue from S3 results and state produced on F; chains 4 and 5 ran on F′ only.
- **Order.** Phase B has no counterbalanced pairs. Time effects between the arms are a stated limit (B8).
- **Review granularity.** There was one review per chain segment, not per run, to keep token use within the account's limits.
- **Scope.** One fixture, one provider and model, one machine. Not covered: publication drift and partial transfer, injected faults, account refresh, other providers, release, deployment and owner acceptance.

## Retained evidence

Execution host root: `Documents\Codex\Markitect-V1-Requalification\528ef346\`.
- `controller\`, `build\`, `launch\` (step and wait scripts, launch records).
- `inputs\<name>\` with `input-binding.json`, and `failed-preparation\`.
- `packets\m-<step>-c<chain>\`, each with `attempt-01\` holding every step record, `state\` (Run records, candidate bundles, transcripts, Notes, Apply report), `package-result.json`, `run-summary.json` and `files.sha256`.
- F's refusal and F′'s acceptance of the T3 input: `Documents\Codex\Markitect-V1-Requalification\e11e6cff-rehearsal\inputs\rh-t3-c1\` and `...\528ef346-rehearsal\inputs\rh-t3-c1\`.

Run log and handover: the Phase B arm M notes of the coordinating session.
