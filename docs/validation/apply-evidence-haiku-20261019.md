# Apply evidence on the library with Haiku — 19 October 2026

**Scope:** the bounded Apply evidence of [#492](https://github.com/Glacius-Labs/Markitect/issues/492) on the library example. It covers a model change, drift repair under an unchanged model, an owner question, budget, deadline and interruption with a restart, and a conforming stand. Source checks, these real runs, Main integration and owner acceptance are reported separately. No owner acceptance is recorded.

## Verdict

Every obligation of #492 was exercised in a real run; one of them, the no-change stand, only partly held. Every outcome was honest:
- Nothing was published unless Verified, and published stands were read back.
- The model and control inputs stayed unchanged; publications changed only artifacts and registry shards.
- Incomplete runs kept their work and evidence, and a new Apply from the committed repository always succeeded.

Two product defects surfaced and are fixed on Main (#484, #486). Three findings remain open and are stated below.

## Setup

| Item | Value |
| --- | --- |
| Agents | Haiku `claude-haiku-5-5`, high effort, for every Execute and Verify turn (owner direction: Haiku first) |
| Example | Library M1 (and M2 for the model change) from `docs/design/canonical-model/library-example`, layout root `realization` with the `cmd/library` exception, required checks `go build` and `go test` |
| Budget per Apply | 20 minutes, 90 agent turns, 4 rounds (`experiments/recursive-apply-demo/library-execution.yaml`); RA-B1 used 14 turns |
| Bound owner decisions | the two decisions of the earlier demonstrations, except in RA-Q1 |
| Harness | a copy of `experiments/recursive-apply-demo`, outside the repository: own container store label, Haiku, and a scenario script for drift, question, contradiction, budget, abort and no-change. The copy and all run directories are kept outside the repository. |
| Oracle | `experiments/runtime-baseline/assess.py` after each Apply, outside the process, no feedback to the Agents |

## Runs

| Run | Build | Scenario | Outcome | Rounds | Engine time | Oracle |
| --- | --- | --- | --- | --- | --- | --- |
| RA-H1 | `f15ea723` | M1 → A1 | Incomplete: deadline in round 2 | 2 | 1141 s | not published |
| RA-H2 | `64f0a713` | M1 → A1 → M2 → A2 | Verified, applied (both) | 1 / 1 | 584 s + 374 s | m1 12/12, m2 12/12, A2 as m1 10/12 |
| RA-R1 | `cc11ff5a` | drift: limit bug + unregistered committed file | refused in preparation (defect, see below) | – | – | – |
| RA-R2 | `cc11ff5a` | drift: limit bug | Verified, applied | 1 | 361 s | 12/12 |
| RA-R3 | `35c2659c` | drift: limit bug + unregistered committed file | Verified, applied | 1 | 389 s | 12/12 |
| RA-Q1 | `35c2659c` | M1 → A1 without bound owner decisions | Verified, applied; no question asked | 2 | 817 s | 12/12 |
| RA-C1 | `5e16af2b` | model contradiction (prose "four", spec 5) | Incomplete: deadline; question asked and answered | 1 | 1126 s | not published |
| RA-B1 | `4bedd20b` | 14 agent turns, then a new Apply | Incomplete (budget), then Verified, applied | 1, 1 | 774 s, 626 s | 12/12 after restart |
| RA-A1 | `4bedd20b` | SIGTERM after 140 s, then a new Apply | Incomplete (cancelled), then Verified, applied | 0, 1 | 118 s, 811 s | 12/12 after restart |
| RA-N2 | `7ed41975` | the verified stand of RA-A1, committed unchanged | Verified, applied: one added test in each of two test files | 1 | 349 s | 12/12 (see finding 4) |

Builds: `f15ea723` is Main before this work. `64f0a713` is the first commit of #484, and `cc11ff5a` is Main with #484. `35c2659c` and `5e16af2b` are commits of #486, before and after its review fixes. `4bedd20b` is Main with #486. `7ed41975` is the final integrated Main. Between the two, Main merged #481–#483, #485 and #506–#510, among them #483 (the Go example image), #506 (image observation once per process) and #507 (Extract fields and shared helpers).

## The obligations of #492

- **Model change:** passed (RA-H2). A2 was realized on the actual A1 after only `maxOpenLoans` changed. The oracle shows the behavior changed with the model.
- **Behavior drift under an unchanged model:** passed (RA-R2, RA-R3). The owner repaired the limit bug in Execute.
- **Repairable committed accounting drift:** passed after a fix (RA-R3). RA-R1 refused it in preparation, so the owner decided DEC-062 (#486). In RA-R3 members removed the stray file.
- **Changes to already conforming areas:** justified in RA-R3. Reminders compared dates as instants instead of calendar days, and Root documented the wire behavior the model requires. In RA-N2 the change is a finding (see 3).
- **Owner question with an applied answer:** partly passed (RA-C1).
  - Root found the contradiction 68 s after the run started, asked, received the answer 3 s later and applied it; the retained work has `MaxOpenLoans = 5`.
  - The run then ran into the deadline (see finding 2), so nothing was published.
  - The owner since decided [DEC-063](../concepts/register.md#dec-063-the-model-is-authoritative-defects-escalate): a model contradiction ends the run with a model-defect report instead of a run-only answer. RA-C1's continuation is therefore no longer the target behavior.
  - RA-Q1 asked nothing: the Managers decided the boundary edge cases themselves. The oracle was green, but under DEC-063 these are escalations.
- **Budget, deadline, interruption and restart:** passed.
  - **Budget (RA-B1):** the round ended without going green, and no further Execute turn fit beside a Verify round.
  - **Deadline (RA-H1, RA-C1).**
  - **SIGTERM (RA-A1):** the run ended Incomplete within 2 s, with its report, evidence and removed homes.
  - **Every Incomplete run:** published nothing, left the repository unchanged and retained the work volume, which is named in the report.
  - **Restart:** the supported restart is a new Apply from the committed repository; there is no journal restore. The lock was released, and the new Apply verified.
- **Protected inputs, ownership, immutable Verify, guarded publication:** passed.
  - Enforced write plans applied in every run, and accounting ran on every candidate.
  - Every published stand returned an `applied` receipt with read-back.
  - The model and control inputs under `.markitect` were unchanged after every publication; only the Managers' registry shards changed.
- **Conforming no-change stand:** not achieved on the final build (RA-N2, finding 3).

## Findings and disposition

1. **Fixed (#484):** RA-H1 failed because of three problems.
   - Root took one turn per child report, 12 of 25 Execute turns.
   - Root asked an unnecessary owner question, because the Brief did not name the grandchild area.
   - An unparsable registry shard cost a round.
   - The tools also refused a second `finish` or `assess` call.

   #484 changed four things: a parent turn per quiet subtree, children's descendants in the Brief, the `check` tool for Apply, and last-call-wins tools. RA-H2 then verified both Applies in one round each.
2. **Open: a cross-area repair (DEC-060) is expensive with Haiku.** In RA-Q1, RA-C1 and the budget run of RA-B1, Verify blocked shared rules that were realized outside their owner. In RA-C1 the repair moved a date module between areas: Root took 18 turns, 78 messages were exchanged, and the run missed the 20-minute deadline shortly before its second Verify round. The run stayed honest, but no Verified outcome was reached within the budget.
3. **Open: no idempotent no-change.** On the conforming stand of RA-N2, the Managers added two tests of modeled behavior and published them, instead of concluding that no change was needed. RA-R3 also changed areas beyond the drift. Each Apply on a conforming stand can therefore produce changes.
4. **Outside Apply, being fixed by the Coordinator stream:** on the final Go example image (`7ed41975`), the oracle harness fails to build with "read-only file system" for its work directory. The cause is `GOTMPDIR=/home/agent` in the image (#483). The oracle starts the image with a read-only root and no writable home. The oracle passed 12/12 on RA-N2 with the Go image of `4bedd20b`.
5. **Fixed (#486, DEC-062):** an unregistered committed file used to refuse preparation with misleading diagnostics. It is now the owner's first work under a declared layout.
6. **Decided, not implemented:** DEC-063 (model defects escalate and end the run) follows this report.

## Evidence kinds, kept apart

- **Source checks:** `go vet`, `go test ./internal/...`, the architecture gate, `markitect check` and the documentation links were green for #484, #486 and #509, each after an independent review. Hosted CI was not run; it is manual during development.
- **Real runs:** the table above, one model, one example. The run directories, transcripts and retained volumes are outside the repository.
- **Main integration:** #484, #486 and #509 (DEC-063, documentation only) are merged. The final Main `7ed41975` ran only the no-change scenario. The other scenarios ran on the earlier builds named above; the differences are listed under Builds.
- **Owner acceptance:** none recorded.
