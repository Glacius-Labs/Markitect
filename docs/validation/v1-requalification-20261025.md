# V1 native requalification — 24 and 25 October 2026

Step 4 of the [delivery plan](../implementation-plan.md#steps): the native requalification of V1 with Codex on the frozen final source. The same runs are arm M of Phase A of the [comparison protocol](v1-comparison-protocol-20261024.md) ([DEC-072](../concepts/register.md#dec-072-stable-v1-justified-by-data)). The coordinator dispatched them under the [standing evidence authorization](../concepts/sources/v1-source-implementation-approval-20261023.md#subsequent-standing-evidence-authorization). Every planned run had exactly one attempt; failures stay recorded below. This record covers arm M only. The comparison with arm B and its analysis belong to the study's own results document.

## Result

- All 12 counted Applies passed protocol C1, which is the technical pass. Each was clear in an independent retained-result review. Each final candidate passed the study's oracle of record: 12 of 12 cases with the opposite-profile discrimination, 6 of 6 hidden probes, and its own build and tests.
- Three longitudinal chains each ran S1 fresh M1, then S2 with the M1 → M2 loan limit 5 → 3, then S3 no-change on the chain's own result. All 9 passed, and strict no-change held in all three S3 runs.
- Three S4 runs started from one fixed seeded realization. In all three the defect was found and repaired, and C6 was met under the final criterion.
- **Not shown natively:** the public Verify Reject → fresh Implement → fresh Verify → publication branch (P5). No S4 run reached it, because Assessment caught the seeded defect every time (see [P5](#p5-public-verify-reject-branch)).
- **Not covered:** [PR #696](https://github.com/Glacius-Labs/Markitect/pull/696), the refined idle stop, merged after F, so this series qualifies F's rule, not #696's.
- **P4**, the obvious-seed ordinary repair run outside the comparison: its negative oracle gate failed as predeclared and passed a post-hoc corrected criterion ([P4](#p4-obvious-seed-outside-the-comparison)). The Apply then passed C1 and published a one-line repair, which the positive oracle confirmed with 12 of 12 cases.

## Binding

| Item | Value |
| --- | --- |
| Source F | `e11e6cff883ba16da2cddab8d54c0367cae8a5d8`, tree `f10d34113e499b0e4904729fca93470fa138cb74`, frozen by the coordinator after C1. Fresh `clone --no-local`, detached and clean; HEAD, tree and status were checked before and after every run |
| Product binary | Built once from F with `go build -trimpath -buildvcs=true`, `CGO_ENABLED=0` and the pinned go1.27.1 toolchain. SHA-256 `c0540b7374f2a78d938f5f81048e2c3b3cbf2f54fa060c61859356c66fd2b1a6`, `vcs.modified=false`, productBuild `0.14.1+e11e6cff…`. Every packet ran a byte-identical copy |
| Images | Built from F's `packaging/runtime`. Base `sha256:1f15c25ea417a5445c43f71f5e1581bbdd87fbe9c964b94b9ee48c0c31d5cab7`; Go (agent and check profile) `sha256:1271a363ef2dfe861fe4fe1842d8434007f3d58072458837e7117b50bf600ba9`. `runtime.env` in both holds exactly `codex=codex-cli 0.162.0`, `codexSchema=sha256:816cef0e7864c26f425a7f709358f66972a9aa44078c69c6ad85943ca95aaf5e` and `agentd=` F. The smoke test (uid 1000, read-only root, no network) showed go1.27.1 and codex-cli 0.162.0 |
| Provider | Codex App Server 0.162.0, `gpt-6-sol`, effort `high`, dedicated normal-account login. The controller only hashed the login file, and the hash was equal before and after every run |
| Configuration | Exact copies of [`examples/runtime-profile.example.yaml`](../../examples/runtime-profile.example.yaml) and [`examples/run.example.json`](../../examples/run.example.json), with only their placeholders filled and the documented `checkProfiles.go` → `library-tests` mapping |
| Limits | Apply deadline 2,700 s; 128 shared messages; 600 s per message; Run cleanup 45 s; persistence 30 s; `maxManagers` 32; no `workDeadline`. Outer limits: product process 3,000 s, preflight 180 s, post-exit cleanup and observation 180 s, packet 3,300 s, engine metadata calls 20 s, explicit cleanup calls 8 s |
| Fixture | Library M1 from F's `testdata` (model, lock, vocabulary, Core 1.1.0), plus `execution.yaml` and an empty registry as [`scripts/run-library-apply.ps1`](../../scripts/run-library-apply.ps1) writes them. M1 digest `sha256:be14ca732b34b422545972b210cbc96caad17f32b9eaa2438d6d64d55efb0d9e`; M2 digest `sha256:7b0b0e3d419b68a7a89a381c4288b35353c3645382a565c6eda8114e0fbf67b0`. Six Managers: Root, `catalog`, `lending`, `lending/reminders`, `library`, `members` |
| State | Fresh for S1 and every S4 run. S2 and S3 started from a hash-verified copy of the previous step's state directory, as the protocol requires, so Shared Notes carried across commands |
| Controller | A new Python standard-library controller, `requalify.py`, SHA-256 `32570e0de21c2c77915d485d1ad47e22c6294fde5a0f27411ae6268cd4a27d16`; every packet bound and copied it. 27 offline self-tests cover exact-string deadlines, a `files.sha256` that never lists itself, a real Ctrl-Break, and report consumers tested on real retained reports. It had two independent reviews before first use. Runs were launched detached and held the shared `NATIVE-RUN.lock`; arm B and the study's assessor took the same lock |

## Preparation

The first build attempt (P0 attempt 01, 24.1 s) failed on a defect in the controller. It expected `codex=0.162.0`, while the image correctly records `codex=codex-cli 0.162.0`, the exact value the product's metadata parser requires. No provider, login or counted run was involved and nothing was left behind. The attempt is retained unchanged; its `build-result.json` has SHA-256 `966948e044bc58a2b4a3ffd40c4fa4447f5e9dd0c1d92ebeb97d4ae2609af08d`. After the controller fix, which was tested against that file, attempt 02 built in 39.0 s (`build-result.json` `52467bf497a8bd3e10972762bd9a6d68c48e0b9677dac496fc585fe3611d381b`).

## Runs

Order followed the protocol: scenario by scenario, M first in repetitions 1 and 3, B first in repetition 2. Seconds are Apply process time. Messages are the sum and the Assessment/Implement/Verify split. Every Run had all six Managers in fresh native Sessions (18 distinct threads per Apply), six handoffs and 14 closed resources. No run left residue or needed a controller removal, and every final candidate bundle reopened exactly.

| Packet | Apply | C1 / scenario | s | Messages | Paths | Final commit / tree | Tokens | `package-result.json` |
| --- | --- | --- | ---: | --- | ---: | --- | ---: | --- |
| m-s1-c1 | `9476949f6f82e7d29302d51ecbc017a6` | pass / met | 591 | 21 (6/9/6) | 21 | `ae37be37` / `101c5840` | 6,954,140 | `78b6f2bb77dd` |
| m-s2-c1 | `a67a4b154d868bb43cf5a9c3bf717a5f` | pass / met | 501 | 18 (6/6/6) | 2 | `f0a94c4d` / `445ade39` | 6,269,431 | `ced444da68a0` |
| m-s3-c1 | `7e6cd86c69436c347531ff48797e2923` | pass / strict met | 536 | 20 (6/8/6) | 0 | `4136b583` / `445ade39` | 7,586,186 | `b34d42effaaa` |
| m-s4-r1 | `9fddae38a4b0e12e7815a807fee83ac7` | pass / met | 526 | 33 (6/21/6) | 5 | `91db7011` / `f99b2d90` | 8,966,223 | `1c30b5f2b7fd` |
| m-s1-c2 | `1cbc0d59c46b49717af696d1d513e470` | pass / met | 612 | 21 (6/9/6) | 21 | `eaa753b6` / `aacc1abf` | 8,001,934 | `6e12bfeeadb2` |
| m-s2-c2 | `e5ad7fc24785eebfb2862b836d349314` | pass / met | 562 | 23 (8/9/6) | 4 | `f5a66799` / `8c7f564e` | 6,781,200 | `28e55372f2d8` |
| m-s3-c2 | `98b498ddcb008fc660dfab134ae7dbcf` | pass / strict met | 565 | 18 (6/6/6) | 0 | `1e6eb575` / `8c7f564e` | 7,071,151 | `4bf7befc0e0e` |
| m-s4-r2 | `d2a829382a4aa615256d94b5027f95a8` | pass / met | 564 | 22 (8/8/6) | 5 | `2e8a681c` / `7034201e` | 7,622,473 | `b306a29ade8b` |
| m-s1-c3 | `cd42985dbf9e050b2d4f1c658a9a1913` | pass / met | 826 | 22 (6/10/6) | 19 | `058d1261` / `b06851dd` | 8,565,779 | `ad5c49c7e2c1` |
| m-s2-c3 | `767c0a0725788555b95341090916a57f` | pass / met | 578 | 19 (6/7/6) | 4 | `f4561987` / `5a5abcf9` | 7,068,228 | `6d458d4fb705` |
| m-s3-c3 | `bdbf92fddf72757c6e2239012e51ff00` | pass / strict met | 558 | 19 (6/6/7) | 0 | `f4561987` / `5a5abcf9` | 6,710,569 | `9b868d4957b4` |
| m-s4-r3 | `b095ce121104f9aa0142decc0491a72c` | pass / met | 585 | 22 (8/8/6) | 4 | `ef1d1327` / `a682cc2d` | 7,482,404 | `a3829153c7c1` |

Totals: 7,004 s of Apply time, 258 messages and 89,079,718 provider-reported tokens (`thread/tokenUsage/updated`, last total per thread). The longest Apply took 826 s of its 2,700 s deadline (m-s1-c3); the most messages in one Apply were 33 of 128 (m-s4-r1).

Oracle of record: the study's assessor at R = `74b7d7df` with the unchanged [container harness](../../tools/library-oracle/README.md#container-harness) at F. For every final candidate above it recorded `cases_passed` 12 of 12, discrimination true, probes passed, own build and `go test` passed, and `mechanical_pass` true. The assessments are in `Documents\Codex\Markitect-V1-Comparison\e11e6cff883b\assess\M-*\assessment.json`.

## What the runs show

- **Fresh realization and change (V12, scenarios 26.1 and 26.2):** three independent M1 realizations passed the m1 oracle. In each chain, M2 changed only what the new limit requires: the constant, its boundary tests and the README sentence, with one small optional test edit in chain 2 and chain 3. The lower limit applies to new borrowing only; return is untouched and no existing loan is deleted.
- **Strict no-change (scenario 26.3):** in all three S3 runs every candidate tree equalled the input tree, publication reported `noChange=true` with no paths, and the target census was unchanged. The reviews confirmed that Implement read, tested and built before it submitted unchanged. In chains 1 and 2 the final candidate is a new commit with the identical tree; in chain 3 it is the input commit itself.
- **Seeded defect (V13):** the S4 seed agrees across code, two tests and the README and contradicts `lending/reminders/assess-overdue`. All three runs found and repaired it. In every run Assessment found it first: `lending/reminders` and other Managers flagged it against the model, and Root passed those Findings into Implement, which repaired code, tests and documentation. Verify then approved, and the outside oracle confirmed every repair.
- **Child diagnostics at the parent (V05, D11):** parents rejected faulty child handoffs and delegated corrections in most runs, for three kinds of fault:
  - code that did not compile;
  - a stray 4 MB `go build` binary flagged as unexplained by the handoff check;
  - a wire-format bug: `id:null` echoed as an empty string.

  In M-S1 c3 the host refused a parent's approval of a structurally invalid child candidate. The parent then rejected it and the child resubmitted. The [A1 contract proof](a1-native-contract-proof-20261024.md) had observed that refusal in controlled sessions only.
- **Questions (V03, D10):** every question in the reviewed transcripts went to the direct parent or a direct child.
- **Shared Notes across commands (V11):** chain 1 left no Note to carry. In chains 2 and 3, S1's Implement Note was read by 13–14 of 18 Managers in S2, and by 11–15 Managers in S3 together with S2's Notes, through the carried state.
- **Durable candidates and readback (V08, V15 in part):** every final candidate bundle reopened to its commit and tree in a fresh bare repository. Publication checked the exact final commit and read it back.
- **Harness and login (V10):** native instructions only, the normal account login, no API-key route.

## P5: public Verify-Reject branch

The S4 seed was designed so that Implement would preserve a defect that looked consistent while Verify rejected it. That assumption failed: Assessment reads the same model and caught the contradiction in all three runs. The branch was therefore never reached, and no run published through it. The coordinator decided to record P5 as not exercised natively in this series, with no extra attempt. This is a positive finding about Assessment. The public correction branch still rests on controlled actual-Git source evidence and on the earlier native component chain on older source ([qualification record](v1-qualification-20261024.md)). P5 stays an open acceptance item: later comparison phases may exercise it naturally; otherwise the owner decides at acceptance whether the existing evidence suffices.

## P4: obvious seed, outside the comparison

P4 was planned as one ordinary-repair run with a one-line seed: `loan.DueDate < assessmentDate` became `<=` in `overdue.go`, tests untouched, as a single neutral root commit `72b36808` (patch SHA-256 `f212e8ec2fa81d043e277444bfad4ee41b8ef82fbb5b063c2fd2a8cb6588bd13`). The predeclared negative gate allowed oracle failures only in E10-overdue, plus E04-admission if that case assesses due-today loans.

The negative gate, `oracle\p4-negative\result.json` with SHA-256 `1be019afccf759c15c996dba52186d00d120985a9b22b922663aa5d1121a0968`:

| | Predeclared | Actual |
| --- | --- | --- |
| Failing cases | E10-overdue, and E04-admission only if it assesses due-today loans | 8 of 12: E02-copies, E03-successful-loan, E04-admission, E05-suspension, E10-overdue, E11-invalid-input, E12-callable, paired-fourth-loan |
| Failing assertion | — | One in each case, always `response.result.notices: expected IDs [], got […]`. The request is `assessOverdue` at 2026-10-15, and every returned notice has due date 2026-10-15 |
| Other assertions | — | None failed before each failing case stopped at its first failure; later steps of those cases did not run. E01, E06, E08 and E09 passed; discrimination was confirmed; no residue remained |
| Gate as written | Pass | **Failed**: the allowed set was predicted too narrowly |

The cause: in many cases the oracle's state snapshots assess overdue on the loans' due date. The boundary seed therefore surfaces in every case that takes such a snapshot, and each case stops at its first failure.

**Corrected criterion, post hoc** (decided by the coordinator on 25 October 2026, after the gate result was observed; written into the P4 packet before the P4 Apply as `gate-criterion-posthoc.json`, SHA-256 `7d9618797e896bde161d4ae705d181c8615abec7950c218fc08236c4ea775a98`): the negative gate passes when at least one case fails and every failure is an overdue-notices assertion caused by a due-today loan. That means the failing request is `assessOverdue`, the expected notice set is empty, and every returned notice's due date equals the assessment date; discrimination must also be confirmed and the run residue-free.
- Reason: the gate exists to show that the seed is detected and breaks nothing else. Only the predicted case list was wrong.
- The mechanical evaluation, `gate-evaluation-posthoc.json` (SHA-256 `c08aa784a66e8129fcca946b5d80711fac5c7da16a247ba0b8c3d0dc94603174`), shows the gate passes the corrected criterion.
- P4 stays outside the comparison and counts toward no comparison statistic. It had one attempt.

The P4 Apply, `87f523e1fdef70765b8f58cebf5c1844`, passed C1:

| Item | Value |
| --- | --- |
| Run | 600 s; 25 messages (8/11/6); 18 distinct threads; every Run closed |
| Sequence | Assessment approved with 12 Findings; Implement submitted; Verify approved; publication of the verified candidate |
| Published | 5 paths: `realization/lending/reminders/overdue.go`, the catalog, lending and members tests, and the members registry shard |
| Final candidate | `eb7847432bad52b55820f923e88beb34c53e70f0` / tree `ca58bbfcea478582ae9c6835a095a47865d68b1f`; the bundle (SHA-256 `d55a908c…`) reopened exactly |
| Tokens | 8,356,330 |
| Evidence | `package-result.json` `ab86d4b28f28…`, `run-summary.json` `656ab7f3caa3…`, `files.sha256` `b9eeb732f676…`; no residue |

The positive oracle on the final candidate, `oracle\p4-positive\result.json` (SHA-256 `d8c631463fb3654588e24aac436fb7f9378b1ce514eac596b8f202b50aef0b63`), confirmed the repair: m2 12 of 12 with discrimination, the evaluator at F, residue-free.

The independent retained-result review found P4 clear, with notes:
- In Assessment, four Managers (Root, `lending`, `lending/reminders`, `library`) named the defect. They read `overdue.go` and ran `go test`, where the existing test `TestAssessOverdueCurrentOpenStrictBeforeAndReadOnly` fails on the seeded input.
- Implement reverted the one line (`<=` back to `<`, the exact inverse of the seed). It also added three cross-package regression tests and an accurate registry `why` text.
- Verify approved.
- No seed wording reached any writer input, no Note was written, and no residue or credential was found.

**Limit:** the seed left its test in place, so P4 shows detection and repair when the project's own test signals the defect. It does not show detection of a defect that no test catches; the coherent S4 seed covers that case.

## Deviations and corrections

- **C6 criterion for S4:**
  - The first list of seeded lines (16 lines, SHA-256 `9a8c9e54…`) included five neutral helper lines that a correct repair keeps.
  - After M-S4 r1 was observed, the coordinator changed the criterion post hoc to the blind reviewer's 10 defect-bearing lines (`37d14c6a…`) for every S4 run of both arms. The step-4 operator's own 11-line list (`29f230a3…`) is kept beside it.
  - All three M runs score 0/10 and 0/11 under the new lists. They would fail only under the original 16-line list, at 5/16, 3/16 and 3/16.
- **Misstatement corrected:** the step-4 operator first reported M-S3 c1's final commit as the start commit. It is a new commit with the identical tree. The study was told the correction.
- **Session interruption:** the operating session ended at about 22:58Z while launching M-S3 c2. The launch had not taken effect: no launch record, no attempt and the lock free. The run was relaunched at 05:06Z as its only attempt. UTC clocks were cross-checked, and the access token was still valid.
- **Unavailable safety classifier:** one independent review ran while the safety classifier was unavailable. Its read-only claim was verified afterwards: both retained attempts it read still match their manifests exactly.

## Limits

- One fixture, one provider and model, one machine. N = 3 per scenario only rules out low reliability: 3 of 3 has a one-sided 95 % lower bound of 37 % per scenario, and 12 of 12 one of 78 % for this suite.
- [PR #696](https://github.com/Glacius-Labs/Markitect/pull/696) (refined idle stop) merged after F. No run ended in a no-progress stop, and one refused `await` occurred, the case where F's rule only warned. #696 and later product fixes get one combined native requalification on the final frozen source.
- The shared S4 base is M's own S2 candidate (M-S2 c1); the study records the same limit for arm B.
- Agent behaviour notes, none of which changed an outcome:
  - Two Managers in M-S3 c3 claimed test or build success before their command returned. Root's own checks passed.
  - `go build` binaries left in working copies recur. The checks catch them every time, but intermediate candidate history keeps the bytes.
- Not covered: publication drift and partial transfer (V15 in full), injected faults (V16), account refresh, other providers, live data migration, human attention, release, deployment and owner acceptance.

## Retained evidence

Execution host root: `Documents\Codex\Markitect-V1-Requalification\e11e6cff\`.
- `controller\`: `requalify.py`, `test_requalify.py`, `launch.py`.
- `build\` and `failed-attempts\p0-build-attempt-01\`.
- `inputs\<name>\`: committed inputs with `input-binding.json`.
- `packets\<name>\`: `profile.json`, the binary and controller copies, the runtime profile and run-config template, `state-from-manifest.json` where state was carried, and `attempt-01\` with every step record, `run.json`, `fixture\`, `state\` (Run records, candidate bundles, transcripts, Notes, Apply report), `start.bundle`, `package-result.json`, `run-summary.json` and `files.sha256`, written last and excluding only itself.
- `seeds\` and `handover\`: approved patches, seeded-input bundles and the seeded-line lists. These are never writer inputs.
- `oracle\p4-negative\` and `oracle\p4-positive\`: the P4 gate and the oracle on P4's final candidate.

Run log and decisions: the step-4 notes of the coordinating session.
