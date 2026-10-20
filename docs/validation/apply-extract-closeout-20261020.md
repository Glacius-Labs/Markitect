# Apply and Extract closeout — 20 October 2026

**Scope:** closes the Apply series of [#492](https://github.com/Glacius-Labs/Markitect/issues/492) and the Extract series of [#495](https://github.com/Glacius-Labs/Markitect/issues/495) with its second round. It binds each result of the earlier records to its build, names the requirements that accepted decisions replaced, and reports one bounded confirmation run on the final product build ([owner assignment](../concepts/sources/coordinator-assignment-20261019.md#closeout-and-division-20-october-2026)). Source checks, real runs, Main integration and owner acceptance are reported separately. No owner acceptance is recorded.

## Verdict

- **Apply:** on the final Apply code, one example (the library) and Claude Haiku, the whole sequence held in one bounded run without any bound owner decision. M1 → A1, the owner's one model change, M2 → A2, and a repeated Apply that left A2 unchanged (`no_change`). Drift repair holds on the same Apply code (RA-R6). A model defect that ends the run, budget, deadline, interruption and restart were shown only on earlier builds. Other projects and other agent families are unproved.
- **Extract:** stable as a mechanism on two projects, and it asks only the necessary questions. The extracted models are not adoptable without owner correction, and no round trip on the original repository has been run.

Both series are closed. A closed series records its result; it is not owner acceptance and starts no follow-up work.

## Builds

| Build | What it is | Apply code compared with Main `0c28097c` |
| --- | --- | --- |
| `f15ea723` | Main before the evidence work | older |
| `64f0a713`, `cc11ff5a` | #484, its first commit and Main | older |
| `35c2659c`, `5e16af2b`, `4bedd20b` | #486 before and after review, Main | older |
| `7ed41975` | Main after #506–#510 | older |
| `6e759baa` | #515 (DEC-063 tool) | older |
| `7005a4bd` | #518 (no owner questions) | one sentence older: the parent's initial item before #520 |
| `4cf58479` | #520 (parents lay scaffolding) | same for Apply |
| `1f5cc074` | this record's run: Main `0c28097c` plus the Library model and harness change below | identical product code |

Main `0c28097c` differs from `4cf58479` only in Extract: a separate initial item for processes with their own mandate, and the readable Extract account. The Extract builds are `f15ea723` (old), `cc89d18b` (round 1), `143327ec` (round-2 final) and `cde139c4` (round-2 shipped head; its product code equals Main `0c28097c` except one line of the Extract account).

## Apply: status per obligation

| Obligation | Result | Runs and build | On the final Apply code |
| --- | --- | --- | --- |
| Model change M1 → A1 → M2 → A2 | passed, oracle 12/12 for A1 and A2; one Verify round per Apply, except two for RA-F1's A1 | RA-H7, RA-H8, RA-H9 (Haiku), RA-S1 (Sonnet) on `4cf58479`; RA-F1 on `1f5cc074` | yes |
| Behavior and accounting drift under an unchanged model | passed, only the drifted files changed | RA-R6 on `4cf58479`; earlier RA-R2 to RA-R5 | yes |
| A conforming stand stays unchanged | failed on `7ed41975` (RA-N2 added tests), passed on `7005a4bd` (RA-N3) | RA-F1N on `1f5cc074`: passed | yes |
| A model defect ends the run without publication | passed: contradiction (RA-C2, RA-C3) and boundary gap (RA-Q3) ended within 71 to 177 s | `6e759baa`, `7005a4bd` | no; not repeated after #520 |
| Budget, deadline, interruption and restart | passed: Incomplete, nothing published, work and evidence retained, a new Apply verified | RA-B1, RA-A1 on `4bedd20b`; RA-H1 on `f15ea723`; RA-C1 on `5e16af2b`; RA-H6 on `7005a4bd` | no; the run engine changed afterwards (#515, #518, #520) |
| Protected inputs, enforced ownership, immutable Verify, guarded publication with read-back | passed in every published run | all builds | yes |

**Replaced requirements.** #492's "genuine owner question with an applied answer" no longer applies: [DEC-063](../concepts/register.md#dec-063-the-model-is-authoritative-defects-escalate) and its follow-up (#518) let Apply ask the owner nothing; a contradiction or a boundary gap ends the run with a model-defect report. RA-C1's continuation and RA-Q2's question are therefore failures of the old target, not evidence for the new one. Bound owner decisions ([DEC-053](../concepts/register.md#dec-053-owner-decisions-bind-the-whole-run)) remain only for decisions within the model; the two that the demonstrations bound are boundary decisions and are now part of the Library model (below). [DEC-062](../concepts/register.md#dec-062-start-stand-drift-is-the-owners-first-work) replaced the refusal of committed accounting drift in preparation. The supported restart is a new Apply from the committed repository; journal restore is not required.

**Remaining limits.**
- One example and one agent family: the library, Claude Haiku (and one Sonnet run). Roombook and Readinglog Apply series are deferred (#493, #494).
- Under parallel load on one machine Haiku slowed down enough that RA-H6 missed its deadline. Sequential two-Apply runs took 17 to 18 minutes.
- Verify is a review by the same model family. It caught real defects (an accepted empty request id in RA-H4) but saw a misplaced shared rule in RA-H6 only in A2.
- A late cross-area repair (DEC-060) was expensive with Haiku (RA-C1). Since #520 no final run needed one, so its cost is not measured again.
- The runs use the experiment harness and an explicit run configuration; the `--host` entry and Apply without the harness are deferred (#500).

## Extract: status

| Question | Result | Builds |
| --- | --- | --- |
| Completion | 7 of 7 library runs and 2 of 2 Readinglog runs completed, plus 3 and 1 on the shipped head; earlier builds lost three runs to a registration loop | `143327ec`, `cde139c4` |
| Escalation | the necessary owner question was asked in every seeded and conflict run (4 of 4, then 2 of 2), with no unnecessary question | `143327ec`, `cde139c4` |
| Project and delivery obligations | improved over the old build: "internals are free" (P14, at least partly) and agreement of code, tests and documentation with the model (P09) are stated in every final model; what the tests must cover (P07) is missing in 6 of 7 final library models | `f15ea723` → `143327ec` |
| Incidental detail made binding | mean over-specifications 42 → 22 → 29; internal wiring that mirrors the Go packages remains binding (9 to 14 items per model) | `f15ea723`, `cc89d18b`, `143327ec` |
| Statements contradicting the reference | none on the final build; one on the shipped head (RXY-3 frees the id echo that reference and code fix) | `143327ec`, `cde139c4` |
| Rebuild from the extracted model | passed the oracle 12/12 in 3 of 3 final plain runs | `143327ec` |

The [comparison](library-extract-comparison-20261019.md) and [round 2](extract-round2-20261020.md) hold the details. What this does not establish:

- **Not adoptable without correction.** Every extracted model still needs the owner to remove bound internal wiring, add the test-coverage obligation and decide policy values that stay prose. The round-2 clarification of [DEC-061](../concepts/register.md#dec-061-what-an-extracted-model-contains) is pending the owner's confirmation.
- **No unchanged Apply on the original repository.** Each rebuild projects an empty case-1 target from the adopted model and rebuilds the realization. No run adopted an extracted model next to the original artifacts and showed that Apply then changes nothing. That round trip, the readable adoption subject and the handoff of known deviations to Apply are not implemented or not run (#496, #499).
- **Extract limits:** the deadline path was shown once in the model phase (RXT-1, `cc89d18b`). An unanswered question at the end, an interruption and the budget in the structure phase are not shown (#497). Structure changes between rounds are not implemented (#498).

## Library model: the two former demonstration decisions

The library demonstrations bound two owner decisions with every run (`experiments/recursive-apply-demo/owner-decisions.json`, [source](../concepts/sources/run-wide-owner-decisions-20261018.md)). They settle boundary gaps, which under DEC-063 belong in the model; without them Apply stopped with a model defect (RA-Q3). Their decided meaning now stands in the owning Definitions, without a new decision:

- [library/callable-interface](../design/canonical-model/library-example/m1/model/library/callable-interface.yaml): an otherwise valid request of a known operation that carries an unknown field, or a field whose name differs from a stated one only in case, returns `invalidRequest`, not a refusal. A missing, `null`, wrongly typed or invalid required argument is a refusal. A missing `args`, or one that is not an object, is an invalid envelope.
- [lending/borrow](../design/canonical-model/library-example/m1/model/lending/borrow.yaml): a borrow whose due date would fall after 9999-12-31 is refused with no loan and no state change; a due date is never clamped, wrapped or given a year after 9999.

The decision file is deleted and the harness binds no owner decisions. The run-specific sentence of the first decision ("A recorded review case or finding that requires refused for these cases is contradicted by this decision") addressed the retired review process and is not model content.

## Confirmation run

One bounded run. Its sequence, limits and profile were fixed before the start, and it was not repeated.

| Item | Value |
| --- | --- |
| Product build | `1f5cc074`: the product code of Main `0c28097c` plus the Library model and harness change above. CLI sha256 `04086ed1…`, runtime image `sha256:fd00efd2…`, Go agent and check image `sha256:244528a5…`, Claude Code 2.1.301 |
| Model | Library M1 of that commit, digest `sha256:33e4a0fe…`; M2 is M1 with only `maxOpenLoans` 5 → 3 |
| Agents | Claude Haiku `claude-haiku-5-5`, high effort, for every Execute and Verify turn; all 66 turns report that model. Host profile sha256 `299a41ca…` |
| Run configuration | owner request "Apply the committed Library model.", no `ownerDecisions`, the attempt's own container labels |
| Limits | RA-F1: 30 minutes wall clock including cleanup, engine deadline after 26 minutes; per Apply 20 minutes, 90 agent turns, 4 rounds, 8 minutes per turn. RA-F1N: deadline after 20 minutes, hard stop after 21 |
| Oracle | `ra_assess.sh` after RA-F1, outside the process and its limit, with no feedback to the Agents |

| Step | Outcome | Rounds | Time | Turns (Execute + Verify) | Publication | Oracle |
| --- | --- | --- | --- | --- | --- | --- |
| A1 (M1, empty start) | Verified | 2 | 759 s | 18 + 12 | applied, 28 paths read back | m1 12/12 |
| A2 (M2 on the actual A1) | Verified | 1 | 371 s | 11 + 6 | applied, 5 paths read back | m2 12/12; under m1 10/12 |
| RA-F1N (A2 committed unchanged, M2 again) | Verified | 1 | 359 s | 13 + 6 | `no_change` | tree identical to A2 |

RA-F1 took 1130 s of its 1800 s.

- **A1's second round:** Verify raised two blocking findings under DEC-060. Lending decided copy availability itself instead of using catalog's rule. Round 2 repaired that through catalog's realization. RA-H7 to RA-H9 needed one round, so the round count varies between runs.
- **Effect of the model change:** from M1 to M2 only `loan-policy` changed. A2 changed the limit constant from 5 to 3, the tests and README that state it, and the lending registry shard. Under the m1 profile, A2 fails exactly the two fourth-loan cases (`E04-admission`, `paired-fourth-loan`), so the behavior followed the model.
- **The former decisions as model content:** no run asked about them, bound them or reported a model defect. The oracle's unknown-field and envelope cases passed for A1 and A2. A2 limits due dates to 9999-12-31 in `realization/lending/date.go`; the oracle has no case for it.
- **No-change:** every area took part in Execute, Verify was green in its first round, and the publication was `no_change`. The working tree, the artifacts and the registry stayed byte-identical to A2: no changed entries and no differing file. Verify recorded three advisory findings: a status rule realized in `library` instead of `members`, and the registry attribution of the shared date rule. They do not block, and no area changed for them.
- **Protected inputs, publication and read-back:** the model, dependencies, lock and execution controls stayed unchanged apart from the owner's one policy change. Both publications touched only `realization/` and `.markitect/registry/`, and every path read back in its published state. No container or work volume of the run remained.
- **Human intervention:** none during the runs. The scripted owner made the three commits: take A1, change the limit, take A2. A first detached start of the driver ended before binding and created nothing; the same driver was then started unchanged.
- **Not covered by this run:** a model defect, budget, deadline and interruption on this build.

## Evidence kinds, kept apart

- **Source checks:** the bounded language probe of the library example passed with unchanged link counts (22 local Definitions, 94 body links, 60 cross-Package links; [recorded result](../design/canonical-model/library-example/verification.json), which also refreshes the hash of the language contract changed on 17 October). `markitect check` passed on the projected M1 target and on this repository. No Go code changed.
- **Real runs:** RA-F1 and RA-F1N above; the earlier runs in the linked records. Run directories, transcripts and retained volumes stay outside the repository under `~/mk-ra` in WSL.
- **Main integration:** this record's PR.
- **Owner acceptance:** none recorded.

## Next step

Recommendation, not started. After the owner's process review, run the prepared trial cases once, as one small fixed series on the final build with the same Haiku profile. The series should include the model-defect stop and one interruption with a fresh restart, which closes the two gaps on the final Apply code. It should also include the first project beyond the library. Select process changes only from failures of that series. For Extract, the owner first confirms or corrects the DEC-061 clarification; the next Extract evidence would be the round trip on an original repository (#499).
